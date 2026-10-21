package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Java checkcast preserves all null references, including a typed nil behind
// Object. Go assertions reject nil interfaces, so normalize null before using
// the existing non-null assertion.
// A parameterized IIFE evaluates the operand exactly once without capturing
// caller identifiers. Nominal/subobject and array casts use their own helpers.
func nullableReferenceAssertion(value, target ast.Expr, targetOrigin inferredJavaTypeOrigin, ctx Ctx) ast.Expr {
	// The written target owns Java identity even if an earlier lowering erased
	// its physical Go representation. Bound syntax belongs to its declaration,
	// rather than the namespace of the cast's consumer or a shadowing binder.
	seen := map[*symbol.TypeParamDeclaration]bool{}
	var descriptor func(inferredJavaTypeOrigin, Ctx) (ast.Expr, bool)
	descriptor = func(origin inferredJavaTypeOrigin, declaring Ctx) (ast.Expr, bool) {
		if origin.parameter == nil {
			return declaredOriginDescriptor(origin, declaring, seen)
		}
		if origin.unresolved || seen[origin.parameter] {
			return nil, false
		}
		seen[origin.parameter] = true
		defer delete(seen, origin.parameter)
		for _, binding := range referenceTypeParameterBindings(declaring) {
			if binding.parameter.Declaration != origin.parameter {
				continue
			}
			result := stdjavaQualifiedExpr("ObjectTypeID", declaring)
			if len(binding.parameter.Bounds) > 0 {
				bound := declaredJavaTypeOrigin(binding.parameter.Bounds[0], binding.context)
				var ok bool
				result, ok = descriptor(bound, binding.context)
				if !ok {
					return nil, false
				}
			}
			for rank := 0; rank < origin.arrayRank; rank++ {
				result = stdjavaCall(ctx, "ArrayTypeID", result)
			}
			return result, true
		}
		return nil, false
	}
	if requested, ok := descriptor(targetOrigin, ctx); ok {
		return stdjavaGenericCall(ctx, "ReferenceCheckCast", []ast.Expr{target}, []ast.Expr{value, requested})
	}

	operand := ast.NewIdent("value")
	zero := zeroValueForType(target)
	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{operand}, Type: ast.NewIdent("any")}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: target}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.IfStmt{Cond: stdjavaCall(ctx, "JavaReferenceEqual", operand, ast.NewIdent("nil")), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{zero}}}}},
				&ast.ReturnStmt{Results: []ast.Expr{&ast.TypeAssertExpr{X: operand, Type: target}}},
			}},
		},
		Args: []ast.Expr{value},
	}
}
