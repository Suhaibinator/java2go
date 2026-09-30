package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func enumMetadataFieldName(scope *symbol.ClassScope) string {
	return collisionSafeExecutionIdentifier("__java2goEnumMetadata", scope)
}
func enumMetadataField(ctx Ctx) *ast.Field {
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(enumMetadataFieldName(ctx.currentClass))}, Type: stdjavaQualifiedExpr("EnumMetadata", ctx)}
}
func enumMetadataValue(receiver ast.Expr, ctx Ctx) ast.Expr {
	return &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(enumMetadataFieldName(ctx.currentClass))}
}
func enumReferenceIdentityDecls(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	recv := ShortName(scope.Class.Name)
	receiver := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(recv)}, Type: &ast.StarExpr{X: ast.NewIdent(scope.Class.Name)}}}}
	info := enumMetadataValue(ast.NewIdent(recv), ctx)
	return []ast.Decl{
		&ast.FuncDecl{Name: ast.NewIdent("JavaEnumMetadata"), Recv: receiver, Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("EnumMetadata", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(recv), &ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: info}}}}}},
		&ast.FuncDecl{Name: ast.NewIdent("JavaDynamicTypeID"), Recv: cloneFieldList(receiver), Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("TypeID", ctx)}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: info, Sel: ast.NewIdent("DynamicTypeID")}}}}}}},
	}
}

// Constant bodies are anonymous subclasses. Count anonymous creation sites in
// source order within this lexical class, excluding nested named declarations.
func enumConstantDynamicType(scope *symbol.ClassScope, constant symbol.EnumConstant) string {
	declaring := javaClassBinaryName(scope)
	if constant.Body == nil {
		return declaring
	}
	ordinal := 0
	found := 0
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil || found != 0 {
			return
		}
		if node != scope.Class.DeclarationNode {
			switch node.Type() {
			case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
				return
			}
		}
		anonymous := node.Type() == "enum_constant" && node.ChildByFieldName("body") != nil
		if node.Type() == "object_creation_expression" {
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if child.Type() == "class_body" {
					anonymous = true
				}
			}
		}
		if anonymous {
			ordinal++
			body := node.ChildByFieldName("body")
			if body != nil && body.StartByte() == constant.Body.StartByte() {
				found = ordinal
				return
			}
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(scope.Class.DeclarationNode)
	return declaring + "$" + strconv.Itoa(found)
}
func enumConstantMetadata(constant symbol.EnumConstant, ordinal ast.Expr, ctx Ctx) ast.Expr {
	return stdjavaCall(ctx, "NewEnumMetadata", metadataString(constant.Name), ordinal, javaTypeIDLiteral(javaClassBinaryName(ctx.currentClass), ctx), javaTypeIDLiteral(enumConstantDynamicType(ctx.currentClass, constant), ctx))
}
func enumConstructorOptions(node *sitter.Node, source []byte, ctx Ctx) constructorLoweringOptions {
	name := synchronizedUniqueLocalName("__java2goEnumMetadataArgument", affineLoopUsedNames(node, source, ctx))
	recv := ast.NewIdent(ShortName(ctx.className))
	value := ast.NewIdent(name)
	return constructorLoweringOptions{
		leadingParams:   []*ast.Field{{Names: []*ast.Ident{value}, Type: stdjavaQualifiedExpr("EnumMetadata", ctx)}},
		leadingThisArgs: []ast.Expr{value},
		terminalPreSuper: []ast.Stmt{
			&ast.AssignStmt{Lhs: []ast.Expr{enumMetadataValue(recv, ctx)}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}},
			&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: recv, Sel: ast.NewIdent(enumMetaNameField)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: value, Sel: ast.NewIdent("Name")}}}},
			&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: recv, Sel: ast.NewIdent(enumMetaOrdinalField)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: value, Sel: ast.NewIdent("Ordinal")}}}},
		},
	}
}
func enumConstantRegistrationStmts(scope *symbol.ClassScope, ctx Ctx) []ast.Stmt {
	var stmts []ast.Stmt
	for _, constant := range scope.EnumConstants {
		if constant.Body == nil {
			continue
		}
		id := javaTypeIDLiteral(enumConstantDynamicType(scope, constant), ctx)
		stmts = append(stmts, &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterJavaType", id, javaTypeIDLiteral(javaClassBinaryName(scope), ctx))}, &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterJavaSourceType", id)}, &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterClassDescriptor", &ast.CompositeLit{Type: stdjavaQualifiedExpr("ClassDescriptor", ctx), Elts: []ast.Expr{metadataKey("Type", id), metadataKey("SimpleName", metadataString("")), metadataKey("HasSimpleName", ast.NewIdent("true"))}})})
	}
	return stmts
}

