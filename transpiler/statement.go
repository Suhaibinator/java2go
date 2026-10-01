package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

// needsExplicitPrimitiveType reports whether a Java primitive local declaration
// must be emitted with an explicit Go type rather than `:=` inference. It covers
// the primitives whose Go type is narrower than the type an untyped constant
// would infer: int->int32, long->int64, short->int16, byte->int8, char->rune,
// float->float32. Double is conditionally pinned below when an integral
// initializer would infer Go int. Boolean already infers to bool. The original
// Java type may carry array brackets or qualifiers; only the bare primitive name
// is matched.
func needsExplicitPrimitiveType(originalType string) bool {
	base, _ := parseJavaTypeString(originalType)
	switch strings.TrimSpace(base) {
	case "int", "long", "short", "byte", "char", "float", "double":
		return true
	}
	return false
}

// isVarKeywordType reports whether a local declaration used Java's `var` type
// inference (so its element type must be inferred from the initializer).
func isVarKeywordType(originalType string) bool {
	t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(originalType), "final"))
	return strings.TrimSpace(t) == "var"
}

// constructorFuncName returns the generated Go constructor function name for a
// class scope (e.g. "newAnimal" for a package-private class, "NewFoo" for a
// public one), or "" if the scope declares no constructor. The name is taken
// from the constructor's symbol definition, which is exactly what the emitted
// constructor FuncDecl uses, so super(...) calls resolve to the right function.
func constructorFuncName(scope *symbol.ClassScope) string {
	if scope == nil {
		return ""
	}
	for _, method := range scope.Methods {
		if method != nil && method.Constructor && method.Name != "" {
			return method.Name
		}
	}
	return ""
}

// isStmtListNode reports whether a node type is one that ParseNode lowers into a
// list of statements (rather than a single statement). These are the constructs
// that must be expanded inline when filling a block body.
func isStmtListNode(nodeType string) bool {
	switch nodeType {
	case "try_statement", "try_with_resources_statement", "synchronized_statement":
		return true
	}
	return false
}

func ParseStmt(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	if stmt := TryParseStmt(node, source, ctx); stmt != nil {
		return stmt
	}

	diag := reportUnsupported("statement", node, source, ctx)
	// Emit a placeholder statement that still compiles, so the rest of the file
	// can be converted. The panic call preserves the diagnostic at runtime.
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.Ident{Name: "panic"},
			Args: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", strings.TrimPrefix(unsupportedComment(diag), "// "))},
			},
		},
	}
}

// Java loop bodies accept any statement; Go requires a block. Preserve a
// source block's scope and wrap single or expanded statements without changing
// the loop node targeted by break and continue.
func parseLoopBody(node *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	if isStmtListNode(node.Type()) {
		return &ast.BlockStmt{List: ParseNode(node, source, ctx).([]ast.Stmt)}
	}
	statement := ParseStmt(node, source, ctx)
	if block, ok := statement.(*ast.BlockStmt); ok {
		return block
	}
	return &ast.BlockStmt{List: []ast.Stmt{statement}}
}

// parseStatementBlock renders one Java block while optionally omitting an
// already-structured source statement. Constructor lowering uses the omission
// for its leading this(...)/super(...) invocation: that invocation controls the
// allocation/initialization phase and must not first be flattened into an
// ordinary Go statement (or parsed twice, which could duplicate hoisted
// declarations in its arguments).
func parseStatementBlock(node, omitted *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	// Hoisted Go declarations share the file, but Java type names introduced
	// by this block must not escape into its enclosing or sibling blocks.
	ctx.localClasses = copyLocalClassBindings(ctx.localClasses)
	// Row-loop LICM plans are discovered while recursively rendering an inner
	// loop, then consumed as the recursion unwinds through these lexical blocks.
	// Initialize the shared map before descending so cloned contexts observe
	// registrations made by their children.
	if ctx.affineArrayRowHoists == nil {
		ctx.affineArrayRowHoists = make(map[affineArrayCallSiteKey][]*affineArrayRowHoist)
	}

	body := &ast.BlockStmt{}
	for _, line := range nodeutil.NamedChildrenOf(node) {
		if line.Type() == "comment" || line.Type() == "line_comment" || line.Type() == "block_comment" {
			continue
		}
		if omitted != nil && line.Type() == omitted.Type() && line.StartByte() == omitted.StartByte() && line.EndByte() == omitted.EndByte() {
			continue
		}
		if stmt := TryParseStmt(line, source, ctx); stmt != nil {
			// A hoisted local class leaves behind an implicit empty statement;
			// skip it so no stray semicolon is emitted.
			if empty, ok := stmt.(*ast.EmptyStmt); ok && empty.Implicit {
				continue
			}
			body.List = append(body.List, applyAffineArrayRowHoists(line, stmt, ctx)...)
			if line.Type() == "local_variable_declaration" {
				body.List = append(body.List, unusedLocalDiscardStatements(stmt, node, source)...)
			}
		} else if isStmtListNode(line.Type()) {
			// Try and synchronized statements are lowered into a list of statements.
			body.List = append(body.List, ParseNode(line, source, ctx).([]ast.Stmt)...)
		} else {
			// Anything else (including unsupported constructs) is converted via
			// ParseStmt, which emits an UNSUPPORTED placeholder rather than crashing.
			body.List = append(body.List, ParseStmt(line, source, ctx))
		}
	}
	return body
}

func parseReturnValue(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if node != nil && node.Type() == "null_literal" && strings.TrimSpace(ctx.expectedType) != "" {
		if isBuiltinJavaString(ctx.expectedType, ctx) {
			return javaStringNullExpr(ctx)
		}
		return zeroValueForType(javaTypeStringToGoTypeExpr(ctx.expectedType, inScopeTypeParameters(ctx), ctx))
	}
	value := ParseExpr(node, source, ctx)
	value = coerceArgumentToExpectedType(value, node, ctx.expectedType, ctx, source)
	return requireNullableValueBackedExpression(value, node, ctx.expectedType, ctx, source)
}

// replayedMethodReturnStmt builds the real function return after a generated
// closure has recorded Java abrupt completion. Explicit Java constructors are
// emitted as Go functions returning the newly allocated receiver, so their
// source-level `return;` must return that receiver rather than a bare Go return.
func replayedMethodReturnStmt(ctx Ctx, hasValue bool, valueName string) *ast.ReturnStmt {
	if ctx.localScope != nil && ctx.localScope.Constructor {
		return &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}}}
	}
	result := &ast.ReturnStmt{}
	if hasValue {
		result.Results = []ast.Expr{&ast.Ident{Name: valueName}}
	}
	return result
}

