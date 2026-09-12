package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// Intrinsic calls cross the same Java conversion boundaries as source methods.
// Keep their parameter types available before parsing arguments: changing only
// a Go lambda's signature afterwards cannot insert boxing into its body.
func intrinsicExpectedArgumentTypes(object *sitter.Node, method string, ctx Ctx, source []byte) []string {
	invocation := object.Parent()
	count := invocationArgumentCount(invocation)
	result := make([]string, count)
	set := func(types ...string) []string {
		copy(result, types)
		return result
	}
	actual := func(index int) string {
		t, _ := inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
		return t
	}
	if class, ok := intrinsicStaticClassName(object, ctx, source); ok {
		if primitive, wrapper := builtinJavaWrapperPrimitive("java.lang."+class, ctx); wrapper {
			switch method {
			case "valueOf":
				if isJavaStringType(actual(0)) || actual(0) == "null" {
					return set("String", "int")
				}
				return set(primitive)
			case "parseByte", "parseShort", "parseInt", "parseLong", "parseFloat", "parseDouble", "parseBoolean":
				return set("String", "int")
			case "compare":
				return set(primitive, primitive)
			case "toString", "hashCode", "isNaN", "isInfinite":
				return set(primitive)
			}
			if class == "Character" {
				if actual(0) == "char" || stripJavaQualifier(actual(0)) == "Character" {
					return set("char")
				}
				return set("int")
			}
		}
		switch class {
		case "Optional", "Collections", "Arrays", "List", "Set", "Stream":
			switch method {
			case "of", "ofNullable", "singletonList", "singleton", "asList":
				element := intrinsicFactoryElementJavaType(invocation, ctx, source)
				for i := range result {
					result[i] = element
				}
				return result
			}
		case "IntStream", "LongStream", "DoubleStream":
			if method == "of" || method == "range" || method == "rangeClosed" || method == "iterate" {
				element := primitiveStreamElementJavaTypes[class]
				for i := range result {
					result[i] = element
				}
				return result
			}
		case "OptionalInt", "OptionalLong", "OptionalDouble":
			if method == "of" {
				return set(primitiveOptionalElementJavaTypes[class])
			}
		case "Math":
			switch method {
			case "sin", "cos", "pow", "sqrt", "floor", "ceil":
				for i := range result {
					result[i] = "double"
				}
			case "abs", "min", "max", "round":
				parameter := intrinsicMathParameterJavaType(invocation, ctx, source)
				for i := range result {
					result[i] = parameter
				}
			}
			return result
		}
		return result
	}
	class, ok := intrinsicReceiverTypeName(object, ctx, source)
	if !ok {
		return result
	}
	if intrinsicFunctionalMethodNames[class] == method {
		javaType, _ := inferExprJavaType(object, ctx, source)
		_, typeArgs := parseJavaTypeString(javaType)
		if sam, bindings, known := builtinFunctionalInterfaceMethod(class, typeArgs); known {
			for index, parameter := range sam.Parameters {
				if index < len(result) {
					result[index] = substituteJavaTypeParams(parameter.OriginalType, bindings)
				}
			}
			return result
		}
	}
	elements := receiverElementJavaTypes(object, ctx, source)
	element := func(index int) string {
		if index < len(elements) {
			return elements[index]
		}
		if p, ok := primitiveOptionalElementJavaTypes[class]; ok {
			return p
		}
		return "Object"
	}
	if _, wrapper := builtinJavaWrapperPrimitive("java.lang."+class, ctx); wrapper {
		switch method {
		case "equals":
			return set("Object")
		case "compareTo":
			return set("java.lang." + class)
		}
	}
	if class == "Comparable" && method == "compareTo" {
		return set(element(0))
	}
	if (class == "Object" || class == "Number") && method == "equals" {
		return set("Object")
	}
	if containsString(listTypeNames, class) {
		switch method {
		case "add":
			if count == 2 {
				return set("int", element(0))
			}
			return set(element(0))
		case "get":
			return set("int")
		case "set":
			return set("int", element(0))
		case "contains", "indexOf", "lastIndexOf":
			return set("Object")
		case "remove":
			if listRemoveUsesIndex(invocation, ctx, source) {
				return set("int")
			}
			return set("Object")
		}
	}
	if containsString(setTypeNames, class) {
		switch method {
		case "add":
			return set(element(0))
		case "contains", "remove":
			return set("Object")
		}
	}
	if containsString(mapTypeNames, class) || class == "ConcurrentHashMap" || class == "ConcurrentMap" {
		switch method {
		case "put", "putIfAbsent":
			return set(element(0), element(1))
		case "get", "containsKey", "containsValue", "remove":
			return set("Object", "Object")
		case "getOrDefault":
			return set("Object", element(1))
		case "replace":
			return set(element(0), element(1), element(1))
		case "compute", "computeIfAbsent", "computeIfPresent":
			return set(element(0))
		case "merge":
			return set(element(0), element(1))
		}
	}
	if class == "Optional" || primitiveOptionalElementJavaTypes[class] != "" {
		if method == "orElse" {
			return set(element(0))
		}
	}
	if class == "Comparator" && method == "compare" {
		return set(element(0), element(0))
	}
	if class == "ExecutorService" {
		if method == "execute" {
			return set("Runnable")
		}
		if method == "submit" {
			resultType, callable := executorSubmitType(invocation, ctx, source)
			if callable {
				return set("Callable<" + resultType + ">")
			}
			return set("Runnable", resultType)
		}
		if method == "awaitTermination" {
			return set("long", "TimeUnit")
		}
	}
	if class == "Thread" && method == "join" {
		return set("long", "int")
	}
	if class == "Future" {
		if method == "get" {
			return set("long", "TimeUnit")
		}
		if method == "cancel" {
			return set("boolean")
		}
	}
	if class == "String" {
		switch method {
		case "charAt", "substring", "subSequence", "repeat":
			return set("int", "int")
		case "equals":
			return set("Object")
		case "concat", "compareTo", "equalsIgnoreCase", "startsWith", "endsWith", "indexOf", "lastIndexOf", "split":
			return set("String", "int")
		}
	}
	if class == "Stream" || primitiveStreamElementJavaTypes[class] != "" {
		switch method {
		case "limit", "skip":
			return set("long")
		case "reduce":
			if count == 3 {
				return set(intrinsicReferenceJavaType(actual(0)))
			}
			if count >= 2 {
				return set(element(0))
			}
		}
	}
	return result
}

