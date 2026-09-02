package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// This file implements a data-driven table that maps Java standard-library
// method calls onto Go expressions. It replaces the scattered one-off special
// cases that previously lived inline in ParseExpr's method_invocation handling
// (System.out.println, String.length, ...).
//
// The table is split into two halves:
//
//   - instanceIntrinsics maps an (unqualified Java receiver type, method name)
//     to a generator. These fire when the receiver expression has a known Java
//     type, e.g. a `String` local calling `.substring(...)`.
//
//   - staticIntrinsics maps a (class name, method name) to a generator. These
//     fire when the receiver is a bare identifier naming a well-known class that
//     is not a user-defined type, e.g. `Math.max(...)` or `Integer.parseInt(...)`.
//
// A generator receives the already-parsed receiver expression (nil for statics),
// the already-parsed argument expressions, and the surrounding context, and
// returns the replacement Go expression. Returning nil means "not handled, fall
// through to the normal method-call path" — this lets a generator bail out when
// an overload it does not support is encountered (e.g. an unexpected arg count).
//
// This design is intentionally extensible: collection intrinsics (List, Map,
// Set, Optional, iterators) register into the same tables in a follow-up, and
// boxed-type constant access (Integer.MAX_VALUE) hooks in through
// staticFieldIntrinsics.

// intrinsicGenerator builds the Go replacement expression for a recognized Java
// call. recv is the parsed receiver (nil for static calls); args are the parsed
// arguments. Returning nil signals that this particular call is not handled.
type intrinsicGenerator func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr

// intrinsicKey identifies an entry by Java type (or class) name and method name.
type intrinsicKey struct {
	typeName   string
	methodName string
}

// constructorGenerator builds the Go replacement for a `new Type<TypeArgs>(args)`
// expression. typeArgs are the resolved Go type-argument expressions (e.g. for
// `new ArrayList<String>()`, typeArgs is [string]). Returning nil falls through
// to the normal constructor path.
type constructorGenerator func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr

var (
	instanceIntrinsics              = map[intrinsicKey]intrinsicGenerator{}
	staticIntrinsics                = map[intrinsicKey]intrinsicGenerator{}
	staticFieldIntrinsics           = map[intrinsicKey]func(ctx Ctx) ast.Expr{}
	instanceIntrinsicResultTypes    = map[intrinsicKey]string{}
	staticIntrinsicResultTypes      = map[intrinsicKey]string{}
	staticFieldIntrinsicResultTypes = map[intrinsicKey]string{}
	// staticIntrinsicDerivedResultTypes holds static intrinsics whose Java result
	// type depends on the call's arguments (Stream.of yields Stream<T> for the T
	// of its first argument), which a fixed result-type string cannot express.
	staticIntrinsicDerivedResultTypes = map[intrinsicKey]derivedResultType{}
	// staticIntrinsicTypeArgs holds static intrinsics that need explicit Go type
	// arguments computed from the Java call site.
	staticIntrinsicTypeArgs = map[intrinsicKey]typeArgDeriver{}
	// constructorIntrinsics is keyed by Java class name; generators select the
	// right overload by arg count.
	constructorIntrinsics = map[string]constructorGenerator{}
)

// registerConstructorIntrinsic adds a `new Type(...)` intrinsic.
func registerConstructorIntrinsic(className string, gen constructorGenerator) {
	constructorIntrinsics[className] = gen
}

// tryConstructorIntrinsic attempts to rewrite a `new className<typeArgs>(args)`
// expression via the constructor intrinsics table. It only fires when className
// is not a user-defined class.
func tryConstructorIntrinsic(className string, typeArgs, args []ast.Expr, ctx Ctx) (ast.Expr, bool) {
	name := stripJavaQualifier(className)
	if name == "" {
		return nil, false
	}
	if resolveClassScopeByQualifiedName(ctx, className) != nil {
		return nil, false
	}
	gen, ok := constructorIntrinsics[name]
	if !ok {
		return nil, false
	}
	if result := gen(typeArgs, args, ctx); result != nil {
		return result, true
	}
	return nil, false
}

// registerInstanceIntrinsic adds an instance-method intrinsic. Follow-up tasks
// (collections) call this from their own init functions.
func registerInstanceIntrinsic(typeName, methodName string, gen intrinsicGenerator) {
	instanceIntrinsics[intrinsicKey{typeName, methodName}] = gen
}

func registerInstanceIntrinsicResultType(typeName, methodName, resultType string) {
	instanceIntrinsicResultTypes[intrinsicKey{typeName, methodName}] = resultType
}

func registerStaticIntrinsicResultType(typeName, methodName, resultType string) {
	staticIntrinsicResultTypes[intrinsicKey{typeName, methodName}] = resultType
}

// typeArgDeriver computes the explicit Go type arguments a static intrinsic
// needs, from its call node. Returning nil leaves the call without them.
type typeArgDeriver func(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr

// registerStaticIntrinsicTypeArgs records a type-argument deriver for a static
// intrinsic. The result is available to the generator as ctx.intrinsicTypeArgs.
func registerStaticIntrinsicTypeArgs(className, methodName string, derive typeArgDeriver) {
	staticIntrinsicTypeArgs[intrinsicKey{className, methodName}] = derive
}

// derivedResultType computes a static intrinsic's Java result type from its call
// node. It returns false when the type cannot be determined.
type derivedResultType func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool)

// registerStaticIntrinsicDerivedResultType records an argument-derived result
// type for a static intrinsic. It takes precedence over any fixed result type
// registered for the same call.
func registerStaticIntrinsicDerivedResultType(className, methodName string, derive derivedResultType) {
	staticIntrinsicDerivedResultTypes[intrinsicKey{className, methodName}] = derive
}