// lowerSimpleArrayAssignmentCall stages a Java simple array assignment as one
// helper call. Go evaluates call arguments from left to right, so the array
// reference, index, and right-hand side are all evaluated exactly once before
// ArraySet performs Java's null/bounds checks and the final store. Compound
// assignments deliberately do not use this path: Java checks and saves their
// old array component before evaluating the right-hand side.
func lowerSimpleArrayAssignmentCall(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	if node == nil || node.Type() != "assignment_expression" || node.ChildCount() < 3 {
		return nil, false
	}
	lhsNode := node.ChildByFieldName("left")
	operatorNode := node.ChildByFieldName("operator")
	rhsNode := node.ChildByFieldName("right")
	if lhsNode == nil || lhsNode.Type() != "array_access" || operatorNode == nil || operatorNode.Content(source) != "=" || rhsNode == nil {
		return nil, false
	}

	arrayNode := lhsNode.ChildByFieldName("array")
	indexNode := lhsNode.ChildByFieldName("index")
	if arrayNode == nil && nodeutil.SemanticNamedChildCount(lhsNode) > 0 {
		arrayNode = nodeutil.SemanticNamedChild(lhsNode, 0)
	}
	if indexNode == nil && nodeutil.SemanticNamedChildCount(lhsNode) > 1 {
		indexNode = nodeutil.SemanticNamedChild(lhsNode, 1)
	}
	if arrayNode == nil || indexNode == nil {
		return nil, false
	}

	lhsJavaType, ok := inferExprJavaType(lhsNode, ctx, source)
	if !ok || strings.TrimSpace(lhsJavaType) == "" {
		return nil, false
	}
	rhsCtx := ctx.Clone()
	rhsCtx.expectedType = lhsJavaType
	rhsCtx.expectedTypeRoot = rhsNode
	rhs := ParseExpr(rhsNode, source, rhsCtx)
	rhs = coerceArgumentToExpectedType(rhs, rhsNode, lhsJavaType, ctx, source)
	array := ParseExpr(arrayNode, source, ctx)
	index := parseJavaIndexExpr(indexNode, source, ctx)
	if javaBinaryOperandMayHaveEffects(indexNode, source, ctx) || javaBinaryOperandMayHaveEffects(rhsNode, source, ctx) {
		array = snapshotJavaExpressionValue(array, ctx)
	}
	if javaBinaryOperandMayHaveEffects(rhsNode, source, ctx) {
		index = snapshotJavaExpressionValueForType(index, "int", ctx)
	}
	if _, componentType, componentID, reified := expressionUsesReifiedReferenceArray(arrayNode, ctx, source); reified {
		typeArguments := []ast.Expr{componentType}
		if unwrapped := unwrapParenthesizedExpressionNode(rhsNode); unwrapped != nil && unwrapped.Type() == "null_literal" {
			// Go cannot infer the helper's Input type from untyped nil. Keep
			// Result as the Java static component view and accept null as any.
			typeArguments = append(typeArguments, ast.NewIdent("any"))
		}
		return stdjavaGenericCall(ctx, "ReferenceArrayAssign", typeArguments, []ast.Expr{
			array,
			index,
			rhs,
			componentID,
		}), true
	}
	if _, componentType, _, primitive := expressionUsesPrimitiveArray(arrayNode, ctx, source); primitive {
		return stdjavaGenericCall(ctx, "PrimitiveArrayAssign", []ast.Expr{componentType}, []ast.Expr{
			array,
			index,
			rhs,
		}), true
	}

	return stdjavaCall(ctx, "ArraySet",
		array,
		index,
		rhs,
	), true
}

// Reference values now retain their nullable representation across boundaries.
func requireNullableValueBackedExpression(value ast.Expr, node *sitter.Node, expectedType string, ctx Ctx, source []byte) ast.Expr {
	return value
}

func inferEnhancedForElementJavaType(valueNode *sitter.Node, source []byte, ctx Ctx) (string, bool) {
	if valueNode == nil {
		return "", false
	}
	// Map views are emitted as native slices. Their element's Java type still
	// comes from the map arguments, including when the loop binding unboxes it.
	if valueNode.Type() == "method_invocation" {
		object := valueNode.ChildByFieldName("object")
		name := valueNode.ChildByFieldName("name")
		if object != nil && name != nil {
			if receiver, known := intrinsicReceiverTypeName(object, ctx, source); known && (containsString(mapTypeNames, receiver) || receiver == "ConcurrentHashMap") {
				elements := receiverElementJavaTypes(object, ctx, source)
				switch name.Content(source) {
				case "values":
					if len(elements) == 2 {
						return elements[1], true
					}
				case "keySet":
					if len(elements) == 2 {
						return elements[0], true
					}
				}
			}
		}
	}
	rangeType, ok := inferExprJavaType(valueNode, ctx, source)
	if !ok {
		return "", false
	}
	rangeType = strings.TrimSpace(rangeType)
	if strings.HasSuffix(rangeType, "[]") {
		elementType := strings.TrimSpace(rangeType[:len(rangeType)-2])
		return elementType, elementType != ""
	}

	if element, iterable := iterationElementType(rangeType, "java.lang.Iterable", ctx); iterable {
		return element, true
	}
	base, typeArgs := parseJavaTypeString(rangeType)
	if len(typeArgs) != 1 {
		return "", false
	}
	base = stripJavaQualifier(base)
	isIterableCollection := containsString(listTypeNames, base) || containsString(setTypeNames, base)
	switch base {
	case "Collection", "Iterable", "Queue", "Deque", "ArrayDeque", "PriorityQueue":
		isIterableCollection = true
	}
	if !isIterableCollection {
		return "", false
	}
	return typeArgs[0], true
}

// enhancedForReferenceElementView converts only the element currently selected
// by a reference-array range. Reference loop variables request their declared
// Java view (for example a Base view while iterating Child[]); primitive loop
// variables first read the boxed component view and then apply Java's permitted
// unboxing/widening conversion.
func enhancedForReferenceElementView(
	raw ast.Expr,
	componentJavaType string,
	componentGoType ast.Expr,
	componentTypeID ast.Expr,
	bindingJavaType string,
	ctx Ctx,
) ast.Expr {
	bindingJavaType = strings.TrimSpace(bindingJavaType)
	if bindingJavaType == "" {
		bindingJavaType = componentJavaType
	}

	if expectedPrimitive, primitiveBinding := javaPrimitiveType(bindingJavaType); primitiveBinding {
		value := ast.Expr(stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{componentGoType}, []ast.Expr{raw, componentTypeID}))
		value, _ = convertJavaValue(value, componentJavaType, expectedPrimitive, ctx)
		return value
	}

	bindingGoType := javaTypeStringToGoTypeExpr(bindingJavaType, inScopeTypeParameters(ctx), ctx)
	bindingGoType = abstractClassToInterface(bindingGoType, bindingJavaType, ctx)
	bindingTypeID, ok := javaTypeDescriptorExpr(bindingJavaType, ctx)
	if !ok {
		bindingGoType = componentGoType
		bindingTypeID = componentTypeID
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{bindingGoType}, []ast.Expr{raw, bindingTypeID})
}

// lowerSimpleLocalNumericCompoundAssignmentStmt handles the statement-only
// subset of Java compound assignment that does not need the value-producing
// closure used by lowerAssignmentExpression. The target must be a primitive
// numeric local (or parameter), and the RHS must be free of observable side
// effects. Complex storage locations and effectful RHS expressions retain the
// staged fallback so their address, old value, and RHS keep Java evaluation
// order.
func lowerSimpleLocalNumericCompoundAssignmentStmt(node *sitter.Node, source []byte, ctx Ctx) (ast.Stmt, bool) {
	if node == nil || node.Type() != "assignment_expression" || node.ChildCount() < 3 || ctx.localScope == nil {
		return nil, false
	}

	lhsNode := node.ChildByFieldName("left")
	opNode := node.ChildByFieldName("operator")
	rhsNode := node.ChildByFieldName("right")
	if lhsNode == nil || lhsNode.Type() != "identifier" || opNode == nil || rhsNode == nil {
		return nil, false
	}
	operator := opNode.Content(source)
	if operator == "=" {
		return nil, false
	}

	local := ctx.localScope.FindVariable(lhsNode.Content(source))
	if local == nil || local.Nullable {
		return nil, false
	}
	lhsNumeric, lhsPrimitive := primitiveJavaNumericType(local.OriginalType)
	if !lhsPrimitive || !isSideEffectFreeCompoundAssignmentRHS(rhsNode) {
		return nil, false
	}

	rhsJavaType, rhsTypeKnown := inferExprJavaType(rhsNode, ctx, source)
	rhsNumeric, rhsPrimitive := primitiveJavaNumericType(rhsJavaType)
	if !rhsTypeKnown || !rhsPrimitive {
		return nil, false
	}

	lhs := ParseExpr(lhsNode, source, ctx)
	rhs := ParseExpr(rhsNode, source, ctx)
	if canUseNativeGoNumericCompoundAssignment(operator, lhsNumeric, rhsNumeric) {
		return &ast.AssignStmt{
			Lhs: []ast.Expr{lhs},
			Tok: StrToToken(operator),
			Rhs: []ast.Expr{rhs},
		}, true
	}

	value, ok := compoundAssignmentValue(operator, lhs, rhs, local.OriginalType, rhsJavaType, ctx)
	if !ok {
		return nil, false
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{lhs},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{value},
	}, true
}

// primitiveJavaNumericType deliberately excludes boxed numeric types. Their
// unboxing/null behavior is observable and remains on the staged fallback.
func primitiveJavaNumericType(javaType string) (string, bool) {
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "byte", "short", "char", "int", "long", "float", "double":
		return stripJavaQualifier(base), true
	default:
		return "", false
	}
}

