package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func sourceArrayListStorageName(scope *symbol.ClassScope, ctx Ctx) string {
	base := "Java2goArrayListStorageFor" + fmt.Sprintf("%X", []byte(javaClassBinaryName(scope)))
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix != 0 {
			candidate += "_" + strconv.Itoa(suffix)
		}
		if !generatedIdentifierExists(candidate, scope) && !superclassEmbeddingMemberExists(candidate, scope, ctx) && !superclassEmbeddingSourceIdentifierExists(candidate, ctx) {
			return candidate
		}
	}
}

func sourceArrayListStorageField(scope *symbol.ClassScope, ctx Ctx) *ast.Field {
	if !sourceArrayListDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(sourceArrayListStorageName(scope, ctx))}, Type: &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("List", ctx), []ast.Expr{sourceArrayListStorageTypeArgument(scope, ctx)})}}
}

func sourceArrayListNativeViewExpr(scope *symbol.ClassScope, receiver ast.Expr, ctx Ctx) ast.Expr {
	root := sourceArrayListRoot(scope, ctx)
	if root == nil {
		return nil
	}
	view := sourceClassViewExpr(scope, root, receiver, ctx)
	if view == nil {
		return nil
	}
	return &ast.SelectorExpr{X: view, Sel: ast.NewIdent(sourceArrayListStorageName(root, ctx))}
}

func sourceArrayListSuperConstructor(args []ast.Expr, scope *symbol.ClassScope, receiver string, ctx Ctx) ast.Stmt {
	if !sourceArrayListDirectRoot(scope, ctx) || len(args) != 0 {
		return nil
	}
	return &ast.AssignStmt{Lhs: []ast.Expr{sourceArrayListNativeViewExpr(scope, ast.NewIdent(receiver), ctx)}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaGenericCall(ctx, "NewArrayList", []ast.Expr{sourceArrayListStorageTypeArgument(scope, ctx)}, nil)}}
}

func sourceArrayListAddProtocol(ctx Ctx) *ast.InterfaceType {
	return &ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("CollectionAddJava2goExecution")}, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("", ctx), {Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}}}}}
}

func sourceArrayListCarrierInitStmt(scope *symbol.ClassScope, receiver string, self ast.Expr, ctx Ctx) ast.Stmt {
	if !sourceArrayListDirectRoot(scope, ctx) || self == nil {
		return nil
	}
	e, v := ast.NewIdent("__java2goArrayListOperation"), ast.NewIdent("__java2goArrayListValue")
	selected := &ast.TypeAssertExpr{X: self, Type: sourceArrayListAddProtocol(ctx)}
	add := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(e.Name, ctx), {Names: []*ast.Ident{v}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{methodCall(selected, "CollectionAddJava2goExecution", e, v)}}}}}
	methods := &ast.CompositeLit{Type: stdjavaQualifiedExpr("ArrayListSourceMethods", ctx), Elts: []ast.Expr{&ast.KeyValueExpr{Key: ast.NewIdent("Add"), Value: add}}}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "BindArrayListSourceReference", sourceArrayListNativeViewExpr(scope, ast.NewIdent(receiver), ctx), &ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent("ObjectInfo")}, methods)}
}

func sourceArrayListBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if sourceArrayListRoot(scope, ctx) == nil {
		return nil
	}
	method, owner := sourceArrayListSelectedAdd(scope, ctx)
	if method == nil {
		return nil
	}
	name := abstractCollectionReceiverName(ctx, nil)
	execution := executionNameForParams(nil, append(scope.GoTypeParameterNames(), name)...)
	value := collisionSafeExecutionIdentifier("__java2goArrayListValue", scope)
	var selected ast.Expr = ast.NewIdent(name)
	if classNeedsVirtualDispatch(owner, ctx) {
		selected = &ast.SelectorExpr{X: selected, Sel: ast.NewIdent(classDispatchFieldName(owner))}
	}
	call := methodCall(selected, executionImplementationName(method, owner, ctx), ast.NewIdent(execution), sourceArrayListScalarAddArgument(scope, ast.NewIdent(value), ctx))
	bridge := &ast.FuncDecl{Name: ast.NewIdent("CollectionAddJava2goExecution"), Recv: abstractCollectionReceiver(ctx, name), Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(execution, ctx), {Names: []*ast.Ident{ast.NewIdent(value)}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(name), &ast.ReturnStmt{Results: []ast.Expr{call}}}}}
	return []ast.Decl{bridge}
}