// derivedResultTypeFromArgument builds a derivedResultType that wraps the
// inferred Java type of one argument in a container, e.g. Stream.of(x) yields
// "Stream<" + typeof(x) + ">".
func derivedResultTypeFromArgument(container string, argIndex int) derivedResultType {
	return func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		argNode := invocationArgumentNode(invocation, argIndex)
		if argNode == nil {
			return "", false
		}
		javaType, ok := inferExprJavaType(argNode, ctx, source)
		if !ok || strings.TrimSpace(javaType) == "" {
			return "", false
		}
		return container + "<" + javaType + ">", true
	}
}

func registerStaticFieldIntrinsicResultType(typeName, fieldName, resultType string) {
	staticFieldIntrinsicResultTypes[intrinsicKey{typeName, fieldName}] = resultType
}

// registerStaticIntrinsic adds a static-method intrinsic.
func registerStaticIntrinsic(className, methodName string, gen intrinsicGenerator) {
	staticIntrinsics[intrinsicKey{className, methodName}] = gen
}

// registerStaticFieldIntrinsic adds a static-field (constant) intrinsic such as
// Integer.MAX_VALUE.
func registerStaticFieldIntrinsic(className, fieldName string, gen func(ctx Ctx) ast.Expr) {
	staticFieldIntrinsics[intrinsicKey{className, fieldName}] = gen
}

// tryInstanceIntrinsic attempts to rewrite a `receiver.method(args)` invocation
// using the intrinsics table. It resolves the receiver's Java type, looks up the
// table, and returns the generated expression (or nil if nothing matched).
func tryInstanceIntrinsic(objectNode *sitter.Node, methodName string, source []byte, ctx Ctx) (ast.Expr, bool) {
	receiverType, ok := intrinsicReceiverTypeName(objectNode, ctx, source)
	if !ok {
		return nil, false
	}

	_, hasNodeIntrinsic := instanceNodeIntrinsics[intrinsicKey{receiverType, methodName}]
	gen, ok := instanceIntrinsics[intrinsicKey{receiverType, methodName}]
	if !ok && !hasNodeIntrinsic {
		return nil, false
	}

	recv := ParseExpr(objectNode, source, ctx)
	// Fields, parameters, and method results can carry the concrete null String
	// sentinel, while explicitly nullable locals can carry interface nil.
	// Normalize every String receiver through the runtime bridge so dereferencing
	// either representation throws NullPointerException.
	if receiverType == "String" && !isDefinitelyNonNullStringExpression(objectNode, ctx, source) {
		recv = stdjavaCall(ctx, "StringRequireNonNull", recv)
	}
	// A node-aware intrinsic parses its own arguments, since it needs their
	// syntax rather than their parsed values.
	if nodeGen, ok := instanceNodeIntrinsics[intrinsicKey{receiverType, methodName}]; ok {
		if result := nodeGen(recv, objectNode.Parent(), ctx, source); result != nil {
			return result, true
		}
		if gen == nil {
			return nil, false
		}
	}

	args := intrinsicArgs(objectNode, methodName, source, ctx)

	// An explicit per-argument typer wins over the element-typed shape, which
	// cannot describe lambdas whose signatures differ from one another.
	if perArgument := lookupLambdaArgumentTypes(receiverType, methodName, objectNode.Parent(), ctx, source); perArgument != nil {
		typeParams := inScopeTypeParameters(ctx)
		for index, types := range perArgument {
			if index >= len(args) {
				continue
			}
			paramTypes := make([]ast.Expr, 0, len(types.paramJavaTypes))
			for _, javaType := range types.paramJavaTypes {
				paramTypes = append(paramTypes, javaTypeStringToGoTypeExpr(javaType, typeParams, ctx))
			}
			var resultType ast.Expr
			if types.resultJavaType != "" {
				resultType = javaTypeStringToGoTypeExpr(types.resultJavaType, typeParams, ctx)
			}
			args[index] = retypeLambdaWithTypes(args[index], paramTypes, resultType)
		}
		if result := gen(recv, args, ctx); result != nil {
			return result, true
		}
		return nil, false
	}

	// A lambda argument to one of these stdlib methods is parsed before its
	// parameter type is known, so it comes back as `func(x any)`. Re-type each
	// lambda parameter from the receiver's element type and give the closure the
	// result type the functional interface requires (predicate -> bool, consumer
	// -> void, mapper -> the value's type), so the generated Go is well-typed.
	if kind, ok := lookupLambdaShape(receiverType, methodName); ok {
		if elem := receiverElementTypeExpr(objectNode, ctx, source); elem != nil {
			elementJavaTypes := receiverElementJavaTypes(objectNode, ctx, source)
			for i, a := range args {
				// A mapper's result type comes from the lambda body, which has to be
				// inferred per argument rather than once for the whole call.
				mappedResult := intrinsicLambdaResultTypeExpr(objectNode, i, elementJavaTypes, kind, ctx, source)
				args[i] = retypeElementLambda(a, elem, mappedResult, kind)
			}
		}
	}

	if result := gen(recv, args, ctx); result != nil {
		return result, true
	}
	return nil, false
}

func isNullableValueBackedLocal(node *sitter.Node, ctx Ctx, source []byte) bool {
	for node != nil && node.Type() == "parenthesized_expression" && node.NamedChildCount() > 0 {
		node = node.NamedChild(0)
	}
	if node == nil || node.Type() != "identifier" || ctx.localScope == nil {
		return false
	}
	local := ctx.localScope.FindVariable(node.Content(source))
	return local != nil && local.Nullable
}

