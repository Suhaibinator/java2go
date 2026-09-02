package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file registers the primitive-stream surface of java.util.stream: the
// IntStream/LongStream/DoubleStream sources, the conversions between them, and
// their numeric terminal operations.
//
// The three primitive streams are modelled as Stream[int32], Stream[int64] and
// Stream[float64] (see stdjava/stream_numeric.go), so every operation already
// registered for Stream applies to them unchanged and only the numeric-specific
// entries live here.

func init() {
	registerNumericStreamIntrinsics()
}

func registerNumericStreamIntrinsics() {
	registerNumericStreamSources()
	registerNumericStreamConversions()
	registerNumericStreamTerminals()
}

func registerNumericStreamSources() {
	// IntStream.range / rangeClosed, and the long forms.
	registerStaticIntrinsic("IntStream", "range", numericRange("IntStreamRange"))
	registerStaticIntrinsic("IntStream", "rangeClosed", numericRange("IntStreamRangeClosed"))
	registerStaticIntrinsic("LongStream", "range", numericRange("LongStreamRange"))
	registerStaticIntrinsic("LongStream", "rangeClosed", numericRange("LongStreamRangeClosed"))
	registerStaticIntrinsicResultType("IntStream", "range", "IntStream")
	registerStaticIntrinsicResultType("IntStream", "rangeClosed", "IntStream")
	registerStaticIntrinsicResultType("LongStream", "range", "LongStream")
	registerStaticIntrinsicResultType("LongStream", "rangeClosed", "LongStream")

	// IntStream.of(...) and friends share Stream.of's runtime constructor.
	for _, streamType := range []string{"IntStream", "LongStream", "DoubleStream"} {
		// The element type is spelled explicitly: IntStream.of(3, 1, 2) passes
		// untyped constants, which Go would otherwise infer as a host-sized int
		// rather than Java's 32-bit int.
		registerStaticIntrinsic(streamType, "of", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(ctx.intrinsicTypeArgs) != 1 {
				return nil
			}
			return stdjavaGenericCall(ctx, "NewStream", ctx.intrinsicTypeArgs, args)
		})
		registerStaticIntrinsicTypeArgs(streamType, "of", primitiveStreamEmptyTypeArgs(streamType))
		registerStaticIntrinsicResultType(streamType, "of", streamType)
		registerStaticIntrinsic(streamType, "empty", emptyStream)
		registerStaticIntrinsicTypeArgs(streamType, "empty", primitiveStreamEmptyTypeArgs(streamType))
		registerStaticIntrinsicResultType(streamType, "empty", streamType)
	}

	// Stream.empty() has no argument to infer its element type from, so it is
	// taken from the type the expression is assigned to.
	registerStaticIntrinsic("Stream", "empty", emptyStream)
	registerStaticIntrinsicTypeArgs("Stream", "empty", expectedElementTypeArgs)

	// Stream.concat(a, b) keeps the element type of its arguments.
	registerStaticIntrinsic("Stream", "concat", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "StreamConcat", args[0], args[1])
	})
	registerStaticIntrinsicDerivedResultType("Stream", "concat", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		argNode := invocationArgumentNode(invocation, 0)
		if argNode == nil {
			return "", false
		}
		return inferExprJavaType(argNode, ctx, source)
	})

	// Arrays.stream(array). A reference array erases its elements to `any` at
	// runtime, so the component type is passed explicitly from the call site.
	registerStaticIntrinsic("Arrays", "stream", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaGenericCall(ctx, "StreamOfArray", ctx.intrinsicTypeArgs, args)
	})
	registerStaticIntrinsicTypeArgs("Arrays", "stream", arrayComponentTypeArgs)
	registerStaticIntrinsicDerivedResultType("Arrays", "stream", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		component, ok := arrayComponentJavaType(invocation, ctx, source)
		if !ok {
			return "", false
		}
		return "Stream<" + component + ">", true
	})
}

func registerNumericStreamConversions() {
	for _, t := range streamTypeNames {
		// mapTo* pins the mapper's result to a primitive; mapToObj leaves it free.
		registerLambdaShape(t, "mapToInt", lambdaResultInt32)
		registerLambdaShape(t, "mapToLong", lambdaResultInt64)
		registerLambdaShape(t, "mapToDouble", lambdaResultFloat64)
		registerLambdaShape(t, "mapToObj", lambdaResultInferred)
		for _, method := range []string{"mapToInt", "mapToLong", "mapToDouble", "mapToObj"} {
			registerInstanceIntrinsic(t, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if !expectArgs(args, 1) {
					return nil
				}
				return stdjavaCall(ctx, "StreamMap", recv, args[0])
			})
		}
		registerInstanceIntrinsicResultType(t, "mapToInt", "IntStream")
		registerInstanceIntrinsicResultType(t, "mapToLong", "LongStream")
		registerInstanceIntrinsicResultType(t, "mapToDouble", "DoubleStream")

		// boxed is an identity: a primitive stream is already a Stream of the
		// boxed type's Go counterpart.
		registerInstanceIntrinsic(t, "boxed", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return stdjavaCall(ctx, "StreamBoxed", recv)
		})
		registerInstanceIntrinsic(t, "asLongStream", numericConversion("StreamAsLongStream"))
		registerInstanceIntrinsic(t, "asDoubleStream", numericConversion("StreamAsDoubleStream"))
		registerInstanceIntrinsicResultType(t, "asLongStream", "LongStream")
		registerInstanceIntrinsicResultType(t, "asDoubleStream", "DoubleStream")
	}
}

