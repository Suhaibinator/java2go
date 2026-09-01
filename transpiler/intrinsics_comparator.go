package transpiler

import (
	"go/ast"
	"strings"
)

// This file registers the java.util.Comparator intrinsics and the
// comparator-taking overloads of the Collections and Arrays sort utilities,
// mapping them onto the stdjava.Comparator[T] runtime type (stdjava/comparator.go).
//
// A Java `Comparator<T>` lambda lowers to a `func(a, b T) int32`, which Go
// assigns to stdjava.Comparator[T] without a conversion, so a comparator
// argument needs no wrapping — only the right parameter and result types, which
// come from the lambda-shape machinery in intrinsics.go.
//
// Known limitation: `Comparator.comparing(p -> p.getName())` cannot type its
// lambda parameter. The key extractor's parameter type comes from the target
// type of the whole expression (the variable being assigned, or the enclosing
// call's element type), which the transpiler does not propagate inward. The
// method-reference form and an explicitly typed lambda
// (`Comparator.comparing((Person p) -> p.getName())`) both work today.

func init() {
	registerComparatorIntrinsics()
	registerComparatorSortIntrinsics()
}

// comparatorTypeNames are the Java types a Comparator-valued receiver may carry.
var comparatorTypeNames = []string{"Comparator"}

// namedFunctionalInterfaceTypeExpr returns the Go runtime type for a declared
// built-in functional interface (e.g. stdjava.Comparator[int32] for
// Comparator<Integer>), or nil when javaType is not one, is used raw, or is
// shadowed by a user-defined class of the same name.
func namedFunctionalInterfaceTypeExpr(javaType string, ctx Ctx) ast.Expr {
	base, typeArgs := parseJavaTypeString(strings.TrimSpace(javaType))
	name := stripJavaQualifier(base)
	if _, ok := builtinFunctionalInterfaces[name]; !ok {
		return nil
	}
	if len(typeArgs) == 0 || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return nil
	}
	return collectionTypeExpr(name, typeArgs, inScopeTypeParameters(ctx), ctx)
}

func registerComparatorIntrinsics() {
	// Comparator.naturalOrder() / reverseOrder(). The Go type parameter is
	// inferred from the context the comparator flows into.
	registerStaticIntrinsic("Comparator", "naturalOrder", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "NaturalOrder")
	})
	registerStaticIntrinsic("Comparator", "reverseOrder", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "ReverseOrder")
	})
	// Collections.reverseOrder() is the same comparator under another name.
	registerStaticIntrinsic("Collections", "reverseOrder", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "ReverseOrder")
	})

	// Comparator.comparing / comparingInt / comparingLong / comparingDouble all
	// build a comparator from a sort-key extractor.
	for _, name := range []string{"comparing", "comparingInt", "comparingLong", "comparingDouble"} {
		registerStaticIntrinsic("Comparator", name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, "ComparatorComparing", args[0])
		})
	}

	for _, t := range comparatorTypeNames {
		// compare(a, b) is a direct call on the comparator func value.
		registerInstanceIntrinsic(t, "compare", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 2) {
				return nil
			}
			return methodCall(recv, "Compare", args...)
		})
		registerInstanceIntrinsic(t, "reversed", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return methodCall(recv, "Reversed")
		})
		// thenComparing takes either another Comparator or a key extractor. The
		// two are distinguished by the argument's shape: a lambda with two
		// parameters is a comparator, one with a single parameter is an extractor.
		registerInstanceIntrinsic(t, "thenComparing", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			if isKeyExtractorExpr(args[0]) {
				return stdjavaCall(ctx, "ComparatorThenComparingKey", recv, args[0])
			}
			return methodCall(recv, "ThenComparing", args[0])
		})
	}
}

// isKeyExtractorExpr reports whether a thenComparing argument is a single-
// parameter key extractor rather than a two-parameter comparator. A non-lambda
// argument (a comparator variable or a method reference) is treated as a
// comparator, which is the commoner form.
func isKeyExtractorExpr(arg ast.Expr) bool {
	funcLit, ok := arg.(*ast.FuncLit)
	if !ok || funcLit.Type == nil || funcLit.Type.Params == nil {
		return false
	}
	parameters := 0
	for _, field := range funcLit.Type.Params.List {
		if len(field.Names) == 0 {
			parameters++
			continue
		}
		parameters += len(field.Names)
	}
	return parameters == 1
}

func registerComparatorSortIntrinsics() {
	// Collections.sort(list, comparator) / max(coll, cmp) / min(coll, cmp), and
	// Arrays.sort(array, cmp). Each takes its comparator's element type from the
	// collection argument, since a static call has no receiver to read it from.
	registerStaticLambdaShape("Collections", "sort", 0, []int{1}, lambdaResultInt32)
	registerStaticLambdaShape("Collections", "max", 0, []int{1}, lambdaResultInt32)
	registerStaticLambdaShape("Collections", "min", 0, []int{1}, lambdaResultInt32)
	registerStaticLambdaShape("Arrays", "sort", 0, []int{1}, lambdaResultInt32)

	// list.sort(comparator) reads its element type from the receiver.
	for _, t := range listTypeNames {
		registerLambdaShape(t, "sort", lambdaResultInt32)
		registerInstanceIntrinsic(t, "sort", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, "SortWith", recv, args[0])
		})
	}
}
