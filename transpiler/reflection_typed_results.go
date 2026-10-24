package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	for _, name := range []string{"getConstructor", "getDeclaredConstructor"} {
		registerInstanceIntrinsicDerivedResultType("Class", name, reflectionDeclaredResultType)
	}
	registerInstanceIntrinsicDerivedResultType("Constructor", "newInstance", reflectionDeclaredResultType)
	for _, receiver := range []string{"Class", "Field"} {
		registerInstanceIntrinsicDerivedResultType(receiver, "getAnnotation", reflectionDeclaredResultType)
	}
}

// Reflection's physical runtime values are erased, but its JDK declarations
// retain Class<T>, Constructor<T>, and <A extends Annotation> result types.
// Admission is performed by the ordinary intrinsic owner resolver before this
// derivation runs; a source declaration with the same spelling is not an API.
func reflectionDeclaredResultType(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	name := invocation.ChildByFieldName("name")
	object := invocation.ChildByFieldName("object")
	if name == nil || object == nil {
		return "", false
	}
	method := name.Content(source)
	if method == "getAnnotation" {
		if invocationArgumentCount(invocation) != 1 {
			return "", false
		}
		argument := invocationArgumentNode(invocation, 0)
		if typ, ok := inferExprJavaType(argument, ctx, source); ok {
			base, args := parseJavaTypeString(typ)
			if owner, admitted := canonicalIntrinsicOwner(base, ctx); admitted && owner == "java.lang.Class" && len(args) == 1 {
				return reflectionTypeArgumentReadType(args[0]), true
			}
		}
		return "", false
	}
	typ, known := inferExprJavaType(object, ctx, source)
	if !known {
		return "", false
	}
	_, args := parseJavaTypeString(typ)
	if len(args) != 1 {
		return "", false
	}
	if method == "getConstructor" || method == "getDeclaredConstructor" {
		return "java.lang.reflect.Constructor<" + args[0] + ">", true
	}
	if method == "newInstance" {
		return reflectionTypeArgumentReadType(args[0]), true
	}
	return "", false
}

func reflectionTypeArgumentReadType(argument string) string {
	argument = strings.TrimSpace(argument)
	if argument == "?" || strings.HasPrefix(argument, "?") && strings.HasPrefix(strings.TrimSpace(argument[1:]), "super") {
		return "java.lang.Object"
	}
	return readableWildcardProjection(argument)
}

func reflectionAnnotationIntrinsic(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	args := intrinsicArgs(invocation.ChildByFieldName("object"), "getAnnotation", source, ctx)
	call := selectorCall(recv, "GetAnnotationExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...))
	return reflectionTypedResultExpr(call, invocation, ctx, source)
}

// Project the erased result with the same nominal, null-preserving check used
// by Java reference reads. A Go assertion alone would reject superclass views
// and panic on null instead of implementing Java's checkcast semantics.
func reflectionTypedResultExpr(value ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	result, known := reflectionDeclaredResultType(invocation, ctx, source)
	if !known {
		return value
	}
	if erased, ok := currentErasedCallableOwnerTypeParameterErasure(result, ctx); ok {
		result = erased
	}
	descriptor, ok := javaSourceTypeDescriptorExpr(result, ctx)
	if !ok {
		return value
	}
	target := abstractClassToInterface(javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx), result, ctx)
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{target}, []ast.Expr{value, descriptor})
}