// canUseNativeGoNumericCompoundAssignment identifies operations for which Go's
// typed compound statement has the same result as Java's promotion followed by
// narrowing. Mixed types need explicit conversions; char needs uint16
// narrowing; and shifts need Java's 5/6-bit distance mask.
func canUseNativeGoNumericCompoundAssignment(operator, lhsType, rhsType string) bool {
	if lhsType != rhsType || lhsType == "char" {
		return false
	}
	if lhsType == "float" || lhsType == "double" {
		switch operator {
		case "+=", "-=", "*=", "/=":
			return true
		default:
			return false
		}
	}
	switch operator {
	case "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=":
		return true
	default:
		return false
	}
}

// isSideEffectFreeCompoundAssignmentRHS is intentionally conservative. A
// statement fast path is an optimization, so unfamiliar expression forms use
// the existing staged lowering. Reads, arithmetic, casts, and indexing are
// safe; calls, updates, assignments, allocation, and ternaries are not admitted.
func isSideEffectFreeCompoundAssignmentRHS(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "identifier", "this", "super",
		"decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal",
		"decimal_floating_point_literal", "hex_floating_point_literal", "character_literal":
		return true
	case "parenthesized_expression":
		return nodeutil.SemanticNamedChildCount(node) == 1 && isSideEffectFreeCompoundAssignmentRHS(nodeutil.SemanticNamedChild(node, 0))
	case "unary_expression":
		count := nodeutil.SemanticNamedChildCount(node)
		return count > 0 && isSideEffectFreeCompoundAssignmentRHS(nodeutil.SemanticNamedChild(node, int(count)-1))
	case "cast_expression":
		return nodeutil.SemanticNamedChildCount(node) == 2 && isSideEffectFreeCompoundAssignmentRHS(nodeutil.SemanticNamedChild(node, 1))
	case "binary_expression":
		return node.ChildCount() >= 3 &&
			isSideEffectFreeCompoundAssignmentRHS(node.ChildByFieldName("left")) &&
			isSideEffectFreeCompoundAssignmentRHS(node.ChildByFieldName("right"))
	case "field_access":
		object := node.ChildByFieldName("object")
		return object != nil && isSideEffectFreeCompoundAssignmentRHS(object)
	case "array_access":
		array := node.ChildByFieldName("array")
		index := node.ChildByFieldName("index")
		if array == nil && nodeutil.SemanticNamedChildCount(node) > 0 {
			array = nodeutil.SemanticNamedChild(node, 0)
		}
		if index == nil && nodeutil.SemanticNamedChildCount(node) > 1 {
			index = nodeutil.SemanticNamedChild(node, 1)
		}
		return isSideEffectFreeCompoundAssignmentRHS(array) && isSideEffectFreeCompoundAssignmentRHS(index)
	default:
		return false
	}
}