// isNullableStringStorageExpression identifies String reads whose generated
// storage can directly contain Java null. Fields receive the concrete sentinel
// at allocation time; explicitly nullable locals use an interface slot. Plain
// parameters and non-null expressions retain the existing direct-string fast
// path, avoiding unnecessary runtime calls and imports.
func isNullableStringStorageExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	for node != nil && node.Type() == "parenthesized_expression" && node.NamedChildCount() > 0 {
		node = node.NamedChild(0)
	}
	if node == nil {
		return false
	}
	if isNullableValueBackedLocal(node, ctx, source) {
		return true
	}
	if node.Type() == "field_access" {
		javaType, ok := inferExprJavaType(node, ctx, source)
		return ok && isJavaStringType(javaType)
	}
	if node.Type() != "identifier" || ctx.currentClass == nil {
		return false
	}
	name := node.Content(source)
	if ctx.localScope != nil {
		if ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil {
			return false
		}
	}
	field := findFieldInHierarchy(ctx.currentClass, name, ctx)
	return field != nil && isJavaStringType(field.OriginalType)
}

func isDefinitelyNonNullStringExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	for node != nil && node.Type() == "parenthesized_expression" && node.NamedChildCount() > 0 {
		node = node.NamedChild(0)
	}
	if node == nil {
		return false
	}
	switch node.Type() {
	case "string_literal":
		return true
	case "binary_expression":
		return node.Child(1) != nil && node.Child(1).Content(source) == "+" && isStringLikeExprNode(node, ctx, source)
	case "method_invocation":
		nameNode := node.ChildByFieldName("name")
		if nameNode == nil {
			return false
		}
		methodName := nameNode.Content(source)
		objectNode := node.ChildByFieldName("object")
		if objectNode == nil {
			return false
		}
		if receiverType, ok := inferExprJavaType(objectNode, ctx, source); ok {
			base, _ := parseJavaTypeString(receiverType)
			if stripJavaQualifier(base) == "String" && stringReturningStringMethods[methodName] {
				return true
			}
		}
		if objectNode.Type() == "identifier" && objectNode.Content(source) == "String" {
			switch methodName {
			case "valueOf", "format", "join":
				return true
			}
		}
	}
	return false
}

// lambdaResultKind describes the result type a re-typed element lambda must have.
type lambdaResultKind int

const (
	// lambdaResultElement: the result type is the element type (mapper/operator,
	// e.g. Function<T,R> approximated as T->T, BinaryOperator<T>).
	lambdaResultElement lambdaResultKind = iota
	// lambdaResultBool: the result is bool (Predicate<T>).
	lambdaResultBool
	// lambdaResultVoid: the closure returns nothing (Consumer<T>).
	lambdaResultVoid
	// lambdaResultInferred: the result type is whatever the lambda body evaluates
	// to (Function<T,R> where R genuinely differs from T, e.g. flatMap's element
	// type, a groupingBy classifier, or a Comparator.comparing key extractor).
	lambdaResultInferred
	// lambdaResultInt32 / lambdaResultInt64 / lambdaResultFloat64: the functional
	// interface pins the result to a Java primitive (ToIntFunction, Comparator's
	// int, comparingDouble, ...) regardless of what the body would infer to.
	lambdaResultInt32
	lambdaResultInt64
	lambdaResultFloat64
	// lambdaResultAny: the closure returns an unconstrained value, used where the
	// runtime accepts any Java object (Optional.orElseThrow's exception supplier).
	lambdaResultAny
)

// lambdaArgumentTypes gives one lambda argument its parameter types and result
// type explicitly, rather than deriving them from a single element type.
//
// The element-typed shape table cannot describe a call whose lambdas differ from
// each other: Collectors.toMap takes a key extractor, a value extractor and a
// merge function with three different signatures, and three-argument
// Stream.reduce takes a (U, T) -> U accumulator beside a (U, U) -> U combiner.
type lambdaArgumentTypes struct {
	// paramJavaTypes is one Java type per lambda parameter.
	paramJavaTypes []string
	// resultJavaType is the closure's result; empty means it returns nothing.
	resultJavaType string
}

// lambdaArgumentTyper computes those types per argument index for one call, or
// returns nil when it does not apply to that call (a different arity, say).
type lambdaArgumentTyper func(invocation *sitter.Node, ctx Ctx, source []byte) map[int]lambdaArgumentTypes

// lambdaArgumentTypers records them per (receiver type, method). An entry here
// takes precedence over the element-typed shape table.
var lambdaArgumentTypers = map[intrinsicKey]lambdaArgumentTyper{}

func registerLambdaArgumentTyper(typeName, methodName string, typer lambdaArgumentTyper) {
	lambdaArgumentTypers[intrinsicKey{typeName, methodName}] = typer
}

// lookupLambdaArgumentTypes runs the registered typer for a call, if any.
func lookupLambdaArgumentTypes(receiverType, methodName string, invocation *sitter.Node, ctx Ctx, source []byte) map[int]lambdaArgumentTypes {
	typer, ok := lambdaArgumentTypers[intrinsicKey{receiverType, methodName}]
	if !ok {
		return nil
	}
	return typer(invocation, ctx, source)
}

