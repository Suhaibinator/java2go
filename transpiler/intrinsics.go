package transpiler

import (
	"go/ast"

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

	gen, ok := instanceIntrinsics[intrinsicKey{receiverType, methodName}]
	if !ok {
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
	args := intrinsicArgs(objectNode, methodName, source, ctx)

	// A lambda argument to one of these stdlib methods is parsed before its
	// parameter type is known, so it comes back as `func(x any)`. Re-type each
	// lambda parameter from the receiver's element type and give the closure the
	// result type the functional interface requires (predicate -> bool, consumer
	// -> void, mapper -> the value's type), so the generated Go is well-typed.
	if kind, ok := elementLambdaMethods[methodName]; ok {
		if elem := receiverElementTypeExpr(objectNode, ctx, source); elem != nil {
			var mappedResult ast.Expr
			if methodName == "map" {
				if receiverJavaType, inferred := inferExprJavaType(objectNode, ctx, source); inferred {
					_, receiverArgs := parseJavaTypeString(receiverJavaType)
					if parent := objectNode.Parent(); parent != nil {
						if resultJavaType, inferred := inferStreamMapResultType(parent, receiverArgs, ctx, source); inferred {
							mappedResult = javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx)
						}
					}
				}
			}
			for i, a := range args {
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
)

// elementLambdaMethods maps an intrinsic instance method to the result kind of
// its element-typed lambda argument.
var elementLambdaMethods = map[string]lambdaResultKind{
	"map":       lambdaResultElement,
	"reduce":    lambdaResultElement,
	"ifPresent": lambdaResultVoid,
	"forEach":   lambdaResultVoid,
	"filter":    lambdaResultBool,
	"anyMatch":  lambdaResultBool,
	"allMatch":  lambdaResultBool,
	"noneMatch": lambdaResultBool,
}

// receiverElementTypeExpr returns the Go type expression for a single-type-arg
// receiver's element type (e.g. for an Optional<Integer> receiver, int32), or nil
// when it cannot be determined.
func receiverElementTypeExpr(objectNode *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	javaType, ok := inferExprJavaType(objectNode, ctx, source)
	if !ok {
		return nil
	}
	_, typeArgs := parseJavaTypeString(javaType)
	if len(typeArgs) != 1 {
		return nil
	}
	return javaTypeStringToGoTypeExpr(typeArgs[0], inScopeTypeParameters(ctx), ctx)
}

// retypeElementLambda rewrites a parsed lambda whose parameter types are the
// placeholder `any` so each parameter takes elementType, and gives the closure
// the result type implied by kind. A single-expression body (lowered to a bare
// ExprStmt) becomes a real return unless the kind is void. Non-lambda args and
// already-typed lambdas are returned unchanged.
func retypeElementLambda(arg, elementType, mappedResult ast.Expr, kind lambdaResultKind) ast.Expr {
	funcLit, ok := arg.(*ast.FuncLit)
	if !ok || funcLit.Type == nil || funcLit.Type.Params == nil || len(funcLit.Type.Params.List) == 0 {
		return arg
	}
	// Java requires explicitly declared lambda parameter types to match the target
	// function, so both inferred and explicit parameters use the receiver element
	// type here.
	for _, field := range funcLit.Type.Params.List {
		field.Type = elementType
	}

	if funcLit.Type.Results != nil {
		return arg
	}

	var resultType ast.Expr
	switch kind {
	case lambdaResultBool:
		resultType = &ast.Ident{Name: "bool"}
	case lambdaResultElement:
		// A mapper (Function<T,R>) may change the type. Infer R from the body's Go
		// expression where recognizable (e.g. a string concatenation lowers to
		// fmt.Sprintf and is a string); otherwise default to the element type,
		// which is correct for identity/arithmetic maps.
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
	case lambdaResultVoid:
		resultType = nil
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
	}
	return arg
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
								Sel: &ast.Ident{Name: enumExecutionStringMethodName(scope)},
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
	if result := gen(nil, args, ctx); result != nil {
		return result, true
	}
	return nil, false
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
	if _, elementLambda := elementLambdaMethods[methodName]; elementLambda && argListNode != nil {
		if receiverJavaType, inferred := inferExprJavaType(objectNode, ctx, source); inferred {
			_, receiverArgs := parseJavaTypeString(receiverJavaType)
			if len(receiverArgs) == 1 {
				args := make([]ast.Expr, 0, argListNode.NamedChildCount())
				for _, argNode := range nodeutil.NamedChildrenOf(argListNode) {
					argCtx := ctx.Clone()
					if argNode.Type() == "lambda_expression" {
						parameters := argNode.ChildByFieldName("parameters")
						parameterCount := 0
						if parameters != nil {
							parameterCount = int(parameters.NamedChildCount())
							if parameters.Type() != "inferred_parameters" && parameters.Type() != "formal_parameters" {
								parameterCount = 1
							}
						}
						argCtx.lambdaParameterJavaTypes = make([]string, parameterCount)
						for index := range argCtx.lambdaParameterJavaTypes {
							argCtx.lambdaParameterJavaTypes[index] = receiverArgs[0]
						}
					}
					args = append(args, ParseExpr(argNode, source, argCtx))
				}
				return args
			}
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