func TryParseStmt(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	switch node.Type() {
	case "assert_statement":
		return parseAssertionStatement(node, source, ctx)
	case ";", "empty_statement":
		return &ast.EmptyStmt{Implicit: true}
	case "ERROR":
		log.WithFields(log.Fields{
			"parsed":    node.Content(source),
			"className": ctx.className,
		}).Warn("Statement parse error")
		return &ast.BadStmt{}
	case "comment", "line_comment", "block_comment":
		return &ast.BadStmt{}
	case "class_declaration", "interface_declaration", "enum_declaration":
		// A type declaration appearing as a statement is a local class. Hoist it to
		// file scope, capturing referenced enclosing locals as fields, and drop the
		// in-body declaration (signalled by an empty statement the block filters).
		hoistLocalClass(node, source, ctx)
		return &ast.EmptyStmt{Implicit: true}
	case "local_variable_declaration":
		return parseLocalVariableDeclaration(node, source, ctx)
	case "variable_declarator":
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")
		if nameNode == nil {
			return &ast.BadStmt{}
		}
		declaration := &ast.AssignStmt{Lhs: []ast.Expr{localBindingIdent(nameNode, source, ctx)}, Tok: token.DEFINE}
		if valueNode != nil {
			value := ParseExpr(valueNode, source, ctx)
			if expectedType := strings.TrimSpace(ctx.expectedType); expectedType != "" && !isVarKeywordType(expectedType) {
				value = coerceArgumentToExpectedType(value, valueNode, expectedType, ctx, source)
			}
			declaration.Rhs = []ast.Expr{value}
		}
		return declaration
	case "assignment_expression":
		if lowered, ok := lowerStaticFieldAssignment(node, source, ctx); ok {
			return &ast.ExprStmt{X: lowered}
		}
		operator := node.ChildByFieldName("operator").Content(source)
		// Compound assignment in Java is not just Go's corresponding assignment
		// token: String += converts arbitrary operands, arithmetic narrows back to
		// the target type, and the target is evaluated exactly once. Reuse the
		// value-producing lowering for every compound operator and discard the
		// resulting value when the assignment appears as a statement.
		if operator != "=" {
			if stmt, ok := lowerSimpleLocalNumericCompoundAssignmentStmt(node, source, ctx); ok {
				return stmt
			}
			return &ast.ExprStmt{X: lowerAssignmentExpression(node, source, ctx)}
		}
		if call, ok := lowerSimpleArrayAssignmentCall(node, source, ctx); ok {
			return &ast.ExprStmt{X: call}
		}

		lhsNode := node.ChildByFieldName("left")
		rhsNode := node.ChildByFieldName("right")
		assignVar := ParseExpr(lhsNode, source, ctx)
		rhsCtx := ctx.Clone()
		if lhsJavaType, ok := inferExprJavaType(lhsNode, ctx, source); ok {
			rhsCtx.expectedType = lhsJavaType
			rhsCtx.expectedTypeRoot = rhsNode
		}
		assignVal := ParseExpr(rhsNode, source, rhsCtx)
		if lhsJavaType := strings.TrimSpace(rhsCtx.expectedType); lhsJavaType != "" {
			assignVal = coerceArgumentToExpectedType(assignVal, rhsNode, lhsJavaType, ctx, source)
		}

		return &ast.AssignStmt{
			Lhs: []ast.Expr{assignVar},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{assignVal},
		}
	case "update_expression":
		operandNode, operatorNode, _ := javaUpdateExpressionParts(node)
		if operandNode == nil || operatorNode == nil {
			return &ast.BadStmt{}
		}
		if _, ok := resolveStaticFieldAccess(operandNode, source, ctx); ok {
			return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
		}
		if operandType, known := inferExprJavaType(operandNode, ctx, source); known {
			if _, wrapper := builtinJavaWrapperPrimitive(operandType, ctx); wrapper {
				return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
			}
		}
		return &ast.IncDecStmt{X: ParseExpr(operandNode, source, ctx), Tok: StrToToken(operatorNode.Type())}
	case "resource_specification":
		return ParseStmt(nodeutil.SemanticNamedChild(node, 0), source, ctx)
	case "resource":
		// Resource variables retain their declared Java type for overload and
		// intrinsic dispatch inside the try body, just like ordinary locals.
		if typeNode, nameNode := node.ChildByFieldName("type"), node.ChildByFieldName("name"); typeNode != nil && nameNode != nil {
			originalType := typeNode.Content(source)
			parsedType := symbol.NodeToStr(astutil.ParseType(typeNode, source))
			if isVarKeywordType(originalType) {
				if inferred, ok := inferExprJavaType(node.ChildByFieldName("value"), ctx, source); ok {
					originalType = inferred
					parsedType = symbol.NodeToStr(javaTypeStringToGoTypeExpr(inferred, inScopeTypeParameters(ctx), ctx))
				}
			}
			recordLocalVariableDefinition(ctx, nameNode.Content(source), originalType, parsedType)
		}
		var offset int
		if nodeutil.SemanticNamedChild(node, 0).Type() == "modifiers" {
			offset = 1
		}
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ParseExpr(nodeutil.SemanticNamedChild(node, 1+offset), source, ctx)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ParseExpr(nodeutil.SemanticNamedChild(node, 2+offset), source, ctx)},
		}
	case "method_invocation":
		return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
	case "constructor_body", "block":
		return parseStatementBlock(node, nil, source, ctx)
	case "expression_statement":
		if stmt := TryParseStmt(nodeutil.SemanticNamedChild(node, 0), source, ctx); stmt != nil {
			return stmt
		}
		return &ast.ExprStmt{X: ParseExpr(nodeutil.SemanticNamedChild(node, 0), source, ctx)}
	case "explicit_constructor_invocation":
		// This is when a constructor calls another constructor with the use of
		// something such as `this(args...)`
		argsNode := node.ChildByFieldName("arguments")
		if argsNode == nil && nodeutil.SemanticNamedChildCount(node) > 1 {
			argsNode = nodeutil.SemanticNamedChild(node, 1)
		}
		var args []ast.Expr
		if argsNode != nil {
			args = ParseNode(argsNode, source, ctx).([]ast.Expr)
		}

		constructorNode := node.ChildByFieldName("constructor")
		if constructorNode == nil && nodeutil.SemanticNamedChildCount(node) > 0 {
			constructorNode = nodeutil.SemanticNamedChild(node, 0)
		}
		if constructorNode != nil && constructorNode.Type() == "super" && ctx.currentClass != nil {
			if statement := characterIOSuperConstructor(args, ctx); statement != nil {
				return statement
			}
			if statement := filterInputSuperConstructor(args, ctx); statement != nil {
				return statement
			}
			superType := strings.TrimSpace(ctx.currentClass.Superclass)
			if superType != "" {
				base, superArgStrs := parseJavaTypeString(superType)
				if base != "" {
					superName := stripJavaQualifier(base)
					// A built-in exception superclass is constructed via the stdjava
					// runtime; the embedded field is named after the runtime type.
					if storage, builtin := builtinExceptionStorageTypeName(base, ctx); builtin {
						recvName := ctx.className
						if recvName == "" && ctx.currentClass.Class != nil {
							recvName = ctx.currentClass.Class.Name
						}
						if recvName != "" {
							if call, ok := assertionErrorConstructorArguments(base, argsNode, args, ctx, source); ok {
								return &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(ShortName(recvName)), Sel: ast.NewIdent(storage)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{call}}
							}

							return &ast.AssignStmt{
								Lhs: []ast.Expr{&ast.SelectorExpr{
									X:   &ast.Ident{Name: ShortName(recvName)},
									Sel: &ast.Ident{Name: storage},
								}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{builtinExceptionConstructorExpr(base, argsNode, args, ctx, source)},
							}
						}
					}
					// Default constructor name; overridden below by the superclass's
					// actual generated constructor name when its scope is known.
					constructorFnName := "New" + superName
					if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil && scope.Class.Name != "" {
						superName = scope.Class.Name
						// Use the superclass's generated constructor function name so the
						// call matches the emitted decl exactly (e.g. `newAnimal`, not
						// `Newanimal`, for a package-private superclass).
						if ctorName := constructorFuncName(scope); ctorName != "" {
							constructorFnName = ctorName
						} else {
							constructorFnName = "New" + superName
						}
					}
					funExpr := ast.Expr(&ast.Ident{Name: constructorFnName})
					if len(superArgStrs) > 0 {
						scopeTypeParams := inScopeTypeParameters(ctx)
						typeArgs := make([]ast.Expr, 0, len(superArgStrs))
						for _, arg := range superArgStrs {
							typeArgs = append(typeArgs, javaTypeStringToGoTypeExpr(arg, scopeTypeParams, ctx))
						}
						funExpr = applyTypeArguments(funExpr, typeArgs)
					}
					recvName := ctx.className
					if recvName == "" && ctx.currentClass.Class != nil {
						recvName = ctx.currentClass.Class.Name
					}
					if recvName != "" {
						return &ast.AssignStmt{
							Lhs: []ast.Expr{&ast.SelectorExpr{
								X:   &ast.Ident{Name: ShortName(recvName)},
								Sel: &ast.Ident{Name: superclassEmbeddedSelectorName(ctx.currentClass, ctx)},
							}},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{&ast.CallExpr{Fun: funExpr, Args: args}},
						}
					}
				}
			}
		}

		return &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "New" + ctx.className},
				Args: args,
			},
		}
	case "return_statement":
		if ctx.tryReturnTarget != nil {
			stmts := []ast.Stmt{}
			if ctx.tryReturnTarget.HasValue && nodeutil.SemanticNamedChildCount(node) > 0 {
				returnCtx := ctx.Clone()
				if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
					returnCtx.expectedType = ctx.localScope.OriginalType
					returnCtx.expectedTypeRoot = nodeutil.SemanticNamedChild(node, 0)
				}
				stmts = append(stmts, &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.ValueName}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{parseReturnValue(nodeutil.SemanticNamedChild(node, 0), source, returnCtx)},
				})
			}
			// A return in a finally block supersedes any break/continue that was
			// pending from the try or catch. Clear the other abrupt-completion
			// channel only after the return expression has evaluated successfully.
			if len(ctx.tryReturnTarget.controlList) > 0 {
				stmts = append(stmts, &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.ControlName}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}},
				})
			}
			stmts = append(stmts, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.FlagName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
			})
			stmts = append(stmts, &ast.ReturnStmt{})
			return &ast.BlockStmt{List: stmts}
		}

		if nodeutil.SemanticNamedChildCount(node) < 1 {
			return replayedMethodReturnStmt(ctx, false, "")
		}
		returnCtx := ctx.Clone()
		if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
			returnCtx.expectedType = ctx.localScope.OriginalType
			returnCtx.expectedTypeRoot = nodeutil.SemanticNamedChild(node, 0)
		}
		return &ast.ReturnStmt{Results: []ast.Expr{parseReturnValue(nodeutil.SemanticNamedChild(node, 0), source, returnCtx)}}
	case "labeled_statement":
		return lowerLabeledStatement(node, source, ctx)
	case "break_statement":
		return lowerTryControlBranch(node, source, ctx, token.BREAK)
	case "continue_statement":
		return lowerTryControlBranch(node, source, ctx, token.CONTINUE)
	case "throw_statement":
		return &ast.ExprStmt{X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "panic"},
			Args: []ast.Expr{ParseExpr(nodeutil.SemanticNamedChild(node, 0), source, ctx)},
		}}
	case "if_statement":
		if instanceofPatternNode(node.ChildByFieldName("condition")) == nil && patternConditionHasBindings(node.ChildByFieldName("condition"), source) {
			return lowerPatternExpressionCondition(node.ChildByFieldName("condition"), source, ctx,
				func(branchCtx Ctx) ast.Stmt {
					return ParseStmt(node.ChildByFieldName("consequence"), source, branchCtx)
				},
				func(branchCtx Ctx) ast.Stmt {
					if alternative := node.ChildByFieldName("alternative"); alternative != nil {
						return ParseStmt(alternative, source, branchCtx)
					}
					return nil
				})
		}

		var other ast.Stmt
		if node.ChildByFieldName("alternative") != nil {
			other = ParseStmt(node.ChildByFieldName("alternative"), source, ctx)
		}

		// `if (x instanceof T t)` (Java 16+) binds t for the if-body. Lower it to
		// the Go type-assertion idiom: `if t, ok := any(x).(T); ok { ... }` with the
		// bound variable registered for the body. Only this pattern form diverges
		// from the plain if lowering below.
		if patternNode := instanceofPatternNode(node.ChildByFieldName("condition")); patternNode != nil {
			initStmt, condExpr, bodyCtx := lowerInstanceofPattern(patternNode, source, ctx)
			body := ParseStmt(node.ChildByFieldName("consequence"), source, bodyCtx)
			if _, ok := body.(*ast.BlockStmt); !ok {
				body = &ast.BlockStmt{List: []ast.Stmt{body}}
			}
			if name := patternNode.ChildByFieldName("name"); name != nil {
				block := body.(*ast.BlockStmt)
				block.List = append([]ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: "_"}}, Tok: token.ASSIGN, Rhs: []ast.Expr{localBindingIdent(name, source, bodyCtx)}}}, block.List...)
			}
			return &ast.IfStmt{
				Init: initStmt,
				Cond: condExpr,
				Body: body.(*ast.BlockStmt),
				Else: other,
			}
		}

		// If the `if` statement is inline, replace the line with a full block
		body := ParseStmt(node.ChildByFieldName("consequence"), source, ctx)
		if _, ok := body.(*ast.BlockStmt); !ok {
			body = &ast.BlockStmt{List: []ast.Stmt{
				body,
			}}
		}

		return &ast.IfStmt{
			Cond: parseJavaBooleanExpr(node.ChildByFieldName("condition"), source, ctx),
			Body: body.(*ast.BlockStmt),
			Else: other,
		}
	case "enhanced_for_statement":
		// An enhanced for statement has the following fields:
		// variables for the variable being declared (ex: int n)
		// then the expression that is being ranged over
		// and finally, the block of the expression

		total := int(nodeutil.SemanticNamedChildCount(node))
		typeNode := node.ChildByFieldName("type")
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")
		bodyNode := node.ChildByFieldName("body")

		// Fallback for grammars that don't provide named fields.
		if nameNode == nil && total >= 3 {
			nameNode = nodeutil.SemanticNamedChild(node, total-3)
		}
		if valueNode == nil && total >= 2 {
			valueNode = nodeutil.SemanticNamedChild(node, total-2)
		}
		if bodyNode == nil && total >= 1 {
			bodyNode = nodeutil.SemanticNamedChild(node, total-1)
		}

		loopCtx := ctx.Clone()
		bindingJavaType := ""
		if nameNode != nil && typeNode != nil {
			originalType := typeNode.Content(source)
			parsedType := symbol.NodeToStr(astutil.ParseType(typeNode, source))
			if isVarKeywordType(originalType) {
				if inferredType, ok := inferEnhancedForElementJavaType(valueNode, source, ctx); ok {
					originalType = inferredType
					parsedType = symbol.NodeToStr(javaTypeStringToGoTypeExpr(inferredType, inScopeTypeParameters(ctx), ctx))
				}
			}
			bindingJavaType = originalType
			recordLocalVariableDefinition(loopCtx, nameNode.Content(source), originalType, parsedType)
		}

		bindingValue := ast.Expr(&ast.Ident{Name: "_"})
		if nameNode != nil {
			bindingValue = ParseExpr(nameNode, source, loopCtx)
		}
		rangeValue := bindingValue
		rangeExpr := ast.Expr(&ast.BadExpr{})
		var referenceBinding ast.Stmt
		if valueNode != nil {
			rangeExpr = ParseExpr(valueNode, source, ctx)
			if componentJavaType, componentGoType, componentTypeID, reified := expressionUsesReifiedReferenceArray(valueNode, ctx, source); reified {
				usedNames := affineLoopUsedNames(node, source, ctx)
				rawName := synchronizedUniqueLocalName(fmt.Sprintf("__java2goEnhancedForElement_%d", node.StartByte()), usedNames)
				rawValue := &ast.Ident{Name: rawName}
				rangeValue = rawValue
				rangeExpr = stdjavaCall(ctx, "ReferenceArrayIterationElements", rangeExpr)
				referenceBinding = &ast.AssignStmt{
					Lhs: []ast.Expr{bindingValue},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{enhancedForReferenceElementView(
						rawValue,
						componentJavaType,
						componentGoType,
						componentTypeID,
						bindingJavaType,
						ctx,
					)},
				}
			} else if _, _, _, primitive := expressionUsesPrimitiveArray(valueNode, ctx, source); primitive {
				rangeExpr = stdjavaCall(ctx, "PrimitiveArrayIterationElements", rangeExpr)
			}
			// Collection iteration reads list slots when the iterator advances;
			// an array-backed list must not snapshot future elements.
			if erasedCollectionExpression(valueNode, ctx, source) || erasedIterableExpression(valueNode, ctx, source) || nativeSetIterationExpression(valueNode, ctx, source) {
				rawValue := &ast.Ident{Name: fmt.Sprintf("__java2goEnhancedForElement_%d", node.StartByte())}
				rangeValue = rawValue
				rangeExpr = stdjavaCall(ctx, "ErasedCollectionIterationElements", intrinsicExecutionExpr(ctx), rangeExpr)
				componentJavaType, known := inferEnhancedForElementJavaType(valueNode, source, ctx)
				if !known {
					componentJavaType = "Object"
				}
				componentJavaType = readableWildcardProjection(componentJavaType)
				componentType := javaTypeStringToGoTypeExpr(componentJavaType, inScopeTypeParameters(ctx), ctx)
				componentID, known := javaTypeDescriptorExpr(componentJavaType, ctx)
				if !known {
					componentID = stdjavaQualifiedExpr("ObjectTypeID", ctx)
				}
				referenceBinding = &ast.AssignStmt{Lhs: []ast.Expr{bindingValue}, Tok: token.DEFINE,
					Rhs: []ast.Expr{enhancedForReferenceElementView(rawValue, componentJavaType, componentType, componentID, bindingJavaType, ctx)}}
			} else if collectionNeedsSliceForRange(valueNode, ctx, source) {
				rangeExpr = stdjavaCall(ctx, "CollectionIterationElements", rangeExpr)
			}
			if referenceBinding == nil && nameNode != nil {
				if elementType, known := inferEnhancedForElementJavaType(valueNode, source, ctx); known {
					rawName := fmt.Sprintf("__java2goEnhancedForElement_%d", node.StartByte())
					rawValue := &ast.Ident{Name: rawName}
					if converted, needed := convertJavaValue(rawValue, elementType, bindingJavaType, ctx); needed && elementType != bindingJavaType {
						rangeValue = rawValue
						referenceBinding = &ast.AssignStmt{Lhs: []ast.Expr{bindingValue}, Tok: token.DEFINE, Rhs: []ast.Expr{converted}}
					}
				}
			}
		}
		rangeBody := &ast.BlockStmt{}
		if bodyNode != nil {
			parsedBody := ParseStmt(bodyNode, source, loopCtx)
			if block, ok := parsedBody.(*ast.BlockStmt); ok {
				rangeBody = block
			} else {
				rangeBody.List = []ast.Stmt{parsedBody}
			}
		}
		if referenceBinding != nil {
			rangeBody.List = append([]ast.Stmt{referenceBinding}, rangeBody.List...)
		}
		if ident, ok := bindingValue.(*ast.Ident); ok && ident.Name != "_" {
			bindingDeclaration := ast.Stmt(&ast.AssignStmt{
				Lhs: []ast.Expr{ident},
				Tok: token.DEFINE,
			})
			if referenceBinding != nil {
				bindingDeclaration = referenceBinding
			}
			discards := unusedLocalDiscardStatements(bindingDeclaration, node, source)
			if referenceBinding != nil && len(rangeBody.List) > 0 {
				rangeBody.List = append(append([]ast.Stmt{rangeBody.List[0]}, discards...), rangeBody.List[1:]...)
			} else {
				rangeBody.List = append(discards, rangeBody.List...)
			}
		}

		return &ast.RangeStmt{
			// We don't need the type of the variable for the range expression
			Key:   &ast.Ident{Name: "_"},
			Value: rangeValue,
			Tok:   token.DEFINE,
			X:     rangeExpr,
			Body:  rangeBody,
		}
	case "for_statement":
		var init, post ast.Stmt
		initNode := node.ChildByFieldName("init")
		if initNode != nil {
			init = parseForInitializer(initNode, source, ctx)
		}
		loopCtx, affineBindings := prepareAffineArrayLoop(node, source, ctx)
		if node.ChildByFieldName("update") != nil {
			post = ParseStmt(node.ChildByFieldName("update"), source, ctx)
		}
		var cond ast.Expr
		if node.ChildByFieldName("condition") != nil {
			cond = parseJavaBooleanExpr(node.ChildByFieldName("condition"), source, ctx)
		}
		// A canonical inner column loop can reuse equal-span row slices only when
		// all selected bindings were proven non-null by an enclosing loop version.
		// Loops that create their own view caches first take the normal versioning
		// path; a later nested loop may consume those proofs.
		if len(affineBindings) == 0 && !loopCtx.disableAffineArrayRowSpecialization {
			if rowCtx, rowPlan := prepareAffineArrayRowLoop(node, source, loopCtx); rowPlan != nil {
				return lowerAffineArrayRowLoop(node, source, loopCtx, rowCtx, rowPlan, init, cond, post)
			}
		}

		fastCtx := loopCtx.Clone()
		// A nested loop may only inherit call sites owned by an enclosing
		// versioned loop. Add only this loop's bindings to the binding-specific
		// proof set; inherited bindings retain the validity state of their own
		// enclosing branch.
		if len(affineBindings) > 0 {
			fastCtx.affineArrayNonNullBindings = cloneAffineArrayNonNullBindings(loopCtx.affineArrayNonNullBindings)
			for _, binding := range affineBindings {
				if binding != nil {
					fastCtx.affineArrayNonNullBindings[binding] = struct{}{}
				}
			}
		}
		guardedCtx := loopCtx.Clone()
		guardedCtx.localScope = cloneLocalScopeDefinition(loopCtx.localScope)
		guardedCtx.suppressUnsupportedDiagnostics = true
		// The guarded copy is rendered from the same Java source spans as the fast
		// copy. Keep its cold-path row planning isolated: otherwise a specialization
		// using only inherited non-null bindings can register an identical hoist in
		// the shared method map, which the fast copy then consumes twice.
		guardedCtx.disableAffineArrayRowSpecialization = true
		guardedCtx.affineArrayRowHoists = make(map[affineArrayCallSiteKey][]*affineArrayRowHoist)

		body := parseLoopBody(node.ChildByFieldName("body"), source, fastCtx)
		if initNode != nil && initNode.Type() == "local_variable_declaration" {
			body.List = append(unusedLocalDiscardStatements(init, node, source), body.List...)
		}

		loop := &ast.ForStmt{
			Init: init,
			Cond: cond,
			Post: post,
			Body: body,
		}
		cacheStatements := affineArrayLoopCacheStatements(affineBindings)
		if len(cacheStatements) == 0 {
			return loop
		}

		guardedBody := parseLoopBody(node.ChildByFieldName("body"), source, guardedCtx)
		if initNode != nil && initNode.Type() == "local_variable_declaration" {
			guardedBody.List = append(unusedLocalDiscardStatements(init, node, source), guardedBody.List...)
		}
		guardedLoop := &ast.ForStmt{
			Init: init,
			Cond: cond,
			Post: post,
			Body: guardedBody,
		}
		validity := affineArrayLoopValidityCondition(affineBindings)
		if validity == nil {
			return loop
		}
		versionedLoop := &ast.IfStmt{
			Cond: validity,
			Body: &ast.BlockStmt{List: applyAffineArrayRowHoists(node, loop, fastCtx)},
			Else: &ast.BlockStmt{List: []ast.Stmt{guardedLoop}},
		}
		return &ast.BlockStmt{List: append(cacheStatements, versionedLoop)}
	case "while_statement":
		if readLineLoop, ok := lowerBufferedReaderReadLineWhile(node, source, ctx); ok {
			return readLineLoop
		}
		return &ast.ForStmt{
			Cond: parseJavaBooleanExpr(node.ChildByFieldName("condition"), source, ctx),
			Body: parseLoopBody(node.ChildByFieldName("body"), source, ctx),
		}
	case "do_statement":
		// Java continue in a do-while evaluates the condition before deciding
		// whether to start another iteration. Native Go continue would skip a
		// guard appended to the loop body, so continues targeting this exact Java
		// node are rewritten to a synthetic condition-phase label. Keep the Java
		// body in its own block so a forward goto exits local-variable scopes
		// instead of illegally jumping over their declarations.
		doCtx := ctx.Clone()
		doCtx.doWhileContinueTargets = cloneDoWhileContinueTargets(ctx.doWhileContinueTargets)
		continueTarget := &doWhileContinueTarget{
			Label: fmt.Sprintf("__java2goDoCondition_%d_%d", node.StartByte(), ctx.nextControlLabelIndex()),
		}
		if key, ok := javaControlKey(node); ok {
			doCtx.doWhileContinueTargets[key] = continueTarget
		}
		body := parseLoopBody(node.ChildByFieldName("body"), source, doCtx)

		conditionGuard := &ast.IfStmt{
			Cond: &ast.UnaryExpr{
				Op: token.NOT,
				X: &ast.ParenExpr{
					X: parseJavaBooleanExpr(node.ChildByFieldName("condition"), source, ctx),
				},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}},
		}
		loopBody := &ast.BlockStmt{List: []ast.Stmt{body}}
		if continueTarget.Used {
			loopBody.List = append(loopBody.List, &ast.LabeledStmt{
				Label: &ast.Ident{Name: continueTarget.Label},
				Stmt:  conditionGuard,
			})
		} else {
			loopBody.List = append(loopBody.List, conditionGuard)
		}

		return &ast.ForStmt{
			Body: loopBody,
		}
	case "switch_statement", "switch_expression":
		// A classic switch statement parses as `switch_expression` in tree-sitter.
		// Children: a parenthesized tag expression followed by the switch_block.
		var tagNode, blockNode *sitter.Node
		for _, c := range nodeutil.NamedChildrenOf(node) {
			switch c.Type() {
			case "switch_block":
				blockNode = c
			default:
				if tagNode == nil {
					tagNode = c
				}
			}
		}
		if blockNode == nil {
			return &ast.SwitchStmt{Body: &ast.BlockStmt{}}
		}
		tag := ParseExpr(tagNode, source, ctx)
		if javaType, known := inferExprJavaType(tagNode, ctx, source); known {
			tag = javaUnboxExpr(tag, javaType, ctx)
		}
		body := parseSwitchBlock(blockNode, source, ctx)
		tag = canonicalStringSwitch(tag, tagNode, body, source, ctx)
		return &ast.SwitchStmt{Tag: tag, Body: body}
	case "switch_block":
		return parseSwitchBlock(node, source, ctx)
	}
	return nil
}

