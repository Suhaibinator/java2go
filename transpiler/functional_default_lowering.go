package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Default-method type inference keeps Java reference boxing at the SAM boundary.
func functionalDefaultTypes(invocation *sitter.Node, ctx Ctx, source []byte) (string, []string, string, bool) {
	object := invocation.ChildByFieldName("object")
	name := invocation.ChildByFieldName("name")
	if object == nil || name == nil || invocationArgumentCount(invocation) != 1 {
		return "", nil, "", false
	}
	typ, ok := inferExprJavaType(object, ctx, source)
	if !ok {
		return "", nil, "", false
	}
	family := nativeFunctionalFamily(typ, ctx)
	_, arguments := parseJavaTypeString(typ)
	c, ok := nativeFunctionalContract(family, arguments)
	if !ok {
		return "", nil, "", false
	}
	method := name.Content(source)
	switch family {
	case "IntUnaryOperator", "LongUnaryOperator", "Consumer":
		if method == "andThen" || (family != "Consumer" && method == "compose") {
			return family, c.arguments, "java.util.function." + family + functionalTypeSuffix(c.arguments), true
		}
	case "Function", "UnaryOperator", "BiFunction", "BinaryOperator":
		if method != "andThen" && method != "compose" {
			return "", nil, "", false
		}
		if method == "compose" && (family == "BiFunction" || family == "BinaryOperator") {
			return "", nil, "", false
		}
		args := append([]string(nil), c.arguments...)
		if family == "UnaryOperator" {
			args = []string{args[0], args[0]}
		}
		if family == "BinaryOperator" {
			args = []string{args[0], args[0], args[0]}
		}
		arg := invocationArgumentNode(invocation, 0)
		variable := "java.lang.Object"
		if method == "andThen" {
			if actual, known := inferExprJavaType(arg, ctx, source); known {
				a, _ := resolvedNativeFunctionalArguments(actual, "Function", ctx)
				if len(a) == 2 {
					variable = reflectionTypeArgumentReadType(a[1])
				}
			}
			if arg.Type() == "lambda_expression" || arg.Type() == "method_reference" {
				if inferred, known := inferLambdaResultJavaType(arg, []string{c.result}, ctx, source); known {
					variable = intrinsicReferenceJavaType(inferred)
				}
			}
			resultArgs := append(append([]string(nil), args[:len(args)-1]...), variable)
			resultFamily := "Function"
			if len(args) == 3 {
				resultFamily = "BiFunction"
			}
			return family, append(args, variable), "java.util.function." + resultFamily + functionalTypeSuffix(resultArgs), true
		}
		if actual, known := inferExprJavaType(arg, ctx, source); known {
			a, _ := resolvedNativeFunctionalArguments(actual, "Function", ctx)
			if len(a) == 2 {
				variable = reflectionTypeArgumentReadType(a[0])
			}
		}
		if expectedBase, a := parseJavaTypeString(ctx.expectedType); nativeFunctionalFamily(expectedBase, ctx) == "Function" && len(a) == 2 {
			variable = reflectionTypeArgumentReadType(a[0])
		}
		return family, append(args, variable), "java.util.function.Function" + functionalTypeSuffix([]string{variable, c.result}), true
	}
	return "", nil, "", false
}
func functionalTypeSuffix(args []string) string {
	if len(args) == 0 {
		return ""
	}
	s := "<"
	for i, a := range args {
		if i > 0 {
			s += ","
		}
		s += a
	}
	return s + ">"
}
func functionalDefaultExpected(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	family, args, _, ok := functionalDefaultTypes(invocation, ctx, source)
	if !ok {
		return nil
	}
	method := invocation.ChildByFieldName("name").Content(source)
	if family == "IntUnaryOperator" || family == "LongUnaryOperator" || family == "Consumer" {
		return []string{"java.util.function." + family + functionalTypeSuffix(args)}
	}
	if method == "compose" {
		return []string{"java.util.function.Function" + functionalTypeSuffix([]string{args[len(args)-1], args[0]})}
	}
	return []string{"java.util.function.Function" + functionalTypeSuffix(args[len(args)-2:])}
}
func registerNativeFunctionalDefaults() {
	for _, family := range []string{"Function", "UnaryOperator", "BiFunction", "BinaryOperator", "Consumer", "IntUnaryOperator", "LongUnaryOperator"} {
		for _, method := range []string{"andThen", "compose"} {
			family, method := family, method
			if method == "compose" && (family == "Consumer" || family == "BiFunction" || family == "BinaryOperator") {
				continue
			}
			registerInstanceIntrinsicDerivedResultType(family, method, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
				_, _, result, ok := functionalDefaultTypes(node, ctx, source)
				return result, ok
			})
			registerInstanceNodeIntrinsic(family, method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				resolved, types, result, ok := functionalDefaultTypes(node, ctx, source)
				if !ok || resolved != family {
					return nil
				}
				object := node.ChildByFieldName("object")
				args := parseTypedIntrinsicInvocationArguments(node, object, "", method, source, ctx)
				if len(args) != 1 {
					return nil
				}
				receiverType, _ := inferExprJavaType(object, ctx, source)
				expected := functionalDefaultExpected(node, ctx, source)
				used := affineLoopUsedNames(node, source, ctx)
				recv := synchronizedUniqueLocalName("__java2goFunctionReceiver", used)
				after := synchronizedUniqueLocalName("__java2goFunctionNext", used)
				body := []ast.Stmt{intrinsicInvocationTypedLocal(recv, receiver, receiverType, ctx), intrinsicInvocationTypedLocal(after, args[0], expected[0], ctx), &ast.ExprStmt{X: stdjavaCall(ctx, "ReferenceRequireNonNull", ast.NewIdent(recv))}}
				runtimeFamily := family
				if family == "UnaryOperator" {
					runtimeFamily = "Function"
				}
				if family == "BinaryOperator" {
					runtimeFamily = "BiFunction"
				}
				runtimeMethod := "AndThen"
				if method == "compose" {
					runtimeMethod = "Compose"
				}
				parameters := []ast.Expr{intrinsicExecutionExpr(ctx), ast.NewIdent(recv), ast.NewIdent(after)}
				call := ast.Expr(stdjavaCall(ctx, runtimeFamily+runtimeMethod+"Execution", parameters...))
				if len(types) > 0 {
					goTypes := []ast.Expr{}
					for _, typ := range types {
						goTypes = append(goTypes, javaTypeStringToGoTypeExpr(typ, inScopeTypeParameters(ctx), ctx))
					}
					call = stdjavaGenericCall(ctx, runtimeFamily+runtimeMethod+"Execution", goTypes, parameters)
				}
				results := &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}}}
				body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
				return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: results}, Body: &ast.BlockStmt{List: body}}}
			})
		}
	}
}