// retypeLambdaWithTypes gives a parsed lambda explicit parameter and result
// types. It is the per-argument counterpart of retypeElementLambda.
func retypeLambdaWithTypes(arg ast.Expr, paramTypes []ast.Expr, resultType ast.Expr) ast.Expr {
	funcLit, ok := arg.(*ast.FuncLit)
	if !ok || funcLit.Type == nil {
		return arg
	}
	if funcLit.Type.Params != nil {
		index := 0
		for _, field := range funcLit.Type.Params.List {
			// A parsed lambda parameter list has one name per field.
			if index < len(paramTypes) {
				field.Type = paramTypes[index]
			}
			index += max(1, len(field.Names))
		}
	}
	if resultType == nil {
		return arg
	}
	funcLit.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: resultType}}}
	// A single-expression body was lowered to a bare ExprStmt; it becomes a real
	// return now that the closure has a result type.
	if len(funcLit.Body.List) == 1 {
		if exprStmt, ok := funcLit.Body.List[0].(*ast.ExprStmt); ok {
			funcLit.Body.List[0] = &ast.ReturnStmt{Results: []ast.Expr{exprStmt.X}}
		}
	}
	// Java widens implicitly at the return, so a ToLongFunction accepts a body of
	// type int. Go has no implicit numeric conversion, so make it explicit — but
	// only for the numeric primitives, where widening is what Java is doing. A
	// conversion to any other result type would change the value rather than its
	// width.
	if isNumericPrimitiveTypeExpr(resultType) {
		convertLambdaReturns(funcLit.Body, resultType)
	}
	return arg
}

// isNumericPrimitiveTypeExpr reports whether a Go type expression is one of the
// numeric primitives a Java numeric widening can target.
func isNumericPrimitiveTypeExpr(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	switch ident.Name {
	case "int8", "int16", "int32", "int64", "float32", "float64":
		return true
	}
	return false
}

// nodeIntrinsicGenerator is an instance intrinsic that needs its own call node,
// because it inspects an argument's syntax rather than just its parsed value.
// Stream.collect is the case: which collector was passed decides the rewrite,
// and the collector's own lambdas have to be typed from the stream's elements.
type nodeIntrinsicGenerator func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr

// instanceNodeIntrinsics is consulted before the ordinary instance table.
var instanceNodeIntrinsics = map[intrinsicKey]nodeIntrinsicGenerator{}

func registerInstanceNodeIntrinsic(typeName, methodName string, gen nodeIntrinsicGenerator) {
	instanceNodeIntrinsics[intrinsicKey{typeName, methodName}] = gen
}

// lambdaShapes records, per (receiver type, method), the result type of that
// intrinsic's element-typed lambda argument.
//
// The key includes the receiver type because the same method name means
// different things on different types: Optional.map and Stream.map have
// different element types, and a name-only key would collide as soon as a second
// type registers the same method.
var lambdaShapes = map[intrinsicKey]lambdaResultKind{}

// registerLambdaShape declares that typeName.methodName takes element-typed
// lambda arguments whose closures return result. Call it next to the matching
// registerInstanceIntrinsic so the two stay in sync.
func registerLambdaShape(typeName, methodName string, result lambdaResultKind) {
	lambdaShapes[intrinsicKey{typeName, methodName}] = result
}

// lookupLambdaShape returns the declared lambda result kind for a call, if any.
func lookupLambdaShape(typeName, methodName string) (lambdaResultKind, bool) {
	kind, ok := lambdaShapes[intrinsicKey{typeName, methodName}]
	return kind, ok
}

// elementFromEnclosingTarget is an elementArg sentinel meaning the element type
// comes from the call this one is an argument to, rather than from one of this
// call's own arguments. Comparator.comparing is the motivating case: the key
// extractor's parameter type is fixed by the collection being sorted, which is
// only visible one level up.
const elementFromEnclosingTarget = -1

// enclosingTargetElementJavaTypes resolves the element type of the call that
// invocation is an argument to: the receiver's element type for an instance
// call, or its first argument's for a static one.
func enclosingTargetElementJavaTypes(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	if invocation == nil {
		return nil
	}
	argumentList := invocation.Parent()
	if argumentList == nil || argumentList.Type() != "argument_list" {
		return nil
	}
	enclosing := argumentList.Parent()
	if enclosing == nil || enclosing.Type() != "method_invocation" {
		return nil
	}
	if object := enclosing.ChildByFieldName("object"); object != nil {
		if types := receiverElementJavaTypes(object, ctx, source); len(types) > 0 {
			return types
		}
	}
	return staticIntrinsicElementJavaTypes(enclosing, 0, ctx, source)
}

// targetElementJavaTypes resolves the element type a target-typed factory should
// take: from the call it is an argument to, or, when it is not an argument at
// all, from the type it is being assigned to. `Comparator<Integer> c =
// Comparator.naturalOrder()` has no enclosing call but does have a declared
// type, and that is the only place its element type appears.
func targetElementJavaTypes(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	if types := enclosingTargetElementJavaTypes(invocation, ctx, source); len(types) > 0 {
		return types
	}
	_, typeArgs := parseJavaTypeString(ctx.expectedType)
	return typeArgs
}

// staticLambdaShape describes a static intrinsic whose lambda arguments cannot
// be typed from a receiver, because there is none, and are typed from the
// element type of one of the call's other arguments instead:
// Collections.sort(list, (a, b) -> ...) types the comparator from list.
type staticLambdaShape struct {
	// elementArg is the index of the argument carrying the element type.
	elementArg int
	// lambdaArgs are the indexes of the arguments to retype.
	lambdaArgs []int
	result     lambdaResultKind
}

// staticLambdaShapes records those shapes per (class, method).
var staticLambdaShapes = map[intrinsicKey]staticLambdaShape{}

// registerStaticLambdaShape declares that className.methodName takes lambda
// arguments at lambdaArgs whose parameters are the element type of the argument
// at elementArg, and whose closures return result.
func registerStaticLambdaShape(className, methodName string, elementArg int, lambdaArgs []int, result lambdaResultKind) {
	staticLambdaShapes[intrinsicKey{className, methodName}] = staticLambdaShape{
		elementArg: elementArg,
		lambdaArgs: lambdaArgs,
		result:     result,
	}
}