// Canonical readLine returns a nullable String pointer; ordinary loop lowering
// now preserves assignment and EOF without a native-string slot rewrite.
func lowerBufferedReaderReadLineWhile(node *sitter.Node, source []byte, ctx Ctx) (ast.Stmt, bool) {
	return nil, false
}

func unwrapParenthesizedExpressionNode(node *sitter.Node) *sitter.Node {
	for node != nil && node.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(node) > 0 {
		node = nodeutil.SemanticNamedChild(node, 0)
	}
	return node
}

// explicitLocalVariableType performs symbol-aware lowering only when a local's
// Go declaration must carry an explicit type (no initializer, or a null
// initializer). Short declarations should not call this helper: resolving their
// source type can register imports that never appear in the emitted Go code.
func explicitLocalVariableType(originalType string, ctx Ctx) ast.Expr {
	if erasure, ok := currentErasedCallableOwnerTypeParameterErasure(originalType, ctx); ok {
		physical := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
		return abstractClassToInterface(physical, erasure, ctx)
	}
	return abstractClassToInterface(
		javaTypeStringToGoTypeExpr(originalType, inScopeTypeParameters(ctx), ctx),
		originalType,
		ctx,
	)
}

// nullableLocalVariableType preserves null for Java reference types whose usual
// Go representation is a non-nullable value. Most Java references already lower
// to pointers, slices, interfaces, or maps and can therefore use their normal
// explicit type. String needs an interface slot when its initializer is null;
// boxed primitive objects already have a nullable pointer representation.
func nullableLocalVariableType(originalType string, ctx Ctx) ast.Expr {
	if usesNullableValueStorage(originalType) {
		return &ast.Ident{Name: "any"}
	}
	return explicitLocalVariableType(originalType, ctx)
}

