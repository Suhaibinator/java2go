package transpiler

import (
	"go/ast"

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
	instanceIntrinsics    = map[intrinsicKey]intrinsicGenerator{}
	staticIntrinsics      = map[intrinsicKey]intrinsicGenerator{}
	staticFieldIntrinsics = map[intrinsicKey]func(ctx Ctx) ast.Expr{}
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
	args := intrinsicArgs(objectNode, methodName, source, ctx)

	// Methods whose argument is a lambda over the receiver's element type
	// (Optional.map, Optional.ifPresent, ...) parse the lambda without knowing
	// its parameter type, so it comes back as `func(x any)`. Re-type the lambda
	// from the receiver's element type so the generated Go closure is well-typed.
	if elementTypedLambdaMethods[methodName] {
		if elem := receiverElementTypeExpr(objectNode, ctx, source); elem != nil {
			for i, a := range args {
				args[i] = retypeLambdaParam(a, elem)
			}
		}
	}

	if result := gen(recv, args, ctx); result != nil {
		return result, true
	}
	return nil, false
}

// elementTypedLambdaMethods are intrinsic instance methods whose lambda argument
// takes the receiver's element type as its single parameter.
var elementTypedLambdaMethods = map[string]bool{
	"map":       true,
	"ifPresent": true,
	"filter":    true,
	"forEach":   true,
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

// retypeLambdaParam, given a parsed single-parameter lambda whose parameter type
// is the placeholder `any`, rewrites the parameter to paramType and — for a
// single-expression body that was lowered to a bare ExprStmt — turns it into a
// `return` with the parameter type as the (inferred) result type. Non-lambda
// args and already-typed lambdas are returned unchanged.
func retypeLambdaParam(arg ast.Expr, paramType ast.Expr) ast.Expr {
	funcLit, ok := arg.(*ast.FuncLit)
	if !ok || funcLit.Type == nil || funcLit.Type.Params == nil {
		return arg
	}
	if len(funcLit.Type.Params.List) != 1 {
		return arg
	}
	field := funcLit.Type.Params.List[0]
	// Only adjust the placeholder `any` parameter type, so an explicitly typed
	// lambda is left untouched.
	if ident, ok := field.Type.(*ast.Ident); !ok || ident.Name != "any" {
		return arg
	}
	field.Type = paramType

	// A single-expression lambda body is lowered to one ExprStmt; with the
	// parameter now typed, give the closure a result type and a real return so it
	// is a valid Go func value. The result type is approximated as the parameter
	// type (correct for identity/arithmetic maps; richer inference is a TODO).
	if funcLit.Type.Results == nil && len(funcLit.Body.List) == 1 {
		if exprStmt, ok := funcLit.Body.List[0].(*ast.ExprStmt); ok {
			funcLit.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: paramType}}}
			funcLit.Body.List[0] = &ast.ReturnStmt{Results: []ast.Expr{exprStmt.X}}
		}
	}
	return arg
}

// tryStaticIntrinsic attempts to rewrite a static call `Class.method(args)` (or a
// static-field access `Class.FIELD`) where Class is a recognized standard-library
// type that is not user-defined.
func tryStaticIntrinsic(objectNode *sitter.Node, methodName string, source []byte, ctx Ctx) (ast.Expr, bool) {
	className, ok := intrinsicStaticClassName(objectNode, ctx, source)
	if !ok {
		return nil, false
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
	return parseArgumentListWithExpectedTypes(argListNode, source, ctx, nil)
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
