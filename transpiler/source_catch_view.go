package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Source declarations take precedence over simple builtin exception spellings.
// Use the same physical view and nominal descriptor as an ordinary source cast.
func sourceCatchDescriptor(javaType string, ctx Ctx) (ast.Expr, bool) {
	base, _ := parseJavaTypeString(javaType)
	if resolveClassScopeByQualifiedName(ctx, base) == nil {
		return nil, false
	}
	descriptor, known := javaTypeDescriptorExpr(javaType, ctx)
	if !known {
		return nil, false
	}
	return descriptor, true
}

// A multi-catch variable has the alternatives' least upper bound, not their
// first type. Until that source-common-base view is proved, retain the existing
// Throwable representation for multi-catch instead of applying a narrow cast.
func sourceCatchVariableView(catchTypes []string, recoveredName string, ctx Ctx) (ast.Expr, string, bool) {
	if len(catchTypes) != 1 {
		return nil, "", false
	}
	descriptor, source := sourceCatchDescriptor(catchTypes[0], ctx)
	if !source {
		return nil, "", false
	}
	physical := javaTypeStringToGoTypeExpr(catchTypes[0], inScopeTypeParameters(ctx), ctx)
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{ast.NewIdent(recoveredName), descriptor}), symbol.NodeToStr(physical), true
}
