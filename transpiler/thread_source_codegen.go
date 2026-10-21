package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/token"
)

func threadSourceNativeViewExpr(scope *symbol.ClassScope, receiver ast.Expr, ctx Ctx) ast.Expr {
	root := threadSourceRoot(scope, ctx)
	if root == nil || receiver == nil {
		return nil
	}
	source := sourceClassViewExpr(scope, root, receiver, ctx)
	if source == nil {
		return nil
	}
	return &ast.SelectorExpr{X: source, Sel: ast.NewIdent(superclassEmbeddedSelectorName(root, ctx))}
}
func threadSourceBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if threadSourceRoot(scope, ctx) == nil {
		return nil
	}
	used := map[string]struct{}{}
	for _, name := range scope.GoTypeParameterNames() {
		used[name] = struct{}{}
	}
	recvName := synchronizedUniqueLocalName("__java2goThreadSource", used)
	executionName := synchronizedUniqueLocalName("__java2goThreadExecution", used)
	recvType := &ast.StarExpr{X: instantiateGenericType(scope.Class.Name, typeParamExprs(scope.GoTypeParameterNames()))}
	recv := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(recvName)}, Type: recvType}}}
	var elts []ast.Expr
	for _, name := range []string{"interrupt", "isInterrupted", "run", "start"} {
		method, owner := threadSourceSelectedOverride(scope, name, ctx)
		if method == nil {
			continue
		}
		signature := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(executionName)}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)}}}}}
		call := methodCall(sourceClassViewExpr(scope, owner, ast.NewIdent(recvName), ctx), executionImplementationName(method, owner, ctx), ast.NewIdent(executionName))
		statement := ast.Stmt(&ast.ExprStmt{X: call})
		if name == "isInterrupted" {
			signature.Results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}
			statement = &ast.ReturnStmt{Results: []ast.Expr{call}}
		}
		elts = append(elts, &ast.KeyValueExpr{Key: ast.NewIdent(threadSourceOverrideSlot(method, owner, ctx)), Value: &ast.FuncLit{Type: signature, Body: &ast.BlockStmt{List: []ast.Stmt{statement}}}})
	}
	return []ast.Decl{&ast.FuncDecl{Name: ast.NewIdent(threadSourceBundleMethod), Recv: recv, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("ThreadSourceMethods", ctx)}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CompositeLit{Type: stdjavaQualifiedExpr("ThreadSourceMethods", ctx), Elts: elts}}}}}}}
}
func threadSourceBindStmt(scope *symbol.ClassScope, receiverName string, ctx Ctx) ast.Stmt {
	if !threadSourceDirectRoot(scope, ctx) {
		return nil
	}
	receiver := ast.NewIdent(receiverName)
	protocol := &ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(threadSourceBundleMethod)}, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("ThreadSourceMethods", ctx)}}}}}}}}
	selected := &ast.TypeAssertExpr{X: ast.NewIdent(constructorSelfParam), Type: protocol}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "BindThreadSourceReference", threadSourceNativeViewExpr(scope, receiver, ctx), &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("ObjectInfo")}, methodCall(selected, threadSourceBundleMethod))}
}
func threadSourceSuperConstructor(args []ast.Expr, scope *symbol.ClassScope, receiver string, ctx Ctx) ast.Stmt {
	if !threadSourceDirectRoot(scope, ctx) {
		return nil
	}
	var call ast.Expr
	switch len(args) {
	case 0:
		call = stdjavaCall(ctx, "NewThread", ast.NewIdent("nil"))
	case 1:
		call = stdjavaCall(ctx, "NewThread", args[0])
	case 2:
		call = stdjavaCall(ctx, "NewThreadNamedReference", args...)
	default:
		return nil
	}
	return &ast.AssignStmt{Lhs: []ast.Expr{threadSourceNativeViewExpr(scope, ast.NewIdent(receiver), ctx)}, Tok: token.ASSIGN, Rhs: []ast.Expr{call}}
}
func threadSourceObjectInfoForwarder(scope *symbol.ClassScope, receiver *ast.FieldList, receiverName string, ctx Ctx) ast.Decl {
	if !threadSourceDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.FuncDecl{Name: ast.NewIdent("JavaObjectInfo"), Recv: receiver, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ObjectInfo", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent(receiverName), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}}, &ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent("ObjectInfo")}}}}}}
}

// The synthetic producer supplies its existing runtime ID; this adds only the
// proven native Thread ancestry and views around the same source object.
func threadSourceSyntheticIdentityDecls(name, id string, scope, parent *symbol.ClassScope, interfaces []*symbol.ClassScope, ctx Ctx) []ast.Decl {
	if threadSourceRoot(scope, ctx) == nil {
		return nil
	}
	declarations := syntheticHierarchicalReferenceIdentityDecls(name, id, scope, parent, interfaces, scope.GoTypeParameterNames(), ctx)
	if len(declarations) != 3 {
		return nil
	}
	view := declarations[1].(*ast.FuncDecl)
	receiver := view.Recv.List[0].Names[0].Name
	cases := view.Body.List[0].(*ast.SwitchStmt).Body
	native := threadSourceNativeViewExpr(scope, ast.NewIdent(receiver), ctx)
	for _, constant := range []string{"ThreadTypeID", "RunnableTypeID"} {
		clause := &ast.CaseClause{List: []ast.Expr{stdjavaQualifiedExpr(constant, ctx)}, Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{native}}}}
		cases.List = append(cases.List[:len(cases.List)-1], clause, cases.List[len(cases.List)-1])
	}
	if threadSourceDirectRoot(scope, ctx) {
		registration := declarations[2].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.CallExpr).Fun.(*ast.FuncLit)
		registration.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr).Args[1] = stdjavaQualifiedExpr("ThreadTypeID", ctx)
		declarations = append(declarations, threadSourceObjectInfoForwarder(scope, view.Recv, receiver, ctx))
	}
	return declarations
}
func threadSourceBindSelfStmt(scope *symbol.ClassScope, receiver, self ast.Expr, ctx Ctx) ast.Stmt {
	if !threadSourceDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "BindThreadSourceReference", threadSourceNativeViewExpr(scope, receiver, ctx), &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent("ObjectInfo")}, methodCall(self, threadSourceBundleMethod))}
}