// summaryStatisticsTypeNames are the Java classes an IntStream-family
// summaryStatistics() result may be declared as.
var summaryStatisticsTypeNames = []string{
	"IntSummaryStatistics",
	"LongSummaryStatistics",
	"DoubleSummaryStatistics",
}

func registerNumericStreamTerminals() {
	for _, t := range streamTypeNames {
		registerInstanceIntrinsic(t, "sum", numericTerminal("StreamSum"))
		registerInstanceIntrinsic(t, "average", numericTerminal("StreamAverage"))
		registerInstanceIntrinsic(t, "summaryStatistics", numericTerminal("StreamSummaryStatistics"))
	}
	registerInstanceIntrinsicResultType("IntStream", "sum", "int")
	registerInstanceIntrinsicResultType("LongStream", "sum", "long")
	registerInstanceIntrinsicResultType("DoubleStream", "sum", "double")
	for _, t := range streamTypeNames {
		registerInstanceIntrinsicResultType(t, "average", "Optional<Double>")
	}

	// Without a result type a chained or var-typed summaryStatistics call cannot
	// resolve its accessors.
	registerInstanceIntrinsicResultType("IntStream", "summaryStatistics", "IntSummaryStatistics")
	registerInstanceIntrinsicResultType("LongStream", "summaryStatistics", "LongSummaryStatistics")
	registerInstanceIntrinsicResultType("DoubleStream", "summaryStatistics", "DoubleSummaryStatistics")
	registerInstanceIntrinsicResultType("Stream", "summaryStatistics", "IntSummaryStatistics")

	// The summary-statistics accessors.
	for method, goName := range map[string]string{
		"getCount":   "GetCount",
		"getSum":     "GetSum",
		"getMin":     "GetMin",
		"getMax":     "GetMax",
		"getAverage": "GetAverage",
	} {
		for _, statsType := range summaryStatisticsTypeNames {
			registerInstanceIntrinsic(statsType, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if !expectArgs(args, 0) {
					return nil
				}
				return methodCall(recv, goName)
			})
		}
	}
	// getCount and getAverage have the same type for all three; getSum, getMin and
	// getMax widen with the element type, as Java's do.
	for _, statsType := range summaryStatisticsTypeNames {
		registerInstanceIntrinsicResultType(statsType, "getCount", "long")
		registerInstanceIntrinsicResultType(statsType, "getAverage", "double")
	}
	for statsType, elementType := range map[string]string{
		"IntSummaryStatistics":    "int",
		"LongSummaryStatistics":   "long",
		"DoubleSummaryStatistics": "double",
	} {
		sumType := "long"
		if statsType == "DoubleSummaryStatistics" {
			sumType = "double"
		}
		registerInstanceIntrinsicResultType(statsType, "getSum", sumType)
		registerInstanceIntrinsicResultType(statsType, "getMin", elementType)
		registerInstanceIntrinsicResultType(statsType, "getMax", elementType)
	}
}

// numericRange builds a two-argument range generator.
func numericRange(runtimeName string) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, runtimeName, args[0], args[1])
	}
}

func numericConversion(runtimeName string) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, runtimeName, recv)
	}
}

func numericTerminal(runtimeName string) intrinsicGenerator {
	return func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, runtimeName, recv)
	}
}

// emptyStream generates stdjava.StreamEmpty[T](), with T supplied by the
// registered type-argument deriver.
func emptyStream(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
	if !expectArgs(args, 0) || len(ctx.intrinsicTypeArgs) != 1 {
		return nil
	}
	return stdjavaGenericCall(ctx, "StreamEmpty", ctx.intrinsicTypeArgs, nil)
}

// primitiveStreamEmptyTypeArgs supplies the fixed element type of a primitive
// stream's empty() factory.
func primitiveStreamEmptyTypeArgs(streamType string) typeArgDeriver {
	return func(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
		element, ok := primitiveStreamElementJavaTypes[streamType]
		if !ok {
			return nil
		}
		return []ast.Expr{javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)}
	}
}

// expectedElementTypeArgs reads Stream.empty()'s element type off the type the
// expression is being assigned to, which is the only place it appears.
func expectedElementTypeArgs(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	base, typeArgs := parseJavaTypeString(ctx.expectedType)
	if stripJavaQualifier(base) != "Stream" || len(typeArgs) != 1 {
		return nil
	}
	return []ast.Expr{javaTypeStringToGoTypeExpr(typeArgs[0], inScopeTypeParameters(ctx), ctx)}
}

// arrayComponentJavaType returns the Java component type of an Arrays.stream
// argument.
func arrayComponentJavaType(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	argNode := invocationArgumentNode(invocation, 0)
	if argNode == nil {
		return "", false
	}
	javaType, ok := inferExprJavaType(argNode, ctx, source)
	if !ok {
		return "", false
	}
	components := javaContainerElementTypes(javaType)
	if len(components) != 1 {
		return "", false
	}
	return components[0], true
}

func arrayComponentTypeArgs(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	component, ok := arrayComponentJavaType(invocation, ctx, source)
	if !ok {
		return nil
	}
	return []ast.Expr{javaTypeStringToGoTypeExpr(component, inScopeTypeParameters(ctx), ctx)}
}