func usesNullableValueStorage(originalType string) bool {
	return false
}

// localVariableDiscardStatements marks Java locals as used without dropping
// their declarations or initializers. Java accepts unused locals while Go does
// not; emitting `_ = local` immediately after the declaration retains
// initializer evaluation and ordering while satisfying Go's compile-time rule.
func localVariableDiscardStatements(stmt ast.Stmt) []ast.Stmt {
	var names []*ast.Ident
	switch typed := stmt.(type) {
	case *ast.AssignStmt:
		if typed.Tok != token.DEFINE {
			return nil
		}
		for _, lhs := range typed.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				names = append(names, ident)
			}
		}
	case *ast.DeclStmt:
		declaration, ok := typed.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR {
			return nil
		}
		for _, spec := range declaration.Specs {
			if valueSpec, ok := spec.(*ast.ValueSpec); ok {
				names = append(names, valueSpec.Names...)
			}
		}
	}

	discards := make([]ast.Stmt, 0, len(names))
	for _, name := range names {
		if name == nil || name.Name == "" || name.Name == "_" {
			continue
		}
		discards = append(discards, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: name.Name}},
		})
	}
	return discards
}

func unusedLocalDiscardStatements(stmt ast.Stmt, scopeNode *sitter.Node, source []byte) []ast.Stmt {
	potential := localVariableDiscardStatements(stmt)
	discards := potential[:0]
	for _, discard := range potential {
		assignment, ok := discard.(*ast.AssignStmt)
		if !ok || len(assignment.Rhs) != 1 {
			continue
		}
		name, ok := assignment.Rhs[0].(*ast.Ident)
		if !ok || javaScopeReadsIdentifier(scopeNode, name.Name, source) {
			continue
		}
		discards = append(discards, discard)
	}
	return discards
}