func intrinsicMathParameterJavaType(invocation *sitter.Node, ctx Ctx, source []byte) string {
	method := invocation.ChildByFieldName("name").Content(source)
	parameter := "int"
	for index := 0; index < invocationArgumentCount(invocation); index++ {
		t, _ := inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
		if primitive, boxed := builtinJavaWrapperPrimitive(t, ctx); boxed {
			t = primitive
		}
		if promoted, ok := javaNumericPromotionType(parameter, t); ok {
			parameter = promoted
		}
	}
	if method == "round" && parameter != "double" {
		return "float"
	}
	return parameter
}

func listRemoveUsesIndex(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	t, _ := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, source)
	p, primitive := javaPrimitiveType(t)
	return primitive && (p == "byte" || p == "short" || p == "char" || p == "int")
}

func intrinsicFactoryElementJavaType(invocation *sitter.Node, ctx Ctx, source []byte) string {
	_, arguments := parseJavaTypeString(ctx.expectedType)
	if len(arguments) == 1 && expectedTypeTargetsExpression(ctx, invocation) {
		return arguments[0]
	}
	var types []string
	for index := 0; index < invocationArgumentCount(invocation); index++ {
		t, ok := inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
		if !ok || t == "null" {
			continue
		}
		t = intrinsicReferenceJavaType(t)
		types = append(types, t)
	}
	if len(types) == 0 {
		return "Object"
	}
	return javaInferenceLeastUpperBound(types, ctx)
}

func intrinsicLambdaTargetJavaType(types lambdaArgumentTypes) string {
	result := types.resultJavaType
	params := types.paramJavaTypes
	if result == "" || result == "void" {
		switch len(params) {
		case 0:
			return "Runnable"
		case 1:
			return "Consumer<" + params[0] + ">"
		case 2:
			return "BiConsumer<" + strings.Join(params, ",") + ">"
		}
	}
	switch len(params) {
	case 0:
		return "Supplier<" + result + ">"
	case 1:
		return "Function<" + params[0] + "," + result + ">"
	case 2:
		return "BiFunction<" + strings.Join(params, ",") + "," + result + ">"
	}
	return ""
}