func sourceArrayListObjectInfoDecl(scope *symbol.ClassScope, receiver *ast.FieldList, name string, ctx Ctx) ast.Decl {
	if !sourceArrayListDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.FuncDecl{Name: ast.NewIdent("JavaObjectInfo"), Recv: receiver, Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ObjectInfo", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent(name), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}}, &ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(name), Sel: ast.NewIdent("ObjectInfo")}}}}}}
}

func sourceArrayListDefaultDecls(ctx Ctx) []ast.Decl {
	var declarations []ast.Decl
	for _, method := range ctx.currentClass.Methods {
		if !method.RuntimeDefault {
			continue
		}
		if scalar := sourceArrayListScalarGetDecls(method, ctx); scalar != nil {
			declarations = append(declarations, scalar...)
			continue
		}
		for _, spec := range sourceArrayListDefaults(ctx.currentClass, ctx) {
			if spec.java != method.OriginalName {
				continue
			}
			params := &ast.FieldList{}
			var args []ast.Expr
			for _, parameter := range method.Parameters {
				params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: javaTypeStringToGoTypeExpr(parameter.OriginalType, inScopeTypeParameters(ctx), ctx)})
				args = append(args, ast.NewIdent(parameter.Name))
			}
			receiver := abstractCollectionReceiverName(ctx, params)
			execution := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
			args = append([]ast.Expr{ast.NewIdent(execution), sourceArrayListNativeViewExpr(ctx.currentClass, ast.NewIdent(receiver), ctx)}, args...)
			results := &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(ctx), ctx)}}}
			declaration := &ast.FuncDecl{Name: ast.NewIdent(method.Name), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), &ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, spec.runtime, args...)}}}}}
			declarations = append(declarations, buildExecutionAwareFuncDecls(declaration, executionImplementationName(method, ctx.currentClass, ctx), execution, ctx)...)
		}
	}
	return declarations
}

// This terminal handles only an actual direct native-parent selection. Source
// super calls in indirect descendants continue through the normal resolver.
func sourceArrayListSuperInvocation(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	object, name := node.ChildByFieldName("object"), node.ChildByFieldName("name")
	if object == nil || object.Type() != "super" || name == nil || !sourceArrayListDirectRoot(ctx.currentClass, ctx) {
		return nil
	}
	for _, spec := range sourceArrayListDefaults(ctx.currentClass, ctx) {
		if name.Content(source) != spec.java || invocationArgumentCount(node) != len(spec.parameters) {
			continue
		}
		definition := &symbol.Definition{OriginalName: spec.java, OriginalType: spec.result, HasBody: true, RuntimeDefault: true}
		for _, formal := range spec.parameters {
			definition.Parameters = append(definition.Parameters, &symbol.Definition{OriginalType: formal})
		}
		args, _ := parseResolvedInvocationArguments(&methodResolution{def: definition, owner: ctx.currentClass}, node.ChildByFieldName("arguments"), source, ctx, spec.parameters, nil, node)
		args = append([]ast.Expr{intrinsicExecutionExpr(ctx), sourceArrayListNativeViewExpr(ctx.currentClass, ast.NewIdent(sourceInstanceReceiverName(ctx)), ctx)}, args...)
		return stdjavaCall(ctx, spec.runtime, args...)
	}
	return nil
}