// javaScopeReadsIdentifier distinguishes a real value read from declarations
// and plain assignment targets. Go rejects a Java local that is only declared
// or assigned, while adding a discard for every local needlessly perturbs all
// generated bodies. This lightweight source-tree check emits guards only for
// locals that otherwise have no read in their lexical statement scope.
func javaScopeReadsIdentifier(scopeNode *sitter.Node, name string, source []byte) bool {
	var reads bool
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || reads {
			return
		}
		if node.Type() == "identifier" && node.Content(source) == name && javaIdentifierIsRead(node, source) {
			reads = true
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(scopeNode)
	return reads
}

func javaIdentifierIsRead(node *sitter.Node, source []byte) bool {
	parent := node.Parent()
	if parent == nil {
		return true
	}
	sameNode := func(other *sitter.Node) bool {
		return other != nil && other.StartByte() == node.StartByte() && other.EndByte() == node.EndByte()
	}
	switch parent.Type() {
	case "variable_declarator", "formal_parameter", "spread_parameter", "catch_formal_parameter":
		if sameNode(parent.ChildByFieldName("name")) || (parent.Type() == "variable_declarator" && sameNode(nodeutil.SemanticNamedChild(parent, 0))) {
			return false
		}
	case "enhanced_for_statement":
		if sameNode(parent.ChildByFieldName("name")) {
			return false
		}
	case "assignment_expression":
		if sameNode(parent.ChildByFieldName("left")) && parent.ChildByFieldName("operator") != nil && parent.ChildByFieldName("operator").Content(source) == "=" {
			return false
		}
	case "field_access":
		if sameNode(parent.ChildByFieldName("field")) {
			return false
		}
	case "method_invocation":
		if sameNode(parent.ChildByFieldName("name")) {
			return false
		}
	}
	return true
}

// parseSwitchBlock lowers a Java switch body into a Go switch body, translating
// Java's fallthrough-by-default semantics into Go's break-by-default. A group
// that ends with `break` becomes an ordinary Go case (the break is dropped); a
// non-terminal group that does not break gets an explicit `fallthrough`. Empty
// groups (a label with no statements) are merged into the next group's case
// list, matching Java's stacked `case` labels.
func parseSwitchBlock(node *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	switchBlock := &ast.BlockStmt{}

	// Arrow-form switches (Java 14+) use `switch_rule` children, each of which is
	// self-contained (no fallthrough). Handle them separately from the classic
	// colon-form `switch_block_statement_group`s.
	for _, c := range nodeutil.NamedChildrenOf(node) {
		if c.Type() == "switch_rule" {
			return parseArrowSwitchBlock(node, source, ctx)
		}
	}

	groups := []*sitter.Node{}
	for _, c := range nodeutil.NamedChildrenOf(node) {
		if c.Type() == "switch_block_statement_group" {
			groups = append(groups, c)
		}
	}

	// Labels carried forward from preceding empty groups (stacked cases).
	var pendingExprs []ast.Expr
	var pendingDefault bool

	for index, group := range groups {
		caseExprs, isDefault, bodyNodes := splitSwitchGroup(group, source, ctx)

		caseExprs = append(pendingExprs, caseExprs...)
		isDefault = isDefault || pendingDefault

		// An empty group (no statements) stacks its labels onto the next group.
		if len(bodyNodes) == 0 && index != len(groups)-1 {
			pendingExprs = caseExprs
			pendingDefault = isDefault
			continue
		}
		pendingExprs = nil
		pendingDefault = false

		clause := &ast.CaseClause{}
		if !isDefault {
			clause.List = caseExprs
		}

		body, terminatedByBreak := parseSwitchGroupBody(bodyNodes, source, ctx)
		clause.Body = body

		// Java cases fall through unless they break/return. Go does the opposite,
		// so add an explicit fallthrough when the group neither breaks nor is the
		// final group.
		if !terminatedByBreak && index != len(groups)-1 && len(body) > 0 {
			clause.Body = append(clause.Body, &ast.BranchStmt{Tok: token.FALLTHROUGH})
		}

		switchBlock.List = append(switchBlock.List, clause)
	}

	return switchBlock
}

// instanceofPatternNode returns the instanceof_expression node if the given
// condition is a direct instanceof pattern with a bound variable (`x instanceof
// T t`), or nil otherwise.
func instanceofPatternNode(condNode *sitter.Node) *sitter.Node {
	if condNode == nil {
		return nil
	}
	// Unwrap a parenthesized condition.
	for condNode.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(condNode) > 0 {
		condNode = nodeutil.SemanticNamedChild(condNode, 0)
	}
	if condNode.Type() == "instanceof_expression" && condNode.ChildByFieldName("name") != nil {
		return condNode
	}
	return nil
}

// lowerInstanceofPattern lowers `x instanceof T t` into the Go type-assertion
// idiom, returning the init statement (`t, ok := any(x).(T)`), the condition
// (`ok`), and a context with the bound variable registered for the if-body.
func lowerInstanceofPattern(node *sitter.Node, source []byte, ctx Ctx) (ast.Stmt, ast.Expr, Ctx) {
	left := node.ChildByFieldName("left")
	right := node.ChildByFieldName("right")
	nameNode := node.ChildByFieldName("name")

	bindName := nameNode.Content(source)
	rightJavaType := right.Content(source)
	assertType := instanceofAssertTypeExpr(right.Content(source), ctx)
	if assertType == nil {
		assertType = &ast.Ident{Name: "any"}
	}

	bodyCtx := ctx.Clone()
	recordLocalVariableDefinition(bodyCtx, bindName, right.Content(source), symbol.NodeToStr(assertType))

	var patternValue ast.Expr = &ast.TypeAssertExpr{
		X:    &ast.CallExpr{Fun: &ast.Ident{Name: "any"}, Args: []ast.Expr{instanceofSubjectExpr(left, rightJavaType, source, ctx)}},
		Type: assertType,
	}
	if _, rank := javaArrayTypeParts(rightJavaType); rank > 0 {
		if descriptor, ok := javaTypeDescriptorExpr(rightJavaType, ctx); ok {
			patternValue = stdjavaGenericCall(ctx, "JavaArrayPattern", []ast.Expr{assertType}, []ast.Expr{
				instanceofSubjectExpr(left, rightJavaType, source, ctx),
				descriptor,
			})
		}
	}
	if descriptor, ok := javaTypeDescriptorExpr(rightJavaType, ctx); ok && !strings.HasSuffix(strings.TrimSpace(rightJavaType), "[]") {
		patternValue = stdjavaGenericCall(ctx, "ObjectPattern", []ast.Expr{assertType}, []ast.Expr{
			instanceofSubjectExpr(left, rightJavaType, source, ctx), descriptor,
		})
	}

	initStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: localBindingName(bindName, bodyCtx)}, &ast.Ident{Name: "ok"}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{patternValue},
	}
	return initStmt, &ast.Ident{Name: "ok"}, bodyCtx
}

// parseArrowSwitchBlock lowers an arrow-form (`case X -> ...`) switch body. Each
// switch_rule maps to a single Go case clause with no fallthrough. A `default ->`
// rule becomes the default clause. The rule body is an expression statement, a
// block, or a throw, all converted as ordinary statements.
func parseArrowSwitchBlock(node *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	switchBlock := &ast.BlockStmt{}

	for _, rule := range nodeutil.NamedChildrenOf(node) {
		if rule.Type() != "switch_rule" {
			continue
		}
		caseExprs, isDefault, bodyNodes := splitSwitchRule(rule, source, ctx)

		clause := &ast.CaseClause{}
		if !isDefault {
			clause.List = caseExprs
		}
		for _, bodyNode := range bodyNodes {
			if stmts := TryParseStmts(bodyNode, source, ctx); stmts != nil {
				clause.Body = append(clause.Body, stmts...)
			} else {
				clause.Body = append(clause.Body, ParseStmt(bodyNode, source, ctx))
			}
		}
		switchBlock.List = append(switchBlock.List, clause)
	}

	return switchBlock
}

