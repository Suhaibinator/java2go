package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

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
	// Collection.stream() / parallelStream() -> stdjava.StreamOfSlice(coll.Slice()).
	// Parallel streams run sequentially here, which keeps every ordering
	// guarantee Java makes and only forgoes the concurrency.
	for _, t := range append(append([]string{}, listTypeNames...), setTypeNames...) {
		for _, method := range []string{"stream", "parallelStream"} {
			registerInstanceIntrinsic(t, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if !expectArgs(args, 0) {
					return nil
				}
				return stdjavaCall(ctx, "StreamOfSlice", methodCall(recv, "Slice"))
			})
		}
	}

	// Stream.of(...) uses Java's inferred reference type explicitly. Go cannot
	// infer Number from heterogeneous Integer and Long wrapper pointers.
	registerStaticIntrinsic("Stream", "of", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(ctx.intrinsicTypeArgs) == 1 {
			return stdjavaGenericCall(ctx, "NewStream", ctx.intrinsicTypeArgs, args)
		}
		return stdjavaCall(ctx, "NewStream", args...)
	})
	registerStaticIntrinsicTypeArgs("Stream", "of", func(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
		return []ast.Expr{javaTypeStringToGoTypeExpr(intrinsicFactoryElementJavaType(invocation, ctx, source), inScopeTypeParameters(ctx), ctx)}
	})
	registerStaticIntrinsicDerivedResultType("Stream", "of", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		return "Stream<" + intrinsicFactoryElementJavaType(invocation, ctx, source) + ">", true
	})

	for _, t := range streamTypeNames {
		// Lambda argument shapes, declared alongside the intrinsics they belong to.
		// map's result type is free (Function<T,R>) and is recovered from the
		// mapper's body; reduce's is pinned to the element type (BinaryOperator<T>).
		registerLambdaShape(t, "filter", lambdaResultBool)
		registerLambdaShape(t, "anyMatch", lambdaResultBool)
		registerLambdaShape(t, "allMatch", lambdaResultBool)
		registerLambdaShape(t, "noneMatch", lambdaResultBool)
		registerLambdaShape(t, "forEach", lambdaResultVoid)
		registerLambdaShape(t, "map", lambdaResultInferred)
		registerLambdaShape(t, "reduce", lambdaResultElement)
		registerLambdaShape(t, "peek", lambdaResultVoid)
		// flatMap's mapper returns a Stream whose element type is free.
		registerLambdaShape(t, "flatMap", lambdaResultInferred)
		// sorted/min/max take a Comparator, whose closure returns Java int.
		registerLambdaShape(t, "sorted", lambdaResultInt32)
		registerLambdaShape(t, "min", lambdaResultInt32)
		registerLambdaShape(t, "max", lambdaResultInt32)

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
		// sorted() needs a type parameter that a Go method cannot carry, so both
		// its natural-ordering and comparator forms are free functions.
		registerInstanceIntrinsic(t, "sorted", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 0:
				return stdjavaCall(ctx, "StreamSorted", recv, executionExpr(ctx))
			case 1:
				return stdjavaCall(ctx, "StreamSortedWith", recv, args[0], executionExpr(ctx))
			}
			return nil
		})
		// reduce has three arities. The first two keep the element type:
		// reduce(accumulator) returns an Optional, reduce(identity, accumulator) a
		// plain value.
		//
		// The three-argument form is `<U> U reduce(U, BiFunction<U,T,U>,
		// BinaryOperator<U>)`, so its accumulator takes (U, T) while its combiner
		// takes (U, U), and U is unrelated to the element type. The element-typed
		// shape table cannot express that — it gives every parameter of every
		// lambda the one element type — so it uses a per-argument typer instead,
		// with U read from the identity argument.
		registerLambdaArgumentTyper(t, "reduce", reduceLambdaArgumentTypes)
		registerInstanceIntrinsic(t, "reduce", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 1:
				return stdjavaCall(ctx, "StreamReduceOptional", recv, args[0])
			case 2:
				return stdjavaCall(ctx, "StreamReduce", recv, args[0], args[1])
			case 3:
				return stdjavaCall(ctx, "StreamReduceCombining", recv, args[0], args[1], args[2])
			}
			return nil
		})
		// flatMap changes the element type, so it is the free function StreamFlatMap.
		registerInstanceIntrinsic(t, "flatMap", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, "StreamFlatMap", recv, args[0])
		})
		// distinct uses Java equality/hashCode for any reference representation.
		registerInstanceIntrinsic(t, "distinct", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return stdjavaCall(ctx, "StreamDistinct", recv, intrinsicExecutionExpr(ctx))
		})

		registerInstanceIntrinsic(t, "skip", streamMethod("Skip", 1))
		registerInstanceIntrinsic(t, "peek", streamMethod("Peek", 1))
		registerInstanceIntrinsic(t, "findFirst", streamMethod("FindFirst", 0))
		registerInstanceIntrinsic(t, "findAny", streamMethod("FindAny", 0))
		registerInstanceIntrinsic(t, "sequential", streamMethod("Sequential", 0))
		registerInstanceIntrinsic(t, "parallel", streamMethod("Parallel", 0))
		registerInstanceIntrinsic(t, "unordered", streamMethod("Unordered", 0))

		// min/max return an Optional and take an optional comparator.
		registerInstanceIntrinsic(t, "min", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 0:
				return stdjavaCall(ctx, "StreamMin", recv, executionExpr(ctx))
			case 1:
				return stdjavaCall(ctx, "StreamMinWith", recv, args[0], executionExpr(ctx))
			}
			return nil
		})
		registerInstanceIntrinsic(t, "max", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 0:
				return stdjavaCall(ctx, "StreamMax", recv, executionExpr(ctx))
			case 1:
				return stdjavaCall(ctx, "StreamMaxWith", recv, args[0], executionExpr(ctx))
			}
			return nil
		})
		// toList() (Java 16+) -> s.ToList().
		registerInstanceIntrinsic(t, "toList", streamMethod("ToList", 0))
	}
}

// reduceLambdaArgumentTypes types the lambdas of the three-argument
// Stream.reduce. It applies to that arity only; the other two are element-typed
// and go through the ordinary shape table.
func reduceLambdaArgumentTypes(invocation *sitter.Node, ctx Ctx, source []byte) map[int]lambdaArgumentTypes {
	if invocationArgumentCount(invocation) != 3 {
		return nil
	}
	identity := invocationArgumentNode(invocation, 0)
	resultType, ok := inferExprJavaType(identity, ctx, source)
	if !ok {
		return nil
	}
	if boxed := ternaryBoxedJavaType(resultType); boxed != "" {
		resultType = "java.lang." + boxed
	}
	elementTypes := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
	if len(elementTypes) != 1 {
		return nil
	}
	return map[int]lambdaArgumentTypes{
		1: {paramJavaTypes: []string{resultType, elementTypes[0]}, resultJavaType: resultType},
		2: {paramJavaTypes: []string{resultType, resultType}, resultJavaType: resultType},
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
