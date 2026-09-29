package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// String is final, so a type-variable argument selects the String overload
// exactly when a declared upper-bound chain reaches canonical java.lang.String.
// Resolve bounds in their declaration context, preserving captured identities.
func intrinsicStringBoundType(actual symbol.JavaType, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	if binding, found := resolveReferenceTypeParameter(actual, ctx); found {
		key := identityKeyForTypeParameter(binding.parameter)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		for _, bound := range binding.parameter.Bounds {
			if intrinsicStringBoundType(bound, binding.context, visiting) {
				return true
			}
		}
		return false
	}
	base, _ := parseJavaTypeString(actual.Original)
	if actual.TypeParameterBindings[base] != nil {
		return false
	}
	return javaInferenceSameType(actual.Original, "java.lang.String", ctx)
}

// Intrinsic parameters use concrete runtime storage even when Java passes a
// reference through a type-variable bound. Go requires an explicit conversion
// from S constrained by string to string. This preserves the native null
// sentinel and performs no Java toString callback or reference allocation.
func convertIntrinsicStringBoundArgument(value ast.Expr, arg *sitter.Node, expected string, ctx Ctx, source []byte) ast.Expr {
	if !javaInferenceSameType(expected, "java.lang.String", ctx) {
		return value
	}
	target, ok := javaTypeStringToGoTypeExpr(expected, inScopeTypeParameters(ctx), ctx).(*ast.Ident)
	if !ok || target.Name != "string" {
		return value
	}
	actual, known := inferExprJavaType(arg, ctx, source)
	if !known {
		return value
	}
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: actual}, ctx); !bound {
		return value
	}
	if !intrinsicStringBoundType(symbol.JavaType{Original: actual}, ctx, map[typeParameterIdentityKey]bool{}) {
		return value
	}
	return &ast.CallExpr{Fun: target, Args: []ast.Expr{value}}
}
