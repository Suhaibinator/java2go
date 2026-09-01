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
// float->float32. double/boolean already infer to the right Go type (float64,
// bool), so they are left to `:=`. The original Java type may carry array
// brackets or qualifiers; only the bare primitive name is matched.
func needsExplicitPrimitiveType(originalType string) bool {
	base, _ := parseJavaTypeString(originalType)
	switch strings.TrimSpace(base) {
	case "int", "long", "short", "byte", "char", "float":
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

// parseStatementBlock renders one Java block while optionally omitting an
// already-structured source statement. Constructor lowering uses the omission
// for its leading this(...)/super(...) invocation: that invocation controls the
// allocation/initialization phase and must not first be flattened into an
// ordinary Go statement (or parsed twice, which could duplicate hoisted
// declarations in its arguments).
func parseStatementBlock(node, omitted *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
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
		if isJavaStringType(ctx.expectedType) {
			return javaNullStringExpr()
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
	lhsNode := node.Child(0)
	operatorNode := node.Child(1)
	rhsNode := node.Child(2)
	if lhsNode == nil || lhsNode.Type() != "array_access" || operatorNode == nil || operatorNode.Content(source) != "=" || rhsNode == nil {
		return nil, false
	}

	arrayNode := lhsNode.ChildByFieldName("array")
	indexNode := lhsNode.ChildByFieldName("index")
	if arrayNode == nil && lhsNode.NamedChildCount() > 0 {
		arrayNode = lhsNode.NamedChild(0)
	}
	if indexNode == nil && lhsNode.NamedChildCount() > 1 {
		indexNode = lhsNode.NamedChild(1)
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
	if _, componentType, componentID, reified := expressionUsesReifiedReferenceArray(arrayNode, ctx, source); reified {
		return stdjavaGenericCall(ctx, "ReferenceArrayAssign", []ast.Expr{componentType}, []ast.Expr{
			ParseExpr(arrayNode, source, ctx),
			ParseExpr(indexNode, source, ctx),
			rhs,
			componentID,
		}), true
	}
	if _, componentType, _, primitive := expressionUsesPrimitiveArray(arrayNode, ctx, source); primitive {
		return stdjavaGenericCall(ctx, "PrimitiveArrayAssign", []ast.Expr{componentType}, []ast.Expr{
			ParseExpr(arrayNode, source, ctx),
			ParseExpr(indexNode, source, ctx),
			rhs,
		}), true
	}

	return stdjavaCall(ctx, "ArraySet",
		ParseExpr(arrayNode, source, ctx),
		ParseExpr(indexNode, source, ctx),
		rhs,
	), true
}

// requireNullableValueBackedExpression converts an interface-backed nullable
// String/boxed local back to the concrete Go value used at a method boundary.
// A selected null consequently raises the same kind of failure as Java
// unboxing/dereferencing; modelling nullable boxed return values themselves
// requires a repository-wide representation change beyond local storage.
func requireNullableValueBackedExpression(value ast.Expr, node *sitter.Node, expectedType string, ctx Ctx, source []byte) ast.Expr {
	if !usesNullableValueStorage(expectedType) ||
		!expressionUsesNullableValueStorage(node, ctx, source) {
		return value
	}
	base, _ := parseJavaTypeString(expectedType)
	if stripJavaQualifier(base) == "String" {
		// Returning or passing a String reference is not a dereference. Preserve
		// null through the concrete-string ABI using the runtime sentinel.
		return stdjavaCall(ctx, "StringReferenceValue", value)
	}
	return &ast.TypeAssertExpr{
		X:    value,
		Type: javaTypeStringToGoTypeExpr(expectedType, inScopeTypeParameters(ctx), ctx),
	}
}

// inferEnhancedForElementJavaType resolves the Java type inferred by `var` in
// an enhanced-for binding. Go's range statement infers the generated variable's
// type automatically, but the transpiler's Java-side expression inference also
// needs the element type for compound assignments, overload selection, and
// intrinsic dispatch inside the loop body.
func inferEnhancedForElementJavaType(valueNode *sitter.Node, source []byte, ctx Ctx) (string, bool) {
	if valueNode == nil {
		return "", false
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
		if actualPrimitive, boxed := ternaryBoxedPrimitive(componentJavaType); boxed && actualPrimitive != expectedPrimitive {
			if _, widening := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive); widening {
				if conversion := goPrimitiveConversionName(expectedPrimitive); conversion != "" {
					return &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{value}}
				}
			}
		}
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

	lhsNode := node.Child(0)
	opNode := node.Child(1)
	rhsNode := node.Child(2)
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
		return node.NamedChildCount() == 1 && isSideEffectFreeCompoundAssignmentRHS(node.NamedChild(0))
	case "unary_expression":
		count := node.NamedChildCount()
		return count > 0 && isSideEffectFreeCompoundAssignmentRHS(node.NamedChild(int(count)-1))
	case "cast_expression":
		return node.NamedChildCount() == 2 && isSideEffectFreeCompoundAssignmentRHS(node.NamedChild(1))
	case "binary_expression":
		return node.ChildCount() >= 3 &&
			isSideEffectFreeCompoundAssignmentRHS(node.Child(0)) &&
			isSideEffectFreeCompoundAssignmentRHS(node.Child(2))
	case "field_access":
		object := node.ChildByFieldName("object")
		return object != nil && isSideEffectFreeCompoundAssignmentRHS(object)
	case "array_access":
		array := node.ChildByFieldName("array")
		index := node.ChildByFieldName("index")
		if array == nil && node.NamedChildCount() > 0 {
			array = node.NamedChild(0)
		}
		if index == nil && node.NamedChildCount() > 1 {
			index = node.NamedChild(1)
		}
		return isSideEffectFreeCompoundAssignmentRHS(array) && isSideEffectFreeCompoundAssignmentRHS(index)
	default:
		return false
	}
}

func TryParseStmt(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	switch node.Type() {
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
		originalType := node.ChildByFieldName("type").Content(source)
		variableType := astutil.ParseType(node.ChildByFieldName("type"), source)
		variableDeclarator := node.ChildByFieldName("declarator")

		// If a variable is being declared, but not set to a value
		// Ex: `int value;`
		if variableDeclarator.NamedChildCount() == 1 {
			return &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{identFromNode(variableDeclarator.ChildByFieldName("name"), source)},
							Type:  explicitLocalVariableType(originalType, ctx),
						},
					},
				},
			}
		}

		ctx.lastType = variableType
		// Set expected type for diamond operator inference
		ctx.expectedType = node.ChildByFieldName("type").Content(source)
		initializerNode := variableDeclarator.ChildByFieldName("value")
		if initializerNode == nil && variableDeclarator.NamedChildCount() > 1 {
			initializerNode = variableDeclarator.NamedChild(1)
		}
		ctx.expectedTypeRoot = initializerNode

		declaration := ParseStmt(variableDeclarator, source, ctx).(*ast.AssignStmt)

		// A nullable String or boxed primitive needs an interface-backed local even
		// when null is nested inside a conditional initializer rather than appearing
		// as the declarator's direct value.
		containsNull := expressionUsesNullableValueStorage(initializerNode, ctx, source)

		names := make([]*ast.Ident, len(declaration.Lhs))
		for ind, decl := range declaration.Lhs {
			ident := decl.(*ast.Ident)
			names[ind] = ident
			recordedOriginalType := originalType
			// Java overload resolution uses a local's declared static type, not the
			// concrete type of its initializer. Only `var` declarations derive their
			// static type from the initializer.
			if isVarKeywordType(strings.TrimSpace(originalType)) && variableDeclarator.NamedChildCount() == 2 {
				if inferredType, ok := inferExprJavaType(variableDeclarator.NamedChild(1), ctx, source); ok && strings.TrimSpace(inferredType) != "" {
					recordedOriginalType = inferredType
				}
			}
			recordLocalVariableDefinition(ctx, ident.Name, recordedOriginalType, symbol.NodeToStr(variableType))
		}

		// If the declaration contains null, declare it with the `var` keyword instead
		// of implicitly
		if containsNull {
			for _, name := range names {
				if local := ctx.localScope.FindVariable(name.Name); local != nil && usesNullableValueStorage(local.OriginalType) {
					markLocalVariableNullable(ctx, name.Name)
				}
			}
			// Java `var` derives its static type from the conditional. Its generated
			// IIFE already has the necessary pointer/interface result type, so retain
			// the short declaration rather than trying to spell `var` as a Go type.
			if isVarKeywordType(strings.TrimSpace(originalType)) {
				return declaration
			}
			return &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names:  names,
							Type:   nullableLocalVariableType(originalType, ctx),
							Values: declaration.Rhs,
						},
					},
				},
			}
		}

		// A Java primitive whose Go type is narrower than the type an untyped
		// constant would infer (int->int32, long->int64, char->rune, ...) must be
		// pinned. Otherwise `int total = 0` becomes a Go `int`, losing Java's
		// 32-bit overflow wrap and clashing with int32 fields/params. Wrap each
		// initializer in the Go type conversion and keep the short declaration:
		// `total := int32(0)`. Unlike `var total int32 = 0`, the `:=` form is also
		// valid in a for-loop init, where this same case is reached.
		pinType := variableType
		pin := needsExplicitPrimitiveType(strings.TrimSpace(originalType))
		// `var x = <int expr>` carries no declared type, so infer it from the
		// initializer and pin if it is a sized integer primitive.
		if !pin && isVarKeywordType(strings.TrimSpace(originalType)) && variableDeclarator.NamedChildCount() == 2 {
			if inferred, ok := inferExprJavaType(variableDeclarator.NamedChild(1), ctx, source); ok && needsExplicitPrimitiveType(inferred) {
				pin = true
				pinType = javaTypeStringToGoTypeExpr(inferred, inScopeTypeParameters(ctx), ctx)
			}
		}
		if pin {
			for ind, rhs := range declaration.Rhs {
				declaration.Rhs[ind] = &ast.CallExpr{Fun: pinType, Args: []ast.Expr{rhs}}
			}
		}

		// A primitive stored in Object undergoes Java boxing. Pin integer literals
		// to Java's 32-bit Integer representation before Go infers a host-sized int.
		if expectedBase, _ := parseJavaTypeString(originalType); stripJavaQualifier(expectedBase) == "Object" {
			initializers := nodeutil.NamedChildrenOf(variableDeclarator)
			for ind, rhs := range declaration.Rhs {
				initializerIndex := ind*2 + 1
				if initializerIndex < len(initializers) {
					declaration.Rhs[ind] = boxPrimitiveForObject(rhs, initializers[initializerIndex], originalType, ctx, source)
				}
			}
		}

		return declaration
	case "variable_declarator":
		var names, values []ast.Expr

		// If there is only one node, then that node is just a name
		if node.NamedChildCount() == 1 {
			names = append(names, identFromNode(node.NamedChild(0), source))
		}

		// Loop through every pair of name and value
		for ind := 0; ind < int(node.NamedChildCount())-1; ind += 2 {
			names = append(names, identFromNode(node.NamedChild(ind), source))
			valueNode := node.NamedChild(ind + 1)
			value := ParseExpr(valueNode, source, ctx)
			value = projectDirectOwnerErasedExpressionForExpected(value, valueNode, ctx, source)
			if expectedType := strings.TrimSpace(ctx.expectedType); expectedType != "" && !isVarKeywordType(expectedType) {
				if actualType, known := inferExprJavaType(valueNode, ctx, source); known &&
					javaDependentTypeParameterAssignable(actualType, expectedType, ctx) {
					value = dependentTypeParameterWideningExpr(value, actualType, expectedType, ctx)
				}
			}
			values = append(values, value)
		}

		return &ast.AssignStmt{Lhs: names, Tok: token.DEFINE, Rhs: values}
	case "assignment_expression":
		if lowered, ok := lowerStaticFieldAssignment(node, source, ctx); ok {
			return &ast.ExprStmt{X: lowered}
		}
		operator := node.Child(1).Content(source)
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

		lhsNode := node.Child(0)
		rhsNode := node.Child(2)
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
		var operandNode *sitter.Node
		if node.Child(0).IsNamed() {
			operandNode = node.Child(0)
		} else {
			operandNode = node.Child(1)
		}
		if _, ok := resolveStaticFieldAccess(operandNode, source, ctx); ok {
			return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
		}
		if node.Child(0).IsNamed() {
			return &ast.IncDecStmt{
				X:   ParseExpr(node.Child(0), source, ctx),
				Tok: StrToToken(node.Child(1).Content(source)),
			}
		}

		return &ast.IncDecStmt{
			X:   ParseExpr(node.Child(1), source, ctx),
			Tok: StrToToken(node.Child(0).Content(source)),
		}
	case "resource_specification":
		return ParseStmt(node.NamedChild(0), source, ctx)
	case "resource":
		var offset int
		if node.NamedChild(0).Type() == "modifiers" {
			offset = 1
		}
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ParseExpr(node.NamedChild(1+offset), source, ctx)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ParseExpr(node.NamedChild(2+offset), source, ctx)},
		}
	case "method_invocation":
		return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
	case "constructor_body", "block":
		return parseStatementBlock(node, nil, source, ctx)
	case "expression_statement":
		if stmt := TryParseStmt(node.NamedChild(0), source, ctx); stmt != nil {
			return stmt
		}
		return &ast.ExprStmt{X: ParseExpr(node.NamedChild(0), source, ctx)}
	case "explicit_constructor_invocation":
		// This is when a constructor calls another constructor with the use of
		// something such as `this(args...)`
		argsNode := node.ChildByFieldName("arguments")
		if argsNode == nil && node.NamedChildCount() > 1 {
			argsNode = node.NamedChild(1)
		}
		var args []ast.Expr
		if argsNode != nil {
			args = ParseNode(argsNode, source, ctx).([]ast.Expr)
		}

		constructorNode := node.ChildByFieldName("constructor")
		if constructorNode == nil && node.NamedChildCount() > 0 {
			constructorNode = node.NamedChild(0)
		}
		if constructorNode != nil && constructorNode.Type() == "super" && ctx.currentClass != nil {
			superType := strings.TrimSpace(ctx.currentClass.Superclass)
			if superType != "" {
				base, superArgStrs := parseJavaTypeString(superType)
				if base != "" {
					superName := stripJavaQualifier(base)
					// A built-in exception superclass is constructed via the stdjava
					// runtime; the embedded field is named after the runtime type.
					if isBuiltinExceptionType(superName) && resolveClassScopeByQualifiedName(ctx, base) == nil {
						recvName := ctx.className
						if recvName == "" && ctx.currentClass.Class != nil {
							recvName = ctx.currentClass.Class.Name
						}
						if recvName != "" {
							return &ast.AssignStmt{
								Lhs: []ast.Expr{&ast.SelectorExpr{
									X:   &ast.Ident{Name: ShortName(recvName)},
									Sel: &ast.Ident{Name: superName},
								}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{&ast.CallExpr{
									Fun:  stdjavaQualifiedExpr("New"+superName, ctx),
									Args: args,
								}},
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
								Sel: &ast.Ident{Name: superName},
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
			if ctx.tryReturnTarget.HasValue && node.NamedChildCount() > 0 {
				returnCtx := ctx.Clone()
				if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
					returnCtx.expectedType = ctx.localScope.OriginalType
					returnCtx.expectedTypeRoot = node.NamedChild(0)
				}
				stmts = append(stmts, &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.ValueName}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{parseReturnValue(node.NamedChild(0), source, returnCtx)},
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

		if node.NamedChildCount() < 1 {
			return replayedMethodReturnStmt(ctx, false, "")
		}
		returnCtx := ctx.Clone()
		if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
			returnCtx.expectedType = ctx.localScope.OriginalType
			returnCtx.expectedTypeRoot = node.NamedChild(0)
		}
		return &ast.ReturnStmt{Results: []ast.Expr{parseReturnValue(node.NamedChild(0), source, returnCtx)}}
	case "labeled_statement":
		return lowerLabeledStatement(node, source, ctx)
	case "break_statement":
		return lowerTryControlBranch(node, source, ctx, token.BREAK)
	case "continue_statement":
		return lowerTryControlBranch(node, source, ctx, token.CONTINUE)
	case "throw_statement":
		return &ast.ExprStmt{X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "panic"},
			Args: []ast.Expr{ParseExpr(node.NamedChild(0), source, ctx)},
		}}
	case "if_statement":
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
			Cond: ParseExpr(node.ChildByFieldName("condition"), source, ctx),
			Body: body.(*ast.BlockStmt),
			Else: other,
		}
	case "enhanced_for_statement":
		// An enhanced for statement has the following fields:
		// variables for the variable being declared (ex: int n)
		// then the expression that is being ranged over
		// and finally, the block of the expression

		total := int(node.NamedChildCount())
		typeNode := node.ChildByFieldName("type")
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")
		bodyNode := node.ChildByFieldName("body")

		// Fallback for grammars that don't provide named fields.
		if nameNode == nil && total >= 3 {
			nameNode = node.NamedChild(total - 3)
		}
		if valueNode == nil && total >= 2 {
			valueNode = node.NamedChild(total - 2)
		}
		if bodyNode == nil && total >= 1 {
			bodyNode = node.NamedChild(total - 1)
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
			// stdjava List/Set are pointer types, not slices, so an enhanced-for
			// over them ranges over their Slice() view instead.
			if collectionNeedsSliceForRange(valueNode, ctx, source) {
				rangeExpr = &ast.CallExpr{
					Fun: &ast.SelectorExpr{X: rangeExpr, Sel: &ast.Ident{Name: "Slice"}},
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
			init = ParseStmt(initNode, source, ctx)
		}
		loopCtx, affineBindings := prepareAffineArrayLoop(node, source, ctx)
		if node.ChildByFieldName("update") != nil {
			post = ParseStmt(node.ChildByFieldName("update"), source, ctx)
		}
		var cond ast.Expr
		if node.ChildByFieldName("condition") != nil {
			cond = ParseExpr(node.ChildByFieldName("condition"), source, ctx)
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

		body := ParseStmt(node.ChildByFieldName("body"), source, fastCtx).(*ast.BlockStmt)
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

		guardedBody := ParseStmt(node.ChildByFieldName("body"), source, guardedCtx).(*ast.BlockStmt)
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
			Cond: ParseExpr(node.NamedChild(0), source, ctx),
			Body: ParseStmt(node.NamedChild(1), source, ctx).(*ast.BlockStmt),
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
		body := ParseStmt(node.NamedChild(0), source, doCtx).(*ast.BlockStmt)

		conditionGuard := &ast.IfStmt{
			Cond: &ast.UnaryExpr{
				Op: token.NOT,
				X: &ast.ParenExpr{
					X: ParseExpr(node.NamedChild(1), source, ctx),
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
		return &ast.SwitchStmt{
			Tag:  ParseExpr(tagNode, source, ctx),
			Body: parseSwitchBlock(blockNode, source, ctx),
		}
	case "switch_block":
		return parseSwitchBlock(node, source, ctx)
	}
	return nil
}

// lowerBufferedReaderReadLineWhile recognizes Java's canonical nullable line
// loop and uses the runtime's (string, presence) bridge:
//
//	while ((line = reader.readLine()) != null) { ... }
//	for reader.ReadLineInto(&line) { ... }
//
// Java String is represented as a Go string, so a direct translation cannot
// compare the read result with nil. Keeping the rewrite at the loop boundary
// preserves empty-line versus EOF behavior and evaluates the reader once per
// condition check.
func lowerBufferedReaderReadLineWhile(node *sitter.Node, source []byte, ctx Ctx) (ast.Stmt, bool) {
	if node == nil || node.Type() != "while_statement" || node.NamedChildCount() < 2 {
		return nil, false
	}

	condition := unwrapParenthesizedExpressionNode(node.NamedChild(0))
	if condition == nil || condition.Type() != "binary_expression" || condition.ChildCount() < 3 {
		return nil, false
	}
	if operator := condition.Child(1); operator == nil || operator.Content(source) != "!=" {
		return nil, false
	}

	left := unwrapParenthesizedExpressionNode(condition.Child(0))
	right := unwrapParenthesizedExpressionNode(condition.Child(2))
	var assignment *sitter.Node
	switch {
	case left != nil && left.Type() == "assignment_expression" && right != nil && right.Type() == "null_literal":
		assignment = left
	case right != nil && right.Type() == "assignment_expression" && left != nil && left.Type() == "null_literal":
		assignment = right
	default:
		return nil, false
	}
	if assignment.ChildCount() < 3 || assignment.Child(1).Content(source) != "=" {
		return nil, false
	}

	targetNode := assignment.Child(0)
	readCall := unwrapParenthesizedExpressionNode(assignment.Child(2))
	if targetNode == nil || readCall == nil || readCall.Type() != "method_invocation" {
		return nil, false
	}
	nameNode := readCall.ChildByFieldName("name")
	receiverNode := readCall.ChildByFieldName("object")
	if nameNode == nil || nameNode.Content(source) != "readLine" || receiverNode == nil {
		return nil, false
	}
	receiverType, ok := inferExprJavaType(receiverNode, ctx, source)
	if !ok || stripJavaQualifier(receiverType) != "BufferedReader" {
		return nil, false
	}

	conditionExpr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ParseExpr(receiverNode, source, ctx),
			Sel: &ast.Ident{Name: "ReadLineInto"},
		},
		Args: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: ParseExpr(targetNode, source, ctx)}},
	}
	return &ast.ForStmt{
		Cond: conditionExpr,
		Body: ParseStmt(node.NamedChild(1), source, ctx).(*ast.BlockStmt),
	}, true
}

func unwrapParenthesizedExpressionNode(node *sitter.Node) *sitter.Node {
	for node != nil && node.Type() == "parenthesized_expression" && node.NamedChildCount() > 0 {
		node = node.NamedChild(0)
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
// explicit type. String and boxed primitives need an interface slot when their
// initializer is null so later comparisons and Java text conversion can still
// observe the distinction between null and a value-type zero.
func nullableLocalVariableType(originalType string, ctx Ctx) ast.Expr {
	if usesNullableValueStorage(originalType) {
		return &ast.Ident{Name: "any"}
	}
	return explicitLocalVariableType(originalType, ctx)
}

func usesNullableValueStorage(originalType string) bool {
	base, _ := parseJavaTypeString(originalType)
	switch stripJavaQualifier(base) {
	case "String", "Integer", "Long", "Short", "Byte", "Character", "Float", "Double", "Boolean":
		return true
	default:
		return false
	}
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
		if sameNode(parent.ChildByFieldName("name")) || (parent.Type() == "variable_declarator" && sameNode(parent.NamedChild(0))) {
			return false
		}
	case "enhanced_for_statement":
		if sameNode(parent.ChildByFieldName("name")) {
			return false
		}
	case "assignment_expression":
		if sameNode(parent.Child(0)) && parent.Child(1) != nil && parent.Child(1).Content(source) == "=" {
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
	for condNode.Type() == "parenthesized_expression" && condNode.NamedChildCount() > 0 {
		condNode = condNode.NamedChild(0)
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

	initStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: bindName}, &ast.Ident{Name: "ok"}},
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
			if child.NamedChildCount() == 0 {
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
			if child.NamedChildCount() == 0 {
				// A `default` label has no child expression.
				isDefault = true
			} else {
				caseExprs = append(caseExprs, ParseExpr(child.NamedChild(0), source, ctx))
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
			if stmtNode.NamedChildCount() == 0 {
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
		if javaTarget.NamedChildCount() < 2 {
			return nil
		}
		javaTarget = javaTarget.NamedChild(1)
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
		return
	}

	definition := &symbol.Definition{
		OriginalName: name,
		Name:         name,
		OriginalType: originalType,
		Type:         parsedType,
	}
	bindDefinitionTypeParameters(definition, visibleTypeParameterDeclarations(ctx))
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