func intrinsicLambdaParseContext(ctx Ctx, arg *sitter.Node, types lambdaArgumentTypes) Ctx {
	ctx = ctx.Clone()
	ctx.lambdaParameterJavaTypes = append([]string(nil), types.paramJavaTypes...)
	ctx.lambdaResultJavaType = types.resultJavaType
	if ctx.lambdaResultJavaType == "" {
		ctx.lambdaResultJavaType = "void"
	}
	ctx.expectedType = intrinsicLambdaTargetJavaType(types)
	// Runtime intrinsic consumers receive ordinary closures, not execution-aware
	// java.lang.Runnable adapters.
	if ctx.expectedType == "Runnable" && arg.Type() == "lambda_expression" {
		ctx.expectedType = ""
	}
	ctx.expectedTypeRoot = arg
	return ctx
}

func intrinsicShapeLambdaTypes(arg *sitter.Node, elements []string, kind lambdaResultKind, ctx Ctx, source []byte) lambdaArgumentTypes {
	count := len(lambdaParameterNames(arg.ChildByFieldName("parameters"), source))
	if arg.Type() == "method_reference" {
		count = 1
	}
	params := make([]string, count)
	for i := range params {
		params[i] = elements[0]
	}
	result := elements[0]
	switch kind {
	case lambdaResultVoid:
		result = "void"
	case lambdaResultBool:
		result = "boolean"
	case lambdaResultInt32:
		result = "int"
	case lambdaResultInt64:
		result = "long"
	case lambdaResultFloat64:
		result = "double"
	case lambdaResultAny:
		result = "Object"
	case lambdaResultInferred:
		if inferred, ok := inferLambdaResultJavaType(arg, params, ctx, source); ok {
			result = intrinsicReferenceJavaType(inferred)
			if result == "null" {
				result = "Object"
				if target, arguments := parseJavaTypeString(ctx.expectedType); len(arguments) == 1 && (stripJavaQualifier(target) == "Optional" || stripJavaQualifier(target) == "Stream") {
					result = arguments[0]
				}
			}
		}
	}
	return lambdaArgumentTypes{paramJavaTypes: params, resultJavaType: result}
}

func intrinsicReferenceJavaType(javaType string) string {
	if boxed := ternaryBoxedJavaType(javaType); boxed != "" {
		return "java.lang." + boxed
	}
	return javaType
}

func intrinsicCollectionMethodResultType(invocation *sitter.Node, receiver string, ctx Ctx, source []byte) (string, bool) {
	name := invocation.ChildByFieldName("name").Content(source)
	if receiver == "ExecutorService" && name == "submit" {
		result, _ := executorSubmitType(invocation, ctx, source)
		return "Future<" + result + ">", true
	}
	elements := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
	if intrinsicFunctionalMethodNames[receiver] == name {
		if sam, bindings, known := builtinFunctionalInterfaceMethod(receiver, elements); known {
			return substituteJavaTypeParams(sam.OriginalType, bindings), true
		}
	}
	element := func(index int) string {
		if index < len(elements) {
			return elements[index]
		}
		return "Object"
	}
	if receiver == "Future" && name == "get" {
		return element(0), true
	}
	if receiver == "Callable" && name == "call" {
		return element(0), true
	}
	if receiver == "Optional" || primitiveOptionalElementJavaTypes[receiver] != "" {
		switch name {
		case "get", "orElse", "orElseGet", "orElseThrow", "getAsInt", "getAsLong", "getAsDouble":
			if primitive := primitiveOptionalElementJavaTypes[receiver]; primitive != "" {
				return primitive, true
			}
			return element(0), true
		case "isPresent", "isEmpty":
			return "boolean", true
		case "filter":
			return "Optional<" + element(0) + ">", true
		case "map", "flatMap":
			result, ok := inferLambdaResultJavaType(invocationArgumentNode(invocation, 0), elements, ctx, source)
			if !ok {
				result = element(0)
			}
			if name == "flatMap" {
				return result, ok
			}
			if result == "null" {
				result = "Object"
				if target, arguments := parseJavaTypeString(ctx.expectedType); stripJavaQualifier(target) == "Optional" && len(arguments) == 1 && expectedTypeTargetsExpression(ctx, invocation) {
					result = arguments[0]
				}
			}
			return "Optional<" + intrinsicReferenceJavaType(result) + ">", true
		}
	}
	if containsString(listTypeNames, receiver) {
		switch name {
		case "get", "set":
			return element(0), true
		case "remove":
			if listRemoveUsesIndex(invocation, ctx, source) {
				return element(0), true
			}
			return "boolean", true
		case "size", "indexOf", "lastIndexOf":
			return "int", true
		case "add", "addAll", "contains", "isEmpty":
			return "boolean", true
		}
	}
	if containsString(mapTypeNames, receiver) || receiver == "ConcurrentHashMap" || receiver == "ConcurrentMap" {
		switch name {
		case "get", "getOrDefault", "put", "putIfAbsent", "compute", "computeIfAbsent", "computeIfPresent", "merge":
			return element(1), true
		case "remove", "replace":
			if invocationArgumentCount(invocation) == 1 || name == "replace" && invocationArgumentCount(invocation) == 2 {
				return element(1), true
			}
			return "boolean", true
		case "size":
			return "int", true
		case "containsKey", "containsValue", "isEmpty":
			return "boolean", true
		}
	}
	if containsString(setTypeNames, receiver) {
		switch name {
		case "size":
			return "int", true
		case "add", "contains", "remove", "isEmpty":
			return "boolean", true
		}
	}
	return "", false
}

