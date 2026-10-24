package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strconv"
	"strings"
)

var atomicUpdaterFamilies = map[string]string{
	"AtomicIntegerFieldUpdater": "int", "AtomicLongFieldUpdater": "long", "AtomicReferenceFieldUpdater": "java.lang.Object",
}
var atomicUpdaterMethods = map[string]struct {
	name   string
	arity  int
	result string
}{
	"get": {"Get", 1, "value"}, "set": {"Set", 2, "void"}, "lazySet": {"LazySet", 2, "void"},
	"compareAndSet": {"CompareAndSet", 3, "boolean"}, "weakCompareAndSet": {"WeakCompareAndSet", 3, "boolean"},
	"getAndSet": {"GetAndSet", 2, "value"}, "getAndAdd": {"GetAndAdd", 2, "value"}, "addAndGet": {"AddAndGet", 2, "value"},
	"getAndIncrement": {"GetAndIncrement", 1, "value"}, "incrementAndGet": {"IncrementAndGet", 1, "value"},
	"getAndDecrement": {"GetAndDecrement", 1, "value"}, "decrementAndGet": {"DecrementAndGet", 1, "value"},
	"getAndUpdate": {"GetAndUpdate", 2, "value"}, "updateAndGet": {"UpdateAndGet", 2, "value"},
	"getAndAccumulate": {"GetAndAccumulate", 3, "value"}, "accumulateAndGet": {"AccumulateAndGet", 3, "value"},
}

func atomicUpdaterFamily(javaType string, ctx Ctx) string {
	owner, ok := canonicalIntrinsicOwner(javaType, ctx)
	if !ok {
		return ""
	}
	family := strings.TrimPrefix(owner, "java.util.concurrent.atomic.")
	if _, ok := atomicUpdaterFamilies[family]; ok && owner == "java.util.concurrent.atomic."+family {
		return family
	}
	return ""
}
func atomicUpdaterValueType(family string, object *sitter.Node, ctx Ctx, source []byte) string {
	if family != "AtomicReferenceFieldUpdater" {
		return atomicUpdaterFamilies[family]
	}
	typ, _ := inferExprJavaType(object, ctx, source)
	_, args := parseJavaTypeString(typ)
	if len(args) == 2 {
		return reflectionTypeArgumentReadType(args[1])
	}
	return "java.lang.Object"
}
func atomicFieldUpdaterRuntimeTypeExpr(javaType string, args, params []string, ctx Ctx) (ast.Expr, bool) {
	if family := atomicUpdaterFamily(javaType, ctx); family != "" {
		return &ast.StarExpr{X: stdjavaQualifiedExpr(family, ctx)}, true
	}
	if name := atomicUpdaterOperatorName(javaType, ctx); name != "" {
		typ := stdjavaQualifiedExpr(name, ctx)
		if name == "UnaryOperator" || name == "BinaryOperator" {
			if len(args) == 0 {
				args = []string{"Object"}
			}
			if len(args) != 1 {
				return nil, false
			}
			typ = applyTypeArguments(typ, []ast.Expr{javaTypeStringToGoTypeExpr(args[0], params, ctx)})
		}
		return typ, true
	}
	return nil, false
}
func atomicFieldUpdaterExpectedArguments(family, method string, count int, object *sitter.Node, ctx Ctx, source []byte) []string {
	if _, ok := atomicUpdaterFamilies[family]; !ok {
		return nil
	}
	if object == nil {
		if method != "newUpdater" {
			return nil
		}
		if family == "AtomicReferenceFieldUpdater" && count == 3 {
			return []string{"java.lang.Class<?>", "java.lang.Class<?>", "java.lang.String"}
		}
		if family != "AtomicReferenceFieldUpdater" && count == 2 {
			return []string{"java.lang.Class<?>", "java.lang.String"}
		}
		return nil
	}
	signature, ok := atomicUpdaterMethods[method]
	if !ok || count != signature.arity {
		return nil
	}
	value := atomicUpdaterValueType(family, object, ctx, source)
	expected := make([]string, count)
	expected[0] = "java.lang.Object"
	for i := 1; i < count; i++ {
		expected[i] = value
	}
	switch method {
	case "getAndUpdate", "updateAndGet", "getAndAccumulate", "accumulateAndGet":
		operator := "UnaryOperator"
		if count == 3 {
			operator = "BinaryOperator"
		}
		switch family {
		case "AtomicIntegerFieldUpdater":
			operator = "Int" + operator
		case "AtomicLongFieldUpdater":
			operator = "Long" + operator
		default:
			operator += "<" + value + ">"
		}
		expected[count-1] = "java.util.function." + operator
	}
	return expected
}
func atomicUpdaterResult(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	object := invocation.ChildByFieldName("object")
	name := invocation.ChildByFieldName("name")
	if object == nil || name == nil {
		return "", false
	}
	typ, _ := inferExprJavaType(object, ctx, source)
	family := atomicUpdaterFamily(typ, ctx)
	signature, ok := atomicUpdaterMethods[name.Content(source)]
	if family == "" || !ok || invocationArgumentCount(invocation) != signature.arity {
		return "", false
	}
	if signature.result == "value" {
		return atomicUpdaterValueType(family, object, ctx, source), true
	}
	return signature.result, true
}
func atomicUpdaterFactoryResult(family string, invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	count := 2
	if family == "AtomicReferenceFieldUpdater" {
		count = 3
	}
	if invocationArgumentCount(invocation) != count {
		return "", false
	}
	types := []string{}
	for i := 0; i < count-1; i++ {
		typ, ok := inferExprJavaType(invocationArgumentNode(invocation, i), ctx, source)
		if !ok {
			types = append(types, "java.lang.Object")
			continue
		}
		_, args := parseJavaTypeString(typ)
		if len(args) == 1 {
			types = append(types, reflectionTypeArgumentReadType(args[0]))
		} else {
			types = append(types, "java.lang.Object")
		}
	}
	return "java.util.concurrent.atomic." + family + "<" + strings.Join(types, ",") + ">", true
}
func registerAtomicFieldUpdaterIntrinsics() {
	registerAtomicUpdaterOperators()
	for family := range atomicUpdaterFamilies {
		family := family
		registerIntrinsicOwner("java.util.concurrent.atomic."+family, true)
		registerStaticIntrinsic(family, "newUpdater", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			count := 2
			if family == "AtomicReferenceFieldUpdater" {
				count = 3
			}
			if len(args) != count {
				return nil
			}
			return stdjavaCall(ctx, "New"+family+"JavaString", append(args, atomicUpdaterCallerTypeID(ctx))...)
		})
		registerStaticIntrinsicDerivedResultType(family, "newUpdater", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			return atomicUpdaterFactoryResult(family, invocation, ctx, source)
		})
		for method := range atomicUpdaterMethods {
			if family == "AtomicReferenceFieldUpdater" && (strings.Contains(method, "Add") || strings.Contains(method, "Increment") || strings.Contains(method, "Decrement")) {
				continue
			}
			method := method
			registerInstanceNodeIntrinsic(family, method, func(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				return lowerAtomicUpdaterInvocation(family, method, receiver, invocation, ctx, source)
			})
			registerInstanceIntrinsicDerivedResultType(family, method, atomicUpdaterResult)
		}
	}
}