// javaContainerElementTypes returns the Java element type(s) of a container
// expression's type: the type arguments of a generic type, the component type of
// an array, or the implied element of a primitive stream.
func javaContainerElementTypes(javaType string) []string {
	javaType = strings.TrimSpace(javaType)
	if component, ok := strings.CutSuffix(javaType, "[]"); ok {
		if component = strings.TrimSpace(component); component != "" {
			return []string{component}
		}
		return nil
	}
	base, typeArgs := parseJavaTypeString(javaType)
	if len(typeArgs) > 0 {
		return typeArgs
	}
	if element, ok := primitiveStreamElementJavaTypes[stripJavaQualifier(base)]; ok {
		return []string{element}
	}
	return nil
}

// staticIntrinsicElementJavaTypes resolves the element type(s) that a static
// intrinsic's lambda arguments should be typed from.
func staticIntrinsicElementJavaTypes(invocation *sitter.Node, elementArg int, ctx Ctx, source []byte) []string {
	if elementArg == elementFromEnclosingTarget {
		return targetElementJavaTypes(invocation, ctx, source)
	}
	argNode := invocationArgumentNode(invocation, elementArg)
	if argNode == nil {
		return nil
	}
	javaType, ok := inferExprJavaType(argNode, ctx, source)
	if !ok {
		return nil
	}
	return javaContainerElementTypes(javaType)
}

// primitiveStreamElementJavaTypes maps the primitive stream types onto the Java
// element type they iterate. Unlike Stream<T> these carry no type argument, so
// their element type cannot be read off the receiver's type arguments.
var primitiveStreamElementJavaTypes = map[string]string{
	"IntStream":    "int",
	"LongStream":   "long",
	"DoubleStream": "double",
}

// receiverElementJavaTypes returns the Java element type(s) of a receiver: its
// type arguments for a generic type (one for Stream<T>/Optional<T>, two for
// Map<K,V>), or the implied element type for a primitive stream. It returns nil
// when the receiver's type is unknown or carries no element type.
func receiverElementJavaTypes(objectNode *sitter.Node, ctx Ctx, source []byte) []string {
	javaType, ok := inferExprJavaType(objectNode, ctx, source)
	if !ok {
		return nil
	}
	base, typeArgs := parseJavaTypeString(javaType)
	if len(typeArgs) > 0 {
		return typeArgs
	}
	if element, ok := primitiveStreamElementJavaTypes[stripJavaQualifier(base)]; ok {
		return []string{element}
	}
	return nil
}

// receiverElementTypeExpr returns the Go type expression for a single-element
// receiver's element type (e.g. int32 for Optional<Integer> or for IntStream), or
// nil when it cannot be determined. Receivers with several type arguments (a
// Map's key and value) have no single element type and return nil.
func receiverElementTypeExpr(objectNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	elementTypes := receiverElementJavaTypes(objectNode, ctx, source)
	if len(elementTypes) != 1 {
		return nil
	}
	return javaTypeStringToGoTypeExpr(elementTypes[0], inScopeTypeParameters(ctx), ctx)
}

// retypeElementLambda rewrites a parsed lambda whose parameter types are the
// placeholder `any` so each parameter takes elementType, and gives the closure
// the result type implied by kind. A single-expression body (lowered to a bare
// ExprStmt) becomes a real return unless the kind is void. Non-lambda args and
// already-typed lambdas are returned unchanged.
func retypeElementLambda(arg, elementType, mappedResult ast.Expr, kind lambdaResultKind) ast.Expr {
	funcLit, ok := arg.(*ast.FuncLit)
	if !ok || funcLit.Type == nil {
		return arg
	}
	// Java requires explicitly declared lambda parameter types to match the target
	// function, so both inferred and explicit parameters use the receiver element
	// type here. A zero-parameter lambda is a Supplier: it has nothing to retype
	// but still needs the result type applied below.
	if funcLit.Type.Params != nil {
		for _, field := range funcLit.Type.Params.List {
			field.Type = elementType
		}
	}

	if funcLit.Type.Results != nil {
		return arg
	}

	var resultType ast.Expr
	switch kind {
	case lambdaResultBool:
		resultType = &ast.Ident{Name: "bool"}
	case lambdaResultVoid:
		resultType = nil
	default:
		// A mapper (Function<T,R>) may change the type, and a primitive-result
		// interface (ToIntFunction, Comparator) pins it outright; both arrive as a
		// non-nil mappedResult. Otherwise infer R from the body's Go expression
		// where recognizable (e.g. a string concatenation lowers to fmt.Sprintf and
		// is a string), and fall back to the element type, which is correct for
		// identity/arithmetic maps and for the operator interfaces.
		resultType = elementType
		if mappedResult != nil {
			resultType = mappedResult
		} else if len(funcLit.Body.List) == 1 {
			if exprStmt, ok := funcLit.Body.List[0].(*ast.ExprStmt); ok {
				if inferred := goExprResultType(exprStmt.X); inferred != nil {
					resultType = inferred
				}
			}
		}
	}

	if resultType != nil {
		funcLit.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: resultType}}}
		// A single-expression body was lowered to one ExprStmt; turn it into a
		// return now that the closure has a result type.
		if len(funcLit.Body.List) == 1 {
			if exprStmt, ok := funcLit.Body.List[0].(*ast.ExprStmt); ok {
				funcLit.Body.List[0] = &ast.ReturnStmt{Results: []ast.Expr{exprStmt.X}}
			}
		}
		// A functional interface that pins its result to a primitive applies
		// Java's implicit widening at the return: ToLongFunction accepts a body
		// of type int. Go has no implicit numeric conversion, so make it explicit.
		if pinnedPrimitiveLambdaResult(kind) {
			convertLambdaReturns(funcLit.Body, resultType)
		}
	}
	return arg
}

