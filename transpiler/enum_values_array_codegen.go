package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// The synthetic member remains a declaration-owned logical E[] signature.
// Its final allocated Go name is shared by selection and the physical producer.
func enumSyntheticValuesDefinition(scope *symbol.ClassScope) *symbol.Definition {
	if scope == nil || !scope.IsEnum {
		return nil
	}
	for _, def := range scope.Methods {
		if def != nil && def.OriginalName == "values" && def.IsStatic && def.RuntimeDefault && def.DeclarationNode == nil && len(def.Parameters) == 0 {
			return def
		}
	}
	return nil
}

func enumValuesArrayDeclarations(storage string, ctx Ctx) []ast.Decl {
	def := enumSyntheticValuesDefinition(ctx.currentClass)
	if def == nil {
		panic("enum values signature was not prepared")
	}
	used := affineLoopUsedNames(ctx.currentClass.Class.DeclarationNode, ctx.currentFile.Source, ctx)
	arrayName := synchronizedUniqueLocalName("__java2goEnumValuesArray", used)
	indexName := synchronizedUniqueLocalName("__java2goEnumValuesIndex", used)
	valueName := synchronizedUniqueLocalName("__java2goEnumValuesElement", used)
	result := &ast.StarExpr{X: stdjavaQualifiedExpr("ReferenceArray", ctx)}
	component := &ast.StarExpr{X: ast.NewIdent(ctx.className)}
	body := []ast.Stmt{
		classInitializationEnsureStmt(ctx.currentClass, ast.NewIdent(executionNameForClass(ctx.currentClass)), ctx),
		&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent(storage), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", stdjavaCall(ctx, "NewJavaNullPointerExceptionMessage", stdjavaCall(ctx, "JavaStringFromHostUTF8", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(enumValuesUninitializedMessage(ctx.currentClass))})))}}}},
		&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(arrayName)}, Tok: token.DEFINE, Rhs: []ast.Expr{stdjavaGenericCall(ctx, "NewReferenceArrayOf", []ast.Expr{component}, []ast.Expr{callIdent("len", ast.NewIdent(storage)), javaTypeIDLiteral(sourceClassRuntimeTypeID(ctx.currentClass, ctx), ctx)})}},
		&ast.RangeStmt{Key: ast.NewIdent(indexName), Value: ast.NewIdent(valueName), Tok: token.DEFINE, X: ast.NewIdent(storage), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: stdjavaCall(ctx, "ReferenceArraySet", ast.NewIdent(arrayName), ast.NewIdent(indexName), ast.NewIdent(valueName))}}}},
		&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent(arrayName)}},
	}
	fn := &ast.FuncDecl{Name: ast.NewIdent(def.Name), Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: result}}}}, Body: &ast.BlockStmt{List: body}}
	if def.Name == ctx.className+"Values" {
		return []ast.Decl{fn}
	} // Existing enum wrapper pass handles the ordinary spelling.
	return buildExecutionAwareFuncDecls(fn, executionImplementationName(def, ctx.currentClass, ctx), executionNameForClass(ctx.currentClass), ctx)
}

func enumSyntheticValuesSelection(node *sitter.Node, ctx Ctx, source []byte) (*methodResolution, *sitter.Node) {
	if node == nil || node.ChildByFieldName("name") == nil || node.ChildByFieldName("name").Content(source) != "values" || invocationArgumentCount(node) != 0 {
		return nil, nil
	}
	object := node.ChildByFieldName("object")
	var selected *methodResolution
	var qualifier *sitter.Node
	args := node.ChildByFieldName("arguments")
	if object != nil {
		if scope := resolveClassScopeByIdentifier(ctx, source, object); scope != nil {
			selected = findBestMethodInHierarchy(scope, "values", args, false, true, ctx, source)
		} else if target := resolveInvocationTarget(object, ctx, source); target != nil {
			selected, _ = findBestMethodForInvocationTarget(target, "values", args, true, true, ctx, source)
			qualifier = object
		}
	} else {
		selected = findBestMethodInHierarchy(ctx.currentClass, "values", args, ctx.localScope != nil && !ctx.localScope.IsStatic, true, ctx, source)
		if selected == nil {
			selected = findEnclosingStaticMethod("values", args, ctx, source)
		}
		if selected == nil {
			selected = resolveStaticImportedMethod(node, ctx, source).source
		}
	}
	if selected == nil || selected.def == nil || selected.def != enumSyntheticValuesDefinition(selected.owner) {
		return nil, nil
	}
	return selected, qualifier
}

func enumSyntheticValuesInvocation(node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	selected, qualifier := enumSyntheticValuesSelection(node, ctx, source)
	if selected == nil {
		return nil
	}
	call := &ast.CallExpr{Fun: qualifiedNameExpr(executionImplementationName(selected.def, selected.owner, ctx), findJavaPackageForClassScope(selected.owner), ctx), Args: []ast.Expr{intrinsicExecutionExpr(ctx)}}
	if qualifier == nil {
		return call
	}
	// Java evaluates an expression qualifier before static invocation, even null.
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ReferenceArray", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ParseExpr(qualifier, source, ctx)}}, &ast.ReturnStmt{Results: []ast.Expr{call}}}}}}
}

// Match javac's declaration-owned synthetic backing name for the enhanced
// null diagnostic. Allocation still uses the private compiler slice; no Java
// object is substituted for its constants or returned arrays.
func enumValuesUninitializedMessage(scope *symbol.ClassScope) string {
	used := make(map[string]bool)
	for _, field := range scope.Fields {
		if field != nil {
			used[field.OriginalName] = true
		}
	}
	for _, method := range scope.Methods {
		// javac records constructors under <init>, not their source type name.
		if method != nil && !method.Constructor {
			used[method.OriginalName] = true
		}
	}
	for _, constant := range scope.EnumConstants {
		used[constant.Name] = true
	}
	for _, member := range scope.Subclasses {
		if member != nil && member.Class != nil {
			used[member.Class.OriginalName] = true
		}
	}
	backing := "$VALUES"
	for used[backing] {
		backing += "$"
	}
	binary := javaClassBinaryName(scope)
	return "Cannot invoke \"[L" + binary + ";.clone()\" because \"" + binary + "." + backing + "\" is null"
}
