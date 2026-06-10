package transpiler

import "go/ast"

// This file registers the java.util.stream intrinsics, mapping Stream pipelines
// onto the eager slice-backed stdjava.Stream[T] runtime type (stdjava/stream.go).
//
// Operations that change the element type (map) or take extra arguments that a Go
// method cannot express as a type parameter (map/reduce/sorted) are emitted as
// stdjava free functions; the rest are methods on the Stream value. The lambda
// arguments are re-typed from the receiver's element type by the shared
// element-typed-lambda machinery in intrinsics.go (map/filter/forEach/... are in
// elementTypedLambdaMethods).

func init() {
	registerStreamIntrinsics()
}

// streamTypeNames are the Java types a Stream-valued receiver may carry.
var streamTypeNames = []string{"Stream", "IntStream", "LongStream", "DoubleStream"}

func registerStreamIntrinsics() {
	// Collection.stream() -> stdjava.StreamOfSlice(coll.Slice())
	for _, t := range append(append([]string{}, listTypeNames...), setTypeNames...) {
		registerInstanceIntrinsic(t, "stream", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return stdjavaCall(ctx, "StreamOfSlice", methodCall(recv, "Slice"))
		})
	}

	// Stream.of(...) -> stdjava.NewStream(...)
	registerStaticIntrinsic("Stream", "of", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "NewStream", args...)
	})

	for _, t := range streamTypeNames {
		// Method-form terminal/intermediate operations that keep the element type.
		registerInstanceIntrinsic(t, "filter", streamMethod("Filter", 1))
		registerInstanceIntrinsic(t, "forEach", streamMethod("ForEach", 1))
		registerInstanceIntrinsic(t, "count", streamMethod("Count", 0))
		registerInstanceIntrinsic(t, "anyMatch", streamMethod("AnyMatch", 1))
		registerInstanceIntrinsic(t, "allMatch", streamMethod("AllMatch", 1))
		registerInstanceIntrinsic(t, "noneMatch", streamMethod("NoneMatch", 1))
		registerInstanceIntrinsic(t, "limit", streamMethod("Limit", 1))

		// map changes element type, so it is the free function StreamMap.
		registerInstanceIntrinsic(t, "map", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, "StreamMap", recv, args[0])
		})
		// sorted() with natural ordering needs a constraints.Ordered type param,
		// which a Go method cannot carry, so it is the free function StreamSorted.
		registerInstanceIntrinsic(t, "sorted", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return stdjavaCall(ctx, "StreamSorted", recv)
		})
		// reduce(identity, accumulator) -> stdjava.StreamReduce(s, identity, acc).
		registerInstanceIntrinsic(t, "reduce", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 2) {
				return nil
			}
			return stdjavaCall(ctx, "StreamReduce", recv, args[0], args[1])
		})
		// collect(Collectors.toList()) -> s.ToList(). Only the toList collector is
		// recognized; other collectors fall through.
		registerInstanceIntrinsic(t, "collect", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			if isCollectorsToList(args[0]) {
				return methodCall(recv, "ToList")
			}
			return nil
		})
		// toList() (Java 16+) -> s.ToList().
		registerInstanceIntrinsic(t, "toList", streamMethod("ToList", 0))
	}
}

// streamMethod builds a generator that emits a method call on the stream value
// when the argument count matches.
func streamMethod(goName string, argc int) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, argc) {
			return nil
		}
		return methodCall(recv, goName, args...)
	}
}

// isCollectorsToList reports whether the parsed expression is a call to
// Collectors.toList() (its argument carries no useful runtime value for the
// slice-backed stream, so it is replaced by ToList()).
func isCollectorsToList(arg ast.Expr) bool {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	if base, ok := sel.X.(*ast.Ident); ok && base.Name == "Collectors" {
		return sel.Sel.Name == "toList" || sel.Sel.Name == "toUnmodifiableList"
	}
	return false
}