// splitSwitchRule separates a switch_rule into its case label expressions (empty
// for default), whether it is the default rule, and the body nodes after the
// arrow.
func splitSwitchRule(rule *sitter.Node, source []byte, ctx Ctx) (caseExprs []ast.Expr, isDefault bool, bodyNodes []*sitter.Node) {
	for _, child := range nodeutil.NamedChildrenOf(rule) {
		if child.Type() == "switch_label" {
			if nodeutil.SemanticNamedChildCount(child) == 0 {
				isDefault = true
			} else {
				for _, labelExpr := range nodeutil.NamedChildrenOf(child) {
					caseExprs = append(caseExprs, ParseExpr(labelExpr, source, ctx))
				}
			}
			continue
		}
		bodyNodes = append(bodyNodes, child)
	}
	return caseExprs, isDefault, bodyNodes
}

// splitSwitchGroup separates a switch_block_statement_group into its case label
// expressions (empty when the group is `default`), whether it is the default
// group, and the statement nodes that make up its body.
func splitSwitchGroup(group *sitter.Node, source []byte, ctx Ctx) (caseExprs []ast.Expr, isDefault bool, bodyNodes []*sitter.Node) {
	for _, child := range nodeutil.NamedChildrenOf(group) {
		if child.Type() == "switch_label" {
			if nodeutil.SemanticNamedChildCount(child) == 0 {
				// A `default` label has no child expression.
				isDefault = true
			} else {
				caseExprs = append(caseExprs, ParseExpr(nodeutil.SemanticNamedChild(child, 0), source, ctx))
			}
			continue
		}
		bodyNodes = append(bodyNodes, child)
	}
	return caseExprs, isDefault, bodyNodes
}

// parseSwitchGroupBody converts the statement nodes of a switch group, dropping a
// trailing `break` (Go cases break implicitly) and reporting whether the group
// was terminated by such a break so the caller can decide on fallthrough.
func parseSwitchGroupBody(bodyNodes []*sitter.Node, source []byte, ctx Ctx) (body []ast.Stmt, terminatedByBreak bool) {
	for _, stmtNode := range bodyNodes {
		if stmtNode.Type() == "break_statement" {
			// A plain `break` ends the case in Java; Go does this implicitly. A
			// labeled break is rare in switches and is preserved as-is.
			if nodeutil.SemanticNamedChildCount(stmtNode) == 0 {
				terminatedByBreak = true
				continue
			}
		}
		if stmts := TryParseStmts(stmtNode, source, ctx); stmts != nil {
			body = append(body, stmts...)
		} else {
			stmt := ParseStmt(stmtNode, source, ctx)
			body = append(body, stmt)
			if stmtNode.Type() == "local_variable_declaration" {
				body = append(body, unusedLocalDiscardStatements(stmt, stmtNode.Parent(), source)...)
			}
		}
	}
	return body, terminatedByBreak
}

func lowerTryControlBranch(node *sitter.Node, source []byte, ctx Ctx, tok token.Token) ast.Stmt {
	javaTarget, label := javaBranchTarget(node, source, tok)
	direct := lowerJavaControlTransferStmt(tok, label, javaTarget, ctx)
	if ctx.tryReturnTarget == nil || ctx.tryControlBoundary == nil {
		return direct
	}

	if javaTarget == nil || javaTargetInsideBoundary(javaTarget, ctx.tryControlBoundary) {
		return direct
	}
	transfer := ctx.tryReturnTarget.registerControlTransfer(tok, label, javaTarget)
	if transfer == nil {
		return direct
	}

	// The generated func literal cannot cross the Java target directly. Record
	// the transfer and return from the closure so resource/finally defers run.
	// Clearing a pending method return lets a branch in finally supersede it.
	return &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.FlagName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: "false"}},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.ControlName}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", transfer.Code)}},
		},
		&ast.ReturnStmt{},
	}}
}

func lowerJavaControlTransferStmt(tok token.Token, label string, javaTarget *sitter.Node, ctx Ctx) ast.Stmt {
	if tok == token.CONTINUE {
		if target := activeDoWhileContinueTarget(javaTarget, ctx); target != nil {
			target.Used = true
			return &ast.BranchStmt{
				Tok:   token.GOTO,
				Label: &ast.Ident{Name: target.Label},
			}
		}
	}

	if label != "" && javaTarget != nil {
		if key, ok := javaControlKey(javaTarget); ok {
			if target := ctx.javaLabelTargets[key]; target != nil {
				target.NeedsGoLabel = true
				if tok == token.BREAK && target.BreakLabel != "" {
					return &ast.BranchStmt{
						Tok:   token.GOTO,
						Label: &ast.Ident{Name: target.BreakLabel},
					}
				}
			}
		}
	}
	branch := &ast.BranchStmt{Tok: tok}
	if label != "" {
		branch.Label = &ast.Ident{Name: label}
	}
	return branch
}

func activeDoWhileContinueTarget(javaTarget *sitter.Node, ctx Ctx) *doWhileContinueTarget {
	for javaTarget != nil && javaTarget.Type() == "labeled_statement" {
		if nodeutil.SemanticNamedChildCount(javaTarget) < 2 {
			return nil
		}
		javaTarget = nodeutil.SemanticNamedChild(javaTarget, 1)
	}
	if javaTarget == nil || javaTarget.Type() != "do_statement" {
		return nil
	}
	key, ok := javaControlKey(javaTarget)
	if !ok {
		return nil
	}
	return ctx.doWhileContinueTargets[key]
}

func cloneDoWhileContinueTargets(source map[javaControlTargetKey]*doWhileContinueTarget) map[javaControlTargetKey]*doWhileContinueTarget {
	cloned := make(map[javaControlTargetKey]*doWhileContinueTarget, len(source)+1)
	for key, target := range source {
		cloned[key] = target
	}
	return cloned
}

func cloneJavaLabelTargets(source map[javaControlTargetKey]*javaLabelTarget) map[javaControlTargetKey]*javaLabelTarget {
	cloned := make(map[javaControlTargetKey]*javaLabelTarget, len(source)+1)
	for key, target := range source {
		cloned[key] = target
	}
	return cloned
}

func recordLocalVariableDefinition(ctx Ctx, name, originalType, parsedType string) {
	if ctx.localScope == nil || strings.TrimSpace(name) == "" {
		return
	}

	for _, existing := range ctx.localScope.Children {
		if existing == nil || existing.OriginalName != name {
			continue
		}
		existing.Type = parsedType
		existing.OriginalType = originalType
		bindDefinitionTypeParameters(existing, visibleTypeParameterDeclarations(ctx))
		existing.Name = hygienicLocalIdentifier(existing.Name, existing.OriginalName, ctx)
		return
	}

	definition := &symbol.Definition{
		OriginalName: name,
		Name:         name,
		OriginalType: originalType,
		Type:         parsedType,
	}
	bindDefinitionTypeParameters(definition, visibleTypeParameterDeclarations(ctx))
	definition.Name = hygienicLocalIdentifier(definition.Name, definition.OriginalName, ctx)
	ctx.localScope.Children = append(ctx.localScope.Children, definition)
}

func markLocalVariableNullable(ctx Ctx, name string) {
	if ctx.localScope == nil || strings.TrimSpace(name) == "" {
		return
	}
	if local := ctx.localScope.FindVariable(name); local != nil {
		local.Nullable = true
	}
}

func ParseStmts(node *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	if stmts := TryParseStmts(node, source, ctx); stmts != nil {
		return stmts
	}
	panic(fmt.Errorf("unhandled stmts type: %v", node.Type()))
}

func TryParseStmts(node *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	switch node.Type() {
	case "assignment_expression":
		if stmts, ok := ParseNode(node, source, ctx).([]ast.Stmt); ok {
			return stmts
		}
	case "try_statement", "try_with_resources_statement", "synchronized_statement":
		if stmts, ok := ParseNode(node, source, ctx).([]ast.Stmt); ok {
			return stmts
		}
	}
	return nil
}
