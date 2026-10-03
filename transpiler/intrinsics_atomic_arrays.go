package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strconv"
	"strings"
)

var atomicArrayFamilies = map[string]string{"AtomicIntegerArray": "int", "AtomicLongArray": "long"}

func atomicArrayFamily(javaType string, ctx Ctx) string {
	base, args := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(args) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return ""
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return ""
	}
	family := strings.TrimPrefix(owner, "java.util.concurrent.atomic.")
	if _, ok := atomicArrayFamilies[family]; ok && owner == "java.util.concurrent.atomic."+family {
		return family
	}
	return ""
}

func atomicArrayRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	if len(typeArgs) != 0 {
		return nil, false
	}
	for _, parameter := range typeParams {
		if javaType == parameter {
			return nil, false
		}
	}
	if family := atomicArrayFamily(javaType, ctx); family != "" {
		return &ast.StarExpr{X: stdjavaQualifiedExpr(family, ctx)}, true
	}
	return nil, false
}

type atomicArrayMethod struct {
	goName, result string
	parameters     []string
}

func atomicArraySignature(family, method string) (atomicArrayMethod, bool) {
	value, known := atomicArrayFamilies[family]
	if !known {
		return atomicArrayMethod{}, false
	}
	switch method {
	case "length":
		return atomicArrayMethod{"Length", "int", nil}, true
	case "get":
		return atomicArrayMethod{"Get", value, []string{"int"}}, true
	case "set":
		return atomicArrayMethod{"Set", "void", []string{"int", value}}, true
	case "compareAndSet":
		return atomicArrayMethod{"CompareAndSet", "boolean", []string{"int", value, value}}, true
	case "getAndSet":
		return atomicArrayMethod{"GetAndSet", value, []string{"int", value}}, true
	case "getAndAdd":
		return atomicArrayMethod{"GetAndAdd", value, []string{"int", value}}, true
	case "addAndGet":
		return atomicArrayMethod{"AddAndGet", value, []string{"int", value}}, true
	}
	return atomicArrayMethod{}, false
}

func atomicArrayExpectedArguments(javaType, method string, count int, ctx Ctx) []string {
	family := atomicArrayFamily(javaType, ctx)
	signature, ok := atomicArraySignature(family, method)
	if !ok || len(signature.parameters) != count {
		return nil
	}
	return signature.parameters
}

func init() {
	for family, value := range atomicArrayFamilies {
		registerIntrinsicOwner("java.util.concurrent.atomic."+family, true)
		registerConstructorNodeIntrinsic(family, func(typeArgs, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if len(typeArgs) != 0 || len(args) != 1 {
				return unsupportedIntrinsicValue(node, "java.util.concurrent.atomic."+family, source, ctx)
			}
			parameter, helper := "int", "New"+family
			argument := invocationArgumentNode(node, 0)
			if !intrinsicInvocationConversionApplicable(argument, parameter, ctx, source) {
				parameter, helper = value+"[]", helper+"FromArray"
				if !intrinsicInvocationConversionApplicable(argument, parameter, ctx, source) {
					return unsupportedIntrinsicValue(node, "java.util.concurrent.atomic."+family, source, ctx)
				}
			}
			return stdjavaCall(ctx, helper, coerceArgumentToExpectedType(args[0], argument, parameter, ctx, source))
		})
		for _, method := range []string{"length", "get", "set", "compareAndSet", "getAndSet", "getAndAdd", "addAndGet"} {
			signature, _ := atomicArraySignature(family, method)
			registerInstanceIntrinsicResultType(family, method, signature.result)
			registerInstanceNodeIntrinsic(family, method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if invocationArgumentCount(node) != len(signature.parameters) {
					return unsupportedIntrinsicValue(node, signature.result, source, ctx)
				}
				for index, parameter := range signature.parameters {
					if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, index), parameter, ctx, source) {
						return unsupportedIntrinsicValue(node, signature.result, source, ctx)
					}
				}
				args := parseTypedIntrinsicInvocationArguments(node, node.ChildByFieldName("object"), "", method, source, ctx)
				var results *ast.FieldList
				if signature.result != "void" {
					results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(signature.result, inScopeTypeParameters(ctx), ctx)}}}
				}
				used := affineLoopUsedNames(node, source, ctx)
				recvName := synchronizedUniqueLocalName("__java2goAtomicArrayReceiver", used)
				body := []ast.Stmt{intrinsicInvocationTypedLocal(recvName, receiver, "java.util.concurrent.atomic."+family, ctx)}
				staged := make([]ast.Expr, len(args))
				for index, arg := range args {
					name := synchronizedUniqueLocalName("__java2goAtomicArrayArg"+strconv.Itoa(index), used)
					body = append(body, intrinsicInvocationTypedLocal(name, arg, signature.parameters[index], ctx))
					staged[index] = ast.NewIdent(name)
				}
				// All arguments finish evaluation before the runtime performs receiver and
				// bounds checks, including calls on a null receiver.
				call := selectorCall(ast.NewIdent(recvName), signature.goName, staged)
				body = append(body, invocationClosureCallStatement(call, results))
				return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: results}, Body: &ast.BlockStmt{List: body}}}
			})
		}
	}
}