var intrinsicFunctionalMethodNames = map[string]string{
	"Function": "apply", "BiFunction": "apply", "Supplier": "get",
	"Consumer": "accept", "BiConsumer": "accept", "Predicate": "test", "BiPredicate": "test",
	"UnaryOperator": "apply", "BinaryOperator": "apply",
	"ToIntFunction": "applyAsInt", "ToLongFunction": "applyAsLong", "ToDoubleFunction": "applyAsDouble",
}

func init() {
	for name, method := range intrinsicFunctionalMethodNames {
		registerInstanceIntrinsic(name, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			return &ast.CallExpr{Fun: recv, Args: args}
		})
	}
}

func tryWrapperConstructorIntrinsic(node *sitter.Node, className string, source []byte, ctx Ctx) (ast.Expr, bool) {
	primitive, builtin := builtinJavaWrapperPrimitive(className, ctx)
	if !builtin {
		return nil, false
	}
	name := stripJavaQualifier(className)
	if invocationArgumentCount(node) != 1 {
		return unsupportedIntrinsicValue(node, "java.lang."+name, source, ctx), true
	}
	arg := invocationArgumentNode(node, 0)
	actual, _ := inferExprJavaType(arg, ctx, source)
	expected := primitive
	if isJavaStringType(actual) || actual == "null" {
		expected = "String"
	} else if name == "Float" {
		if actualPrimitive, _ := builtinJavaWrapperPrimitive(actual, ctx); actualPrimitive == "double" || actual == "double" {
			expected = "double"
		}
	}
	if !intrinsicInvocationConversionApplicable(arg, expected, ctx, source) {
		return unsupportedIntrinsicValue(node, "java.lang."+name, source, ctx), true
	}
	argCtx := ctx.Clone()
	argCtx.expectedType = expected
	argCtx.expectedTypeRoot = arg
	value := coerceArgumentToExpectedType(ParseExpr(arg, source, argCtx), arg, expected, ctx, source)
	if expected == "String" {
		parser := map[string]string{"Boolean": "ParseBoolean", "Byte": "ParseByte", "Short": "ParseShort", "Integer": "ParseInt", "Long": "ParseLong", "Float": "ParseFloat", "Double": "ParseDouble"}[name]
		if parser == "" {
			return nil, false
		}
		value = stdjavaCall(ctx, parser, value)
	}
	if name == "Float" && expected == "double" {
		return stdjavaCall(ctx, "NewFloatFromDouble", value), true
	}
	return stdjavaCall(ctx, "New"+name, value), true
}

func wrapperValueOfApplicable(invocation *sitter.Node, wrapper string, ctx Ctx, source []byte) bool {
	expected := intrinsicExpectedArgumentTypes(invocation.ChildByFieldName("object"), "valueOf", ctx, source)
	if len(expected) == 0 || len(expected) > 2 {
		return false
	}
	if len(expected) == 2 && (expected[0] != "String" || (wrapper != "Byte" && wrapper != "Short" && wrapper != "Integer" && wrapper != "Long")) {
		return false
	}
	if wrapper == "Character" && expected[0] == "String" {
		return false
	}
	for index, target := range expected {
		if !intrinsicInvocationConversionApplicable(invocationArgumentNode(invocation, index), target, ctx, source) {
			return false
		}
	}
	return true
}

