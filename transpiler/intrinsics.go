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

// constructorGenerator builds the Go replacement for a `new Type(args)`
// expression. Returning nil falls through to the normal constructor path.
type constructorGenerator func(args []ast.Expr, ctx Ctx) ast.Expr

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

// tryConstructorIntrinsic attempts to rewrite a `new className(args)` expression
// via the constructor intrinsics table. It only fires when className is not a
// user-defined class.
func tryConstructorIntrinsic(className string, args []ast.Expr, ctx Ctx) (ast.Expr, bool) {
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
	if result := gen(args, ctx); result != nil {
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
	if result := gen(recv, args, ctx); result != nil {
		return result, true
	}
	return nil, false
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

func pkgCall(ctx Ctx, javaPkg, name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: qualifiedNameExpr(name, javaPkg, ctx), Args: args}
}

func methodCall(recv ast.Expr, method string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun:  &ast.SelectorExpr{X: recv, Sel: &ast.Ident{Name: method}},
		Args: args,
	}
}
