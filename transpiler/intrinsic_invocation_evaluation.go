package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	sitter "github.com/smacker/go-tree-sitter"
)

// stageStringIntrinsicInvocation gives a nullable String receiver the same
// invocation boundary as a source method: receiver, converted arguments, then
// receiver check and body. Keeping the null check inside the final expression
// would otherwise run it while Go evaluates that expression's first argument.
func stageStringIntrinsicInvocation(object *sitter.Node, method string, receiver ast.Expr, generate intrinsicGenerator, ctx Ctx, source []byte) ast.Expr {
	invocation := object.Parent()
	resultType, known := inferIntrinsicMethodResultType(invocation, ctx, source)
	if !known {
		return nil
	}
	var results *ast.FieldList
	if resultType != "void" {
		results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx)}}}
	}

	used := affineLoopUsedNames(invocation, source, ctx)
	receiverName := synchronizedUniqueLocalName("__java2goInvocationReceiver", used)
	body := []ast.Stmt{stagedInvocationLocal(receiverName, receiver)}
	if ident, nilReceiver := receiver.(*ast.Ident); nilReceiver && ident.Name == "nil" {
		body[0] = intrinsicInvocationTypedLocal(receiverName, receiver, "java.lang.String", ctx)
	}

	arguments := intrinsicArgs(object, method, source, ctx)
	expected := intrinsicInvocationExpectedArgumentTypes(invocation, object, "", method, ctx, source)
	staged := make([]ast.Expr, len(arguments))
	for index, argument := range arguments {
		name := synchronizedUniqueLocalName("__java2goInvocationArg"+strconv.Itoa(index), used)
		node := invocationArgumentNode(invocation, index)
		statement := stagedInvocationLocal(name, argument)
		if invocationArgumentNeedsContextualType(argument, node) {
			javaType := ""
			if index < len(expected) {
				javaType = expected[index]
			}
			if javaType == "" {
				javaType, _ = inferExprJavaType(node, ctx, source)
			}
			if javaType == "" || javaType == ternaryNullJavaType {
				return nil
			}
			statement = intrinsicInvocationTypedLocal(name, argument, javaType, ctx)
		}
		body = append(body, statement)
		staged[index] = ast.NewIdent(name)
	}
	checkedReceiver := stdjavaCall(ctx, "RequireJavaString", ast.NewIdent(receiverName))
	call := generate(checkedReceiver, staged, ctx)
	if call == nil {
		return nil
	}
	body = append(body, invocationClosureCallStatement(call, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: results},
		Body: &ast.BlockStmt{List: body},
	}}
}

func intrinsicInvocationTypedLocal(name string, value ast.Expr, javaType string, ctx Ctx) ast.Stmt {
	return &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent(name)},
			Type:   javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx),
			Values: []ast.Expr{value},
		}},
	}}
}