func intrinsicInvocationConversionApplicable(argument *sitter.Node, expected string, ctx Ctx, source []byte) bool {
	if _, _, applicable := javaInvocationConversionCost(argument, expected, inScopeTypeParameters(ctx), ctx, source); applicable {
		return true
	}
	_, _, applicable := javaLooseInvocationConversionCost(argument, expected, inScopeTypeParameters(ctx), ctx, source)
	return applicable
}

func unsupportedIntrinsicValue(node *sitter.Node, resultType string, source []byte, ctx Ctx) ast.Expr {
	diagnostic := reportUnsupported("intrinsic invocation", node, source, ctx)
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx)}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(diagnostic.String())})}}},
	}}
}

func parseTypedIntrinsicArguments(object *sitter.Node, method string, source []byte, ctx Ctx) []ast.Expr {
	invocation := object.Parent()
	argList := invocation.ChildByFieldName("arguments")
	if argList == nil {
		return nil
	}
	expected := intrinsicExpectedArgumentTypes(object, method, ctx, source)
	perArgument := map[int]lambdaArgumentTypes{}
	if receiver, ok := intrinsicReceiverTypeName(object, ctx, source); ok {
		if typed := lookupLambdaArgumentTypes(receiver, method, invocation, ctx, source); typed != nil {
			perArgument = typed
		} else if kind, found := lookupLambdaShape(receiver, method); found {
			if elements := receiverElementJavaTypes(object, ctx, source); len(elements) == 1 {
				for index, arg := range nodeutil.NamedChildrenOf(argList) {
					if arg.Type() == "lambda_expression" || arg.Type() == "method_reference" {
						perArgument[index] = intrinsicShapeLambdaTypes(arg, elements, kind, ctx, source)
					}
				}
			}
		}
	}
	if class, ok := intrinsicStaticClassName(object, ctx, source); ok {
		if shape, found := staticLambdaShapes[intrinsicKey{class, method}]; found {
			if elements := staticIntrinsicElementJavaTypes(invocation, shape.elementArg, ctx, source); len(elements) == 1 {
				for _, index := range shape.lambdaArgs {
					if arg := invocationArgumentNode(invocation, index); arg != nil && (arg.Type() == "lambda_expression" || arg.Type() == "method_reference") {
						perArgument[index] = intrinsicShapeLambdaTypes(arg, elements, shape.result, ctx, source)
					}
				}
			}
		}
	}
	args := make([]ast.Expr, 0, argList.NamedChildCount())
	for index, arg := range nodeutil.NamedChildrenOf(argList) {
		argCtx := ctx.Clone()
		argCtx.expectedType = ""
		argCtx.expectedTypeRoot = arg
		if index < len(expected) {
			argCtx.expectedType = expected[index]
		}
		if types, ok := perArgument[index]; ok && (arg.Type() == "lambda_expression" || arg.Type() == "method_reference") {
			argCtx = intrinsicLambdaParseContext(argCtx, arg, types)
		}
		parsed := ParseExpr(arg, source, argCtx)
		if index < len(expected) && expected[index] != "" && arg.Type() != "lambda_expression" && arg.Type() != "method_reference" {
			parsed = coerceArgumentToExpectedType(parsed, arg, expected[index], ctx, source)
		}
		if intrinsicLaterArgumentsMayWrite(invocation, index) {
			valueType := ""
			if index < len(expected) {
				valueType = expected[index]
			}
			if valueType == "" {
				valueType, _ = inferExprJavaType(arg, ctx, source)
			}
			parsed = snapshotJavaExpressionValueForType(parsed, valueType, ctx)
		}
		args = append(args, parsed)
	}
	return args
}

func intrinsicLaterArgumentsMayWrite(invocation *sitter.Node, index int) bool {
	for later := index + 1; later < invocationArgumentCount(invocation); later++ {
		if intrinsicArgumentMayWrite(invocationArgumentNode(invocation, later)) {
			return true
		}
	}
	return false
}

func intrinsicArgumentMayWrite(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "assignment_expression", "update_expression", "method_invocation", "object_creation_expression":
		return true
	case "lambda_expression", "method_reference":
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if intrinsicArgumentMayWrite(child) {
			return true
		}
	}
	return false
}
