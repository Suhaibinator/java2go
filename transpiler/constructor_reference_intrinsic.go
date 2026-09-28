package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// A nullary library constructor can implement Supplier<R>. Its generic type
// arguments come from the written constructor type or the target result R.
func nullaryBuiltinConstructorReference(node *sitter.Node, class, target, result string, parameters []string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if len(parameters) != 0 || result == "void" {
		return nil, false
	}
	_, arguments := parseJavaTypeString(target)
	if len(arguments) == 0 {
		_, arguments = parseJavaTypeString(result)
	}
	types := make([]ast.Expr, len(arguments))
	for index, arg := range arguments {
		types[index] = javaTypeStringToGoTypeExpr(arg, inScopeTypeParameters(ctx), ctx)
	}
	call, ok := tryConstructorIntrinsic(class, types, nil, node, ctx, source)
	if !ok {
		return nil, false
	}
	signature := &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}}}}
	if methodReferenceUsesExecutionSAM(ctx) {
		signature.Params.List = []*ast.Field{executionParameterField(executionParameterName(node, source, ctx), ctx)}
	}
	var closure ast.Expr = &ast.FuncLit{Type: signature, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}}
	if adapted := wrapLambdaWithFunctionalInterfaceAdapter(closure, ctx.expectedType, methodReferenceUsesExecutionSAM(ctx), ctx); adapted != nil {
		closure = adapted
	}
	return closure, true
}