// Receiver and every converted argument are saved before the nullable updater
// is checked. Callback adaptation never invokes or null-checks the callback;
// runtime validates the target receiver before a null callback as the JDK does.
func lowerAtomicUpdaterInvocation(family, method string, receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	signature := atomicUpdaterMethods[method]
	if invocationArgumentCount(invocation) != signature.arity {
		return nil
	}
	object := invocation.ChildByFieldName("object")
	args := parseTypedIntrinsicInvocationArguments(invocation, object, "", method, source, ctx)
	expected := atomicFieldUpdaterExpectedArguments(family, method, len(args), object, ctx, source)
	result, ok := atomicUpdaterResult(invocation, ctx, source)
	if !ok {
		return nil
	}
	var results *ast.FieldList
	if result != "void" {
		results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}}}
	}
	used := affineLoopUsedNames(invocation, source, ctx)
	recv := synchronizedUniqueLocalName("__java2goUpdaterReceiver", used)
	body := []ast.Stmt{intrinsicInvocationTypedLocal(recv, receiver, "java.util.concurrent.atomic."+family, ctx)}
	staged := []ast.Expr{intrinsicExecutionExpr(ctx)}
	for i, arg := range args {
		name := synchronizedUniqueLocalName("__java2goUpdaterArg"+strconv.Itoa(i), used)
		body = append(body, intrinsicInvocationTypedLocal(name, arg, expected[i], ctx))
		staged = append(staged, ast.NewIdent(name))
	}
	body = append(body, &ast.ExprStmt{X: stdjavaCall(ctx, "ReferenceRequireNonNull", ast.NewIdent(recv))})
	if isExternalAtomicUpdaterOperator(expected[len(expected)-1], ctx) {
		last := len(staged) - 1
		operator := atomicUpdaterOperatorName(expected[len(expected)-1], ctx)
		if family == "AtomicReferenceFieldUpdater" {
			value := atomicUpdaterValueType(family, object, ctx, source)
			staged[last] = atomicUpdaterReferenceCallback(staged[last], operator, value, ctx)
			if staged[last] == nil {
				return nil
			}
		} else {
			staged[last] = stdjavaCall(ctx, operator+"CallbackExecution", staged[last])
		}
	}
	call := selectorCall(ast.NewIdent(recv), signature.name+"Execution", staged)
	if family == "AtomicReferenceFieldUpdater" && signature.result == "value" {
		descriptor, known := javaSourceTypeDescriptorExpr(result, ctx)
		if known {
			target := abstractClassToInterface(javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx), result, ctx)
			call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{target}, []ast.Expr{call, descriptor})
		}
	}
	body = append(body, invocationClosureCallStatement(call, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: results}, Body: &ast.BlockStmt{List: body}}}
}

func atomicUpdaterCallerTypeID(ctx Ctx) ast.Expr {
	if ctx.currentClass != nil {
		return javaTypeIDLiteral(sourceClassRuntimeTypeID(ctx.currentClass, ctx), ctx)
	}
	return javaTypeIDLiteral(javaBinaryClassName(ctx), ctx)
}