// pinnedPrimitiveLambdaResult reports whether a result kind fixes the closure's
// result to a Java primitive, so a body of a narrower type must be converted.
func pinnedPrimitiveLambdaResult(kind lambdaResultKind) bool {
	switch kind {
	case lambdaResultInt32, lambdaResultInt64, lambdaResultFloat64:
		return true
	}
	return false
}

// convertLambdaReturns wraps every value this closure returns in a conversion to
// resultType, including returns nested inside an if, loop or switch. Returns
// belonging to a nested function literal are left alone, since those are a
// different closure's.
func convertLambdaReturns(body *ast.BlockStmt, resultType ast.Expr) {
	if body == nil {
		return
	}
	ast.Inspect(body, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.FuncLit:
			// A nested closure has its own result type; do not descend.
			return false
		case *ast.ReturnStmt:
			if len(typed.Results) == 1 {
				typed.Results[0] = &ast.CallExpr{Fun: resultType, Args: []ast.Expr{typed.Results[0]}}
			}
			return false
		}
		return true
	})
}

// invocationArgumentNode returns the index'th argument node of a method
// invocation, or nil when there is no such argument.
func invocationArgumentNode(invocation *sitter.Node, index int) *sitter.Node {
	if invocation == nil || index < 0 {
		return nil
	}
	argsNode := invocation.ChildByFieldName("arguments")
	if argsNode == nil || index >= int(argsNode.NamedChildCount()) {
		return nil
	}
	return argsNode.NamedChild(index)
}

// invocationArgumentCount returns how many arguments a method invocation passes.
func invocationArgumentCount(invocation *sitter.Node) int {
	if invocation == nil {
		return 0
	}
	argsNode := invocation.ChildByFieldName("arguments")
	if argsNode == nil {
		return 0
	}
	return int(argsNode.NamedChildCount())
}

// intrinsicLambdaResultTypeExpr returns the Go type that the index'th lambda
// argument's closure must return, or nil to let retypeElementLambda apply its
// own default (the receiver's element type, refined by goExprResultType).
//
// A primitive-result functional interface pins the type outright. A free result
// type (Function<T,R>) is recovered by binding the lambda's parameters to the
// receiver's element types and inferring what its body evaluates to.
func intrinsicLambdaResultTypeExpr(objectNode *sitter.Node, argIndex int, elementJavaTypes []string, kind lambdaResultKind, ctx Ctx, source []byte) ast.Expr {
	switch kind {
	case lambdaResultInt32:
		return &ast.Ident{Name: "int32"}
	case lambdaResultInt64:
		return &ast.Ident{Name: "int64"}
	case lambdaResultFloat64:
		return &ast.Ident{Name: "float64"}
	case lambdaResultAny:
		return &ast.Ident{Name: "any"}
	case lambdaResultInferred:
		// Handled below.
	default:
		// lambdaResultElement pins the result to the element type, and bool/void
		// carry no inference; none of them consult the lambda body.
		return nil
	}

	lambda := invocationArgumentNode(objectNode.Parent(), argIndex)
	if lambda == nil {
		return nil
	}
	resultJavaType, ok := inferLambdaResultJavaType(lambda, elementJavaTypes, ctx, source)
	if !ok {
		return nil
	}
	return javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx)
}

// goExprResultType returns a Go type expression for a lowered Go expression when
// it can be recognized, or nil to fall back to a caller default. It recognizes
// the string-concatenation form (fmt.Sprintf(...)) as a string.
func goExprResultType(expr ast.Expr) ast.Expr {
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if base, ok := sel.X.(*ast.Ident); ok && base.Name == "fmt" && sel.Sel != nil {
				switch sel.Sel.Name {
				case "Sprintf", "Sprint":
					return &ast.Ident{Name: "string"}
				}
			}
		}
	}
	return nil
}

// tryStaticIntrinsic attempts to rewrite a static call `Class.method(args)` (or a
// static-field access `Class.FIELD`) where Class is a recognized standard-library
// type that is not user-defined.
func tryStaticIntrinsic(objectNode *sitter.Node, methodName string, source []byte, ctx Ctx) (ast.Expr, bool) {
	className, ok := intrinsicStaticClassName(objectNode, ctx, source)
	if !ok {
		return nil, false
	}
	if className == "String" && methodName == "valueOf" && executionExpr(ctx) != nil {
		if parent := objectNode.Parent(); parent != nil {
			if arguments := parent.ChildByFieldName("arguments"); arguments != nil && arguments.NamedChildCount() == 1 {
				argumentNode := arguments.NamedChild(0)
				if javaType, inferred := inferExprJavaType(argumentNode, ctx, source); inferred {
					base, _ := parseJavaTypeString(javaType)
					if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.IsEnum {
						return &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ParseExpr(argumentNode, source, ctx),
								Sel: &ast.Ident{Name: executionStringMethodName(scope)},
							},
							Args: []ast.Expr{executionExpr(ctx)},
						}, true
					}
					return javaStringValueOfForType(javaType, ParseExpr(argumentNode, source, ctx), ctx), true
				}
			}
		}
	}

	gen, ok := staticIntrinsics[intrinsicKey{className, methodName}]
	if !ok {
		return nil, false
	}

	args := intrinsicArgs(objectNode, methodName, source, ctx)
	if derive, ok := staticIntrinsicTypeArgs[intrinsicKey{className, methodName}]; ok {
		ctx.intrinsicTypeArgs = derive(objectNode.Parent(), ctx, source)
	}
	// A static call has no receiver to take an element type from, so a declared
	// static shape names the argument that carries it instead.
	if shape, ok := staticLambdaShapes[intrinsicKey{className, methodName}]; ok {
		invocation := objectNode.Parent()
		elementJavaTypes := staticIntrinsicElementJavaTypes(invocation, shape.elementArg, ctx, source)
		if len(elementJavaTypes) == 1 {
			elementType := javaTypeStringToGoTypeExpr(elementJavaTypes[0], inScopeTypeParameters(ctx), ctx)
			for _, index := range shape.lambdaArgs {
				if index >= len(args) {
					continue
				}
				mappedResult := staticLambdaResultTypeExpr(invocation, index, elementJavaTypes, shape.result, ctx, source)
				args[index] = retypeElementLambda(args[index], elementType, mappedResult, shape.result)
			}
		}
	}
	if result := gen(nil, args, ctx); result != nil {
		return result, true
	}
	return nil, false
}

