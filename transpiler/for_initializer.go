package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// parseForInitializer preserves Java's left-to-right declarations and the scope
// of every for-local. A Go parallel declaration cannot represent int i=f(),
// j=i+g(): its RHS expressions cannot refer to the newly declared i. The IIFE
// evaluates each original initializer once, in order, then returns all locals
// into the real Go for initializer. Keeping a real for initializer also preserves
// labeled break/continue and keeps the locals out of the following statement.
func parseForInitializer(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	if node.Type() != "local_variable_declaration" {
		return ParseStmt(node, source, ctx)
	}
	var declarators []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	}
	if len(declarators) < 2 {
		return ParseStmt(node, source, ctx)
	}
	body := &ast.BlockStmt{}
	var names, values []ast.Expr
	var results []*ast.Field
	originalType := node.ChildByFieldName("type").Content(source)
	for _, declarator := range declarators {
		body.List = append(body.List, parseLocalVariableDeclarator(node, declarator, source, ctx))
		sourceName := declarator.ChildByFieldName("name").Content(source)
		name := localBindingIdent(declarator.ChildByFieldName("name"), source, ctx)
		names = append(names, name)
		values = append(values, ast.NewIdent(name.Name))
		resultType := explicitLocalVariableType(originalType, ctx)
		if ctx.localScope != nil {
			if local := ctx.localScope.FindVariable(sourceName); local != nil && local.Nullable {
				resultType = nullableLocalVariableType(originalType, ctx)
			}
		}
		results = append(results, &ast.Field{Type: resultType})
	}
	body.List = append(body.List, &ast.ReturnStmt{Results: values})
	initializer := &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: results}}, Body: body}}
	return &ast.AssignStmt{Lhs: names, Tok: token.DEFINE, Rhs: []ast.Expr{initializer}}
}

// parseLocalVariableDeclarator shares the existing local-initialization rules
// between ordinary declarations and each declarator of a for initializer.
func parseLocalVariableDeclarator(node, variableDeclarator *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	originalType := node.ChildByFieldName("type").Content(source)
	variableType := astutil.ParseType(node.ChildByFieldName("type"), source)
	sourceName := variableDeclarator.ChildByFieldName("name").Content(source)
	recordLocalVariableDefinition(ctx, sourceName, originalType, symbol.NodeToStr(variableType))

	// If a variable is being declared, but not set to a value
	// Ex: `int value;`
	if nodeutil.SemanticNamedChildCount(variableDeclarator) == 1 {
		return &ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{localBindingIdent(variableDeclarator.ChildByFieldName("name"), source, ctx)},
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
	if initializerNode == nil && nodeutil.SemanticNamedChildCount(variableDeclarator) > 1 {
		initializerNode = nodeutil.SemanticNamedChild(variableDeclarator, 1)
	}
	ctx.expectedTypeRoot = initializerNode

	declaration := ParseStmt(variableDeclarator, source, ctx).(*ast.AssignStmt)

	// Nullable initializers need an explicit type. String locals retain their
	// interface storage while wrapper references use their ordinary pointer.
	containsNull := expressionUsesNullableValueStorage(initializerNode, ctx, source)

	names := make([]*ast.Ident, len(declaration.Lhs))
	for ind, decl := range declaration.Lhs {
		ident := decl.(*ast.Ident)
		names[ind] = ident
		recordedOriginalType := originalType
		// Java overload resolution uses a local's declared static type, not the
		// concrete type of its initializer. Only `var` declarations derive their
		// static type from the initializer.
		if isVarKeywordType(strings.TrimSpace(originalType)) && nodeutil.SemanticNamedChildCount(variableDeclarator) == 2 {
			if inferredType, ok := inferExprJavaType(nodeutil.SemanticNamedChild(variableDeclarator, 1), ctx, source); ok && strings.TrimSpace(inferredType) != "" {
				recordedOriginalType = inferredType
			}
		}
		recordLocalVariableDefinition(ctx, sourceName, recordedOriginalType, symbol.NodeToStr(variableType))
	}

	// If the declaration contains null, declare it with the `var` keyword instead
	// of implicitly
	if containsNull {
		for range names {
			if local := ctx.localScope.FindVariable(sourceName); local != nil && usesNullableValueStorage(local.OriginalType) {
				markLocalVariableNullable(ctx, sourceName)
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
	if pin && strings.TrimSpace(originalType) == "double" && initializerNode != nil {
		// An untyped floating constant and every already-double expression infer
		// float64 correctly. Only integral or otherwise non-double initializers
		// need an explicit conversion for a Java double local.
		if inferred, ok := inferExprJavaType(initializerNode, ctx, source); ok {
			if canonical, numeric := canonicalJavaNumericType(inferred); numeric && canonical == "double" {
				pin = false
			}
		}
	}
	// `var x = <int expr>` carries no declared type, so infer it from the
	// initializer and pin if it is a sized integer primitive.
	if !pin && isVarKeywordType(strings.TrimSpace(originalType)) && nodeutil.SemanticNamedChildCount(variableDeclarator) == 2 {
		if inferred, ok := inferExprJavaType(nodeutil.SemanticNamedChild(variableDeclarator, 1), ctx, source); ok && needsExplicitPrimitiveType(inferred) {
			pin = true
			pinType = javaTypeStringToGoTypeExpr(inferred, inScopeTypeParameters(ctx), ctx)
		}
	}
	if pin {
		for ind, rhs := range declaration.Rhs {
			declaration.Rhs[ind] = &ast.CallExpr{Fun: pinType, Args: []ast.Expr{rhs}}
		}
	}

	// A lambda assigned to a built-in functional interface must carry that
	// interface's named runtime type rather than the unnamed func type Go
	// would infer from the literal, or the interface's default methods
	// (Comparator.reversed, thenComparing, compare) are unavailable on it.
	if named := namedFunctionalInterfaceTypeExpr(originalType, ctx); named != nil {
		for ind, rhs := range declaration.Rhs {
			if _, isFuncLit := rhs.(*ast.FuncLit); isFuncLit {
				declaration.Rhs[ind] = &ast.CallExpr{Fun: named, Args: []ast.Expr{rhs}}
			}
		}
	}

	// Throwable subclasses have concrete constructors but Java variables keep
	// their declared reference type, including across later subtype assignments.
	if _, builtinThrowable := builtinThrowableReferenceName(originalType, ctx); builtinThrowable {
		return &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
			&ast.ValueSpec{Names: names, Type: explicitLocalVariableType(originalType, ctx), Values: declaration.Rhs},
		}}}
	}

	base, _ := parseJavaTypeString(originalType)
	switch stripJavaQualifier(base) {
	case "Object", "Number", "Comparable", "Serializable", "Constable", "ConstantDesc":
		if resolveClassScopeByQualifiedName(ctx, base) == nil {
			return &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
				&ast.ValueSpec{Names: names, Type: explicitLocalVariableType(originalType, ctx), Values: declaration.Rhs},
			}}}
		}
	}

	return declaration
}
