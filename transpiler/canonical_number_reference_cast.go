package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// A runtime Number reference has a structural Go interface representation, but
// Java checkcast tests declared ancestry. This also applies after a class-owned
// generic binder has been lowered to its Number erasure. Source classes and
// un-erased binders keep their own cast paths.
func lowerCanonicalNumberReferenceCast(value ast.Expr, targetJavaType string, ctx Ctx) (ast.Expr, bool) {
	base, rank := javaArrayTypeParts(strings.TrimSpace(targetJavaType))
	if rank != 0 {
		return nil, false
	}
	base, arguments := parseJavaTypeString(base)
	if len(arguments) != 0 {
		return nil, false
	}
	if _, binder := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); binder {
		return nil, false
	}
	if resolveClassScopeByQualifiedName(ctx, base) != nil || qualifyDeclaredNominalReference(base, ctx) != "java.lang.Number" {
		return nil, false
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{stdjavaQualifiedExpr("JavaNumber", ctx)}, []ast.Expr{value, stdjavaQualifiedExpr("NumberTypeID", ctx)}), true
}
