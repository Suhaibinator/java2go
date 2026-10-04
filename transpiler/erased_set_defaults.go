package transpiler

import "go/ast"

// Canonical Set references use the collection protocol. Concrete native Set
// allocations retain their own storage and logical generic type arguments.
func registerErasedSetIntrinsics() {
	for _, operation := range []struct {
		java, runtime, result string
		arity                 int
	}{
		{"add", "CollectionAddExecution", "boolean", 1},
		{"remove", "CollectionRemoveExecution", "boolean", 1},
		{"contains", "CollectionContainsExecution", "boolean", 1},
		{"size", "CollectionSizeExecution", "int", 0},
		{"isEmpty", "CollectionIsEmptyExecution", "boolean", 0},
		{"clear", "CollectionClearExecution", "void", 0},
		{"containsAll", "CollectionContainsAllExecution", "boolean", 1},
	} {
		for _, owner := range []string{"Set", "AbstractSet"} {
			registerInstanceIntrinsic(owner, operation.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != operation.arity {
					return nil
				}
				return stdjavaCall(ctx, operation.runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
			})
			registerInstanceIntrinsicResultType(owner, operation.java, operation.result)
		}
	}
}