// staticLambdaResultTypeExpr is intrinsicLambdaResultTypeExpr for a static call,
// where the invocation node is reached directly rather than through a receiver.
func staticLambdaResultTypeExpr(invocation *sitter.Node, argIndex int, elementJavaTypes []string, kind lambdaResultKind, ctx Ctx, source []byte) ast.Expr {
	switch kind {
	case lambdaResultInt32:
		return &ast.Ident{Name: "int32"}
	case lambdaResultInt64:
		return &ast.Ident{Name: "int64"}
	case lambdaResultFloat64:
		return &ast.Ident{Name: "float64"}
	case lambdaResultAny:
		return &ast.Ident{Name: "any"}
	case lambdaResultInferred:
		// Handled below.
	default:
		return nil
	}
	resultJavaType, ok := inferLambdaResultJavaType(invocationArgumentNode(invocation, argIndex), elementJavaTypes, ctx, source)
	if !ok {
		return nil
	}
	return javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx)
}

// tryStaticFieldIntrinsic attempts to rewrite a static-field access such as
// Integer.MAX_VALUE.
func tryStaticFieldIntrinsic(objectNode *sitter.Node, fieldName string, source []byte, ctx Ctx) (ast.Expr, bool) {
	className, ok := intrinsicStaticClassName(objectNode, ctx, source)
	if !ok {
		return nil, false
	}
	gen, ok := staticFieldIntrinsics[intrinsicKey{className, fieldName}]
	if !ok {
		return nil, false
	}
	if result := gen(ctx); result != nil {
		return result, true
	}
	return nil, false
}

// intrinsicArgs parses the argument list of the method invocation that contains
// objectNode. objectNode is the receiver; its parent is the method_invocation.
func intrinsicArgs(objectNode *sitter.Node, methodName string, source []byte, ctx Ctx) []ast.Expr {
	parent := objectNode.Parent()
	if parent == nil {
		return nil
	}
	argListNode := parent.ChildByFieldName("arguments")
	if argListNode != nil {
		// A per-argument typer seeds each lambda with its own parameter types.
		if receiverType, ok := intrinsicReceiverTypeName(objectNode, ctx, source); ok {
			if perArgument := lookupLambdaArgumentTypes(receiverType, methodName, parent, ctx, source); perArgument != nil {
				return parseArgumentsWithPerArgumentTypes(argListNode, perArgument, source, ctx)
			}
		}
		if elementJavaType, lambdaArgs, ok := intrinsicLambdaParameterJavaType(objectNode, methodName, ctx, source); ok {
			return parseArgumentsWithLambdaParameterType(argListNode, elementJavaType, lambdaArgs, source, ctx)
		}
	}
	var expectedTypes []string
	if (methodName == "submit" || methodName == "execute") && argListNode != nil && argListNode.NamedChildCount() == 1 {
		if javaType, ok := inferExprJavaType(objectNode, ctx, source); ok {
			base, _ := parseJavaTypeString(javaType)
			if stripJavaQualifier(base) == "ExecutorService" && resolveClassScopeByQualifiedName(ctx, base) == nil {
				expectedTypes = []string{"Runnable"}
			}
		}
	}
	return parseArgumentListWithExpectedTypes(argListNode, source, ctx, expectedTypes)
}

// intrinsicLambdaParameterJavaType resolves the Java type an intrinsic's lambda
// parameters take, together with the argument positions to apply it to. A nil
// position list means every lambda argument of the call.
//
// A lambda argument is parsed before its target type is known, so without this
// its parameters come back as `any` and the body cannot resolve any member
// access on them.
func intrinsicLambdaParameterJavaType(objectNode *sitter.Node, methodName string, ctx Ctx, source []byte) (string, []int, bool) {
	if receiverType, ok := intrinsicReceiverTypeName(objectNode, ctx, source); ok {
		if _, declared := lookupLambdaShape(receiverType, methodName); declared {
			if elementTypes := receiverElementJavaTypes(objectNode, ctx, source); len(elementTypes) == 1 {
				return elementTypes[0], nil, true
			}
		}
	}
	if className, ok := intrinsicStaticClassName(objectNode, ctx, source); ok {
		if shape, declared := staticLambdaShapes[intrinsicKey{className, methodName}]; declared {
			elementTypes := staticIntrinsicElementJavaTypes(objectNode.Parent(), shape.elementArg, ctx, source)
			if len(elementTypes) == 1 {
				return elementTypes[0], shape.lambdaArgs, true
			}
		}
	}
	return "", nil, false
}