func enumInitializedExpr(scope *symbol.ClassScope, value, result ast.Expr, ctx Ctx) ast.Expr {
	execution := intrinsicExecutionExpr(ctx)
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: result}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{classInitializationEnsureStmt(scope, execution, ctx), &ast.ReturnStmt{Results: []ast.Expr{value}}}}}}
}
func enumInitializationStatements(ctx Ctx, source []byte, executionName string) []ast.Stmt {
	initialization := ctx.Clone()
	initialization.executionContextName = executionName
	var statements []ast.Stmt
	var values []ast.Expr
	for index, constant := range ctx.currentClass.EnumConstants {
		statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(constant.EmittedName())}, Tok: token.ASSIGN, Rhs: []ast.Expr{buildEnumConstantInitializer(constant, &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(index)}, initialization, source)}})
		values = append(values, ast.NewIdent(constant.EmittedName()))
	}
	statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_" + symbol.Lowercase(ctx.className) + "Values")}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.CompositeLit{Type: &ast.ArrayType{Elt: &ast.StarExpr{X: ast.NewIdent(ctx.className)}}, Elts: values}}})
	return statements
}
func enumInstanceInitializerDecls(body *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	var initializers []ast.Stmt
	initialization := ctx.Clone()
	initialization.localScope = &symbol.Definition{OriginalName: fieldInitMethodName, Name: fieldInitMethodName}
	initialization.executionContextName = executionNameForClass(ctx.currentClass)
	var scan func(*sitter.Node)
	scan = func(node *sitter.Node) {
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if child.Type() == "enum_body_declarations" {
				scan(child)
				continue
			}
			if child.Type() == "block" {
				initializers = append(initializers, ParseStmt(child, source, initialization))
				continue
			}
			if child.Type() != "field_declaration" {
				continue
			}
			for _, declarator := range nodeutil.VariableDeclarators(child) {
				field := fieldDefinitionForDeclarator(ctx.currentClass, declarator, source)
				value := declarator.ChildByFieldName("value")
				if field == nil || field.IsStatic || value == nil {
					continue
				}
				valueCtx := initialization.Clone()
				valueCtx.expectedType = field.OriginalType
				valueCtx.expectedTypeRoot = value
				expr := coerceArgumentToExpectedType(ParseExpr(value, source, valueCtx), value, field.OriginalType, valueCtx, source)
				initializers = append(initializers, &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(ShortName(ctx.className)), Sel: ast.NewIdent(field.Name)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{expr}})
			}
		}
	}
	scan(body)
	ctx.currentClass.HasInstanceFieldInitializers = len(initializers) > 0
	return buildInstanceFieldInitializerMethodDecl(ctx, initializers)
}

func enumReflectionFieldDescriptors(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	var fields []ast.Expr
	getter := func(value ast.Expr) ast.Expr {
		return &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{value}}}}}
	}
	for _, constant := range scope.EnumConstants {
		fields = append(fields, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Name", metadataString(constant.Name)), metadataKey("GoName", metadataGoName(constant.EmittedName())), metadataKey("Type", javaTypeIDLiteral(javaClassBinaryName(scope), ctx)), metadataKey("Final", ast.NewIdent("true")), metadataKey("EnumConstant", ast.NewIdent("true")), metadataKey("StaticGet", getter(ast.NewIdent(constant.EmittedName())))}})
	}
	for _, field := range scope.Fields {
		id, ok := javaTypeDescriptorExpr(field.OriginalType, ctx)
		if !ok {
			continue
		}
		values := []ast.Expr{metadataKey("Name", metadataString(field.OriginalName)), metadataKey("GoName", metadataGoName(field.Name)), metadataKey("Type", id), metadataKey("Final", ast.NewIdent(strconv.FormatBool(field.IsFinal))), metadataKey("NonPublic", ast.NewIdent(strconv.FormatBool(!reflectionPublic(field, scope))))}
		if field.IsStatic {
			values = append(values, metadataKey("StaticGet", getter(ast.NewIdent(field.Name))))
		}
		fields = append(fields, &ast.CompositeLit{Elts: values})
	}
	return fields
}
func enumStaticExecutionName(scope *symbol.ClassScope, name string) string {
	return collisionSafeExecutionIdentifier(scope.Class.Name+name+executionMethodSuffix, scope)
}
func enumStaticExecutionDecls(declarations []ast.Decl, ctx Ctx) []ast.Decl {
	var result []ast.Decl
	for _, declaration := range declarations {
		if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Recv == nil && (fn.Name.Name == ctx.className+"Values" || fn.Name.Name == ctx.className+"ValueOf") {
			name := "Values"
			if fn.Name.Name == ctx.className+"ValueOf" {
				name = "ValueOf"
			}
			result = append(result, buildExecutionAwareFuncDecls(fn, enumStaticExecutionName(ctx.currentClass, name), executionNameForClass(ctx.currentClass), ctx)...)
		} else {
			result = append(result, declaration)
		}
	}
	return result
}

func enumConstantNamed(scope *symbol.ClassScope, name string) bool {
	for _, constant := range scope.EnumConstants {
		if constant.Name == name {
			return true
		}
	}
	return false
}

// Synthetic enum operations must not preempt user-declared static overloads.
func enumSyntheticStaticCall(scope *symbol.ClassScope, name string, arguments *sitter.Node, ctx Ctx, source []byte) bool {
	count := 0
	if arguments != nil {
		count = int(arguments.NamedChildCount())
	}
	if name == "values" {
		return count == 0
	}
	if name != "valueOf" || count != 1 {
		return false
	}
	resolution := findBestMethodInHierarchy(scope, name, arguments, false, true, ctx, source)
	return resolution != nil && resolution.def.DeclarationNode == nil
}

// Canonical names join named member declarations with dots. Local or anonymous
// enclosing declarations have no canonical name; Java concatenation prints null.
// Original names retain legal dollar characters within an identifier.
func javaSourceClassCanonicalName(scope *symbol.ClassScope) string {
	for current := scope; current != nil && current.Class != nil; current = current.Enclosing {
		for node := current.Class.DeclarationNode; node != nil; node = node.Parent() {
			switch node.Type() {
			case "method_declaration", "constructor_declaration", "lambda_expression", "object_creation_expression", "enum_constant":
				return "null"
			}
		}
	}
	parts := []string{}
	for current := scope; current != nil && current.Class != nil; current = current.Enclosing {
		parts = append([]string{current.Class.OriginalName}, parts...)
	}
	name := strings.Join(parts, ".")
	if file := findFileScopeForClassScope(scope); file != nil && strings.TrimSpace(file.Package) != "" {
		name = file.Package + "." + name
	}
	return name
}
