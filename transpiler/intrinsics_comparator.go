package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
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
// The argument-free factories and the key-extracting `comparing` family carry no
// element type of their own: it is fixed by the collection being sorted, one
// level up the tree. They therefore resolve it from the enclosing call (see
// elementFromEnclosingTarget in intrinsics.go) rather than from their own
// arguments, which is what lets `people.sort(Comparator.comparing(...))` and
// `Collections.sort(xs, Comparator.naturalOrder())` compile.
//
// A factory assigned to a local rather than passed straight to a sort has no
// enclosing call, so the element type is read from the declared variable type
// instead. It is only unresolvable when neither is present.

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
	registerStaticIntrinsic("Comparator", "naturalOrder", orderFactory("NaturalOrder"))
	registerStaticIntrinsic("Comparator", "reverseOrder", orderFactory("ReverseOrder"))
	// Collections.reverseOrder() is the same comparator under another name.
	registerStaticIntrinsic("Collections", "reverseOrder", orderFactory("ReverseOrder"))

	// Comparator.comparing / comparingInt / comparingLong / comparingDouble all
	// build a comparator from a sort-key extractor. The extractor's parameter
	// type is not visible from this call — it is fixed by the collection being
	// sorted — so it is taken from the enclosing call, and its result type is
	// whatever the extractor's body evaluates to.
	comparingResultKinds := map[string]lambdaResultKind{
		"comparing":       lambdaResultInferred,
		"comparingInt":    lambdaResultInt32,
		"comparingLong":   lambdaResultInt64,
		"comparingDouble": lambdaResultFloat64,
	}
	for name, resultKind := range comparingResultKinds {
		registerStaticIntrinsic("Comparator", name, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, "ComparatorComparing", args[0])
		})
		registerStaticLambdaShape("Comparator", name, elementFromEnclosingTarget, []int{0}, resultKind)
	}

	// naturalOrder / reverseOrder carry no argument to infer their element type
	// from, so it too comes from the enclosing call.
	for _, name := range []string{"naturalOrder", "reverseOrder"} {
		registerStaticIntrinsicTypeArgs("Comparator", name, enclosingTargetTypeArgs)
	}
	registerStaticIntrinsicTypeArgs("Collections", "reverseOrder", enclosingTargetTypeArgs)

	// Every Comparator-producing call reports a Comparator result, so a method
	// chained onto it resolves to the generated Go spelling instead of keeping
	// its Java one.
	for _, method := range []string{"comparing", "comparingInt", "comparingLong", "comparingDouble", "naturalOrder", "reverseOrder"} {
		registerStaticIntrinsicResultType("Comparator", method, "Comparator")
	}
	registerStaticIntrinsicResultType("Collections", "reverseOrder", "Comparator")
	for _, t := range comparatorTypeNames {
		registerInstanceIntrinsicResultType(t, "reversed", "Comparator")
		registerInstanceIntrinsicResultType(t, "thenComparing", "Comparator")
		registerInstanceIntrinsicResultType(t, "compare", "int")
	}

	for _, t := range comparatorTypeNames {
		// thenComparing's key-extractor overload takes one element-typed
		// parameter and returns a freely-typed sort key.
		registerLambdaShape(t, "thenComparing", lambdaResultInferred)

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

// orderFactory builds a generator for the argument-free comparator factories.
// Go cannot infer their element type from the call, so it is spelled out from
// the enclosing target; without it the emitted call does not compile.
func orderFactory(runtimeName string) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) || len(ctx.intrinsicTypeArgs) != 1 {
			return nil
		}
		return stdjavaGenericCall(ctx, runtimeName, ctx.intrinsicTypeArgs, nil)
	}
}

// enclosingTargetTypeArgs supplies the element type of the call an intrinsic is
// an argument to — or of the type it is assigned to — as an explicit Go type
// argument.
func enclosingTargetTypeArgs(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	elementTypes := targetElementJavaTypes(invocation, ctx, source)
	if len(elementTypes) != 1 {
		return nil
	}
	return []ast.Expr{javaTypeStringToGoTypeExpr(elementTypes[0], inScopeTypeParameters(ctx), ctx)}
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