// parseArgumentsWithLambdaParameterType parses a call's arguments, seeding each
// selected lambda argument's parse context so its parameters carry
// elementJavaType instead of the `any` placeholder.
func parseArgumentsWithLambdaParameterType(argListNode *sitter.Node, elementJavaType string, lambdaArgs []int, source []byte, ctx Ctx) []ast.Expr {
	selected := func(index int) bool {
		if lambdaArgs == nil {
			return true
		}
		for _, candidate := range lambdaArgs {
			if candidate == index {
				return true
			}
		}
		return false
	}

	args := make([]ast.Expr, 0, argListNode.NamedChildCount())
	for index, argNode := range nodeutil.NamedChildrenOf(argListNode) {
		argCtx := ctx.Clone()
		if argNode.Type() == "lambda_expression" && selected(index) {
			names := lambdaParameterNames(argNode.ChildByFieldName("parameters"), source)
			argCtx.lambdaParameterJavaTypes = make([]string, len(names))
			for i := range argCtx.lambdaParameterJavaTypes {
				argCtx.lambdaParameterJavaTypes[i] = elementJavaType
			}
		}
		args = append(args, ParseExpr(argNode, source, argCtx))
	}
	return args
}

// parseArgumentsWithPerArgumentTypes parses a call's arguments, seeding each
// lambda listed in perArgument with its own parameter types so its body can
// resolve member access on them.
func parseArgumentsWithPerArgumentTypes(argListNode *sitter.Node, perArgument map[int]lambdaArgumentTypes, source []byte, ctx Ctx) []ast.Expr {
	args := make([]ast.Expr, 0, argListNode.NamedChildCount())
	for index, argNode := range nodeutil.NamedChildrenOf(argListNode) {
		argCtx := ctx.Clone()
		if types, ok := perArgument[index]; ok && argNode.Type() == "lambda_expression" {
			argCtx.lambdaParameterJavaTypes = append([]string(nil), types.paramJavaTypes...)
		}
		args = append(args, ParseExpr(argNode, source, argCtx))
	}
	return args
}

// intrinsicReceiverTypeName resolves the unqualified Java type of a receiver
// expression for instance-intrinsic lookup. It returns false when the type is
// unknown or when the receiver is itself a user-defined class (those calls must
// go through the normal resolution path).
func intrinsicReceiverTypeName(objectNode *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	javaType, ok := inferExprJavaType(objectNode, ctx, source)
	if !ok {
		return "", false
	}
	base, _ := parseJavaTypeString(javaType)
	if erased, bounded := javaTypeParameterErasure(base, ctx); bounded {
		base, _ = parseJavaTypeString(erased)
	}
	name := stripJavaQualifier(base)
	if name == "" {
		return "", false
	}
	// A receiver whose type resolves to a user-defined class is never an
	// intrinsic; let normal method resolution handle it.
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	return name, true
}

// intrinsicStaticClassName returns the class name for a static call when the
// receiver is a bare identifier (e.g. `Math`, `Integer`) that is not a
// user-defined class.

func inferIntrinsicMethodResultType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if node == nil || node.Type() != "method_invocation" {
		return "", false
	}
	objectNode := node.ChildByFieldName("object")
	nameNode := node.ChildByFieldName("name")
	if objectNode == nil || nameNode == nil {
		return "", false
	}
	methodName := nameNode.Content(source)
	if receiverType, ok := intrinsicReceiverTypeName(objectNode, ctx, source); ok {
		if resultType := instanceIntrinsicResultTypes[intrinsicKey{receiverType, methodName}]; resultType != "" {
			return resultType, true
		}
	}
	if className, ok := intrinsicStaticClassName(objectNode, ctx, source); ok {
		if derive, ok := staticIntrinsicDerivedResultTypes[intrinsicKey{className, methodName}]; ok {
			if resultType, derived := derive(node, ctx, source); derived {
				return resultType, true
			}
		}
		if resultType := staticIntrinsicResultTypes[intrinsicKey{className, methodName}]; resultType != "" {
			return resultType, true
		}
	}
	return "", false
}

func inferIntrinsicFieldResultType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if node == nil || node.Type() != "field_access" {
		return "", false
	}
	objectNode := node.ChildByFieldName("object")
	fieldNode := node.ChildByFieldName("field")
	if objectNode == nil || fieldNode == nil {
		return "", false
	}
	className, ok := intrinsicStaticClassName(objectNode, ctx, source)
	if !ok {
		return "", false
	}
	resultType := staticFieldIntrinsicResultTypes[intrinsicKey{className, fieldNode.Content(source)}]
	return resultType, resultType != ""
}

func intrinsicStaticClassName(objectNode *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if objectNode == nil || objectNode.Type() != "identifier" {
		return "", false
	}
	name := objectNode.Content(source)
	if name == "" {
		return "", false
	}
	// If the identifier names a local/field, it is a value, not a class.
	if _, ok := inferIdentifierJavaType(name, ctx); ok {
		return "", false
	}
	// A user-defined class with a matching scope is handled by normal static
	// resolution.
	if resolveClassScopeByQualifiedName(ctx, name) != nil {
		return "", false
	}
	return name, true
}

// --- small AST construction helpers shared by the intrinsic generators ---

func callIdent(name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: &ast.Ident{Name: name}, Args: args}
}

func stdjavaCall(ctx Ctx, name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: stdjavaQualifiedExpr(name, ctx), Args: args}
}

// stdjavaGenericCall emits stdjava.Name[typeArgs](args), used for the generic
// collection constructors (e.g. stdjava.NewList[string]()).
func stdjavaGenericCall(ctx Ctx, name string, typeArgs, args []ast.Expr) *ast.CallExpr {
	fun := applyTypeArguments(stdjavaQualifiedExpr(name, ctx), typeArgs)
	return &ast.CallExpr{Fun: fun, Args: args}
}

func pkgCall(ctx Ctx, javaPkg, name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: qualifiedNameExpr(name, javaPkg, ctx), Args: args}
}

func methodCall(recv ast.Expr, method string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: method}},
		Args: args,
	}
}
