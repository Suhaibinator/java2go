package transpiler

import (
	"go/ast"
	"go/token"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file lowers `stream.collect(Collectors.X(...))` onto the collector
// runtime in stdjava/collectors.go.
//
// A Collector is not modelled as a runtime value. Java's Collector<T, A, R>
// carries three type parameters that a single Go type cannot hold, so the call
// is rewritten in place: the collector's identity decides which runtime
// function the stream is passed to. That means `collect` needs its own call
// node rather than just its parsed arguments, which is what the node-aware
// intrinsic registry provides.
//
// A collector in a *downstream* position — groupingBy's second argument, or
// mapping's — lowers instead to a `func(Stream[T]) R` applied to each group's
// own stream, so the forms compose to any depth.
//
// The collector's lambdas cannot be typed by the element-typed shape machinery:
// toMap's key extractor, value extractor and merge function have three
// different signatures. They are parsed here with their own parameter types
// seeded, then given explicit result types.

func init() {
	for _, t := range streamTypeNames {
		registerInstanceNodeIntrinsic(t, "collect", lowerCollectCall)
	}
}

// downstreamGroupParameter is the name given to the stream parameter of a
// lowered downstream collector.
const downstreamGroupParameter = "__java2goGroup"

// lowerCollectCall rewrites `stream.collect(Collectors.X(...))`.
func lowerCollectCall(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if invocationArgumentCount(invocation) != 1 {
		return nil
	}
	objectNode := invocation.ChildByFieldName("object")
	elementTypes := receiverElementJavaTypes(objectNode, ctx, source)
	if len(elementTypes) != 1 {
		return nil
	}
	collected, _ := lowerCollector(invocationArgumentNode(invocation, 0), recv, elementTypes[0], ctx, source)
	return collected
}

// lowerCollector rewrites one Collectors factory call applied to streamExpr,
// whose elements are elementJavaType. It returns the expression and the Java
// type it produces, or nil when the collector is not recognized.
func lowerCollector(collector *sitter.Node, streamExpr ast.Expr, elementJavaType string, ctx Ctx, source []byte) (ast.Expr, string) {
	name, ok := collectorFactoryName(collector, source)
	if !ok {
		return nil, ""
	}
	arity := invocationArgumentCount(collector)

	switch name {
	case "toList", "toUnmodifiableList":
		if arity != 0 {
			return nil, ""
		}
		return methodCall(streamExpr, "ToList"), "List<" + elementJavaType + ">"

	case "toSet", "toUnmodifiableSet":
		if arity != 0 {
			return nil, ""
		}
		return stdjavaCall(ctx, "StreamToSet", streamExpr), "Set<" + elementJavaType + ">"

	case "counting":
		if arity != 0 {
			return nil, ""
		}
		return stdjavaCall(ctx, "StreamCounting", streamExpr), "long"

	case "joining":
		// joining() / joining(sep) / joining(sep, prefix, suffix); the runtime
		// takes all three, so the shorter forms pass empty strings.
		separator, prefix, suffix := emptyStringLit(), emptyStringLit(), emptyStringLit()
		switch arity {
		case 0:
		case 1:
			separator = ParseExpr(invocationArgumentNode(collector, 0), source, ctx)
		case 3:
			separator = ParseExpr(invocationArgumentNode(collector, 0), source, ctx)
			prefix = ParseExpr(invocationArgumentNode(collector, 1), source, ctx)
			suffix = ParseExpr(invocationArgumentNode(collector, 2), source, ctx)
		default:
			return nil, ""
		}
		return stdjavaCall(ctx, "StreamJoining", streamExpr, separator, prefix, suffix), "String"

	case "summingInt", "summingLong", "summingDouble":
		if arity != 1 {
			return nil, ""
		}
		resultType := numericCollectorJavaType(name)
		value := parseCollectorLambda(collector, 0, []string{elementJavaType}, resultType, ctx, source)
		return stdjavaCall(ctx, "StreamSummingOf", streamExpr, value), resultType

	case "averagingInt", "averagingLong", "averagingDouble":
		if arity != 1 {
			return nil, ""
		}
		value := parseCollectorLambda(collector, 0, []string{elementJavaType}, numericCollectorJavaType(name), ctx, source)
		return stdjavaCall(ctx, "StreamAveragingOf", streamExpr, value), "double"

	case "toMap":
		if arity != 2 && arity != 3 {
			return nil, ""
		}
		keyType := collectorLambdaResultJavaType(collector, 0, elementJavaType, ctx, source)
		valueType := collectorLambdaResultJavaType(collector, 1, elementJavaType, ctx, source)
		if keyType == "" || valueType == "" {
			return nil, ""
		}
		key := parseCollectorLambda(collector, 0, []string{elementJavaType}, keyType, ctx, source)
		value := parseCollectorLambda(collector, 1, []string{elementJavaType}, valueType, ctx, source)
		if arity == 2 {
			return stdjavaCall(ctx, "StreamToMap", streamExpr, key, value), "Map<" + keyType + "," + valueType + ">"
		}
		// The merge function resolves duplicate keys: (V, V) -> V.
		merge := parseCollectorLambda(collector, 2, []string{valueType, valueType}, valueType, ctx, source)
		return stdjavaCall(ctx, "StreamToMapMerging", streamExpr, key, value, merge),
			"Map<" + keyType + "," + valueType + ">"

	case "groupingBy":
		if arity != 1 && arity != 2 {
			return nil, ""
		}
		keyType := collectorLambdaResultJavaType(collector, 0, elementJavaType, ctx, source)
		if keyType == "" {
			return nil, ""
		}
		classifier := parseCollectorLambda(collector, 0, []string{elementJavaType}, keyType, ctx, source)
		if arity == 1 {
			return stdjavaCall(ctx, "StreamGroupingBy", streamExpr, classifier),
				"Map<" + keyType + ",List<" + elementJavaType + ">>"
		}
		downstream, downstreamType := lowerDownstreamCollector(collector, 1, elementJavaType, ctx, source)
		if downstream == nil {
			return nil, ""
		}
		return stdjavaCall(ctx, "StreamGroupingByDownstream", streamExpr, classifier, downstream),
			"Map<" + keyType + "," + downstreamType + ">"

	case "partitioningBy":
		if arity != 1 && arity != 2 {
			return nil, ""
		}
		predicate := parseCollectorLambda(collector, 0, []string{elementJavaType}, "boolean", ctx, source)
		if arity == 1 {
			return stdjavaCall(ctx, "StreamPartitioningBy", streamExpr, predicate),
				"Map<Boolean,List<" + elementJavaType + ">>"
		}
		downstream, downstreamType := lowerDownstreamCollector(collector, 1, elementJavaType, ctx, source)
		if downstream == nil {
			return nil, ""
		}
		return stdjavaCall(ctx, "StreamPartitioningByDownstream", streamExpr, predicate, downstream),
			"Map<Boolean," + downstreamType + ">"

	case "mapping":
		// mapping(mapper, downstream) maps each element before collecting, so it
		// only appears in a downstream position; lowerDownstreamCollector handles
		// it there. Applied directly to a stream it is still well defined.
		if arity != 2 {
			return nil, ""
		}
		mappedType := collectorLambdaResultJavaType(collector, 0, elementJavaType, ctx, source)
		if mappedType == "" {
			return nil, ""
		}
		mapper := parseCollectorLambda(collector, 0, []string{elementJavaType}, mappedType, ctx, source)
		mapped := stdjavaCall(ctx, "StreamMap", streamExpr, mapper)
		return lowerCollector(invocationArgumentNode(collector, 1), mapped, mappedType, ctx, source)
	}
	return nil, ""
}

// lowerDownstreamCollector lowers a collector in a downstream position into a
// `func(group Stream[T]) R` that applies it to one group's stream.
func lowerDownstreamCollector(collector *sitter.Node, argIndex int, elementJavaType string, ctx Ctx, source []byte) (ast.Expr, string) {
	group := &ast.Ident{Name: downstreamGroupParameter}
	applied, resultType := lowerCollector(invocationArgumentNode(collector, argIndex), group, elementJavaType, ctx, source)
	if applied == nil {
		return nil, ""
	}
	streamType := applyTypeArguments(stdjavaQualifiedExpr("Stream", ctx),
		[]ast.Expr{javaTypeStringToGoTypeExpr(elementJavaType, inScopeTypeParameters(ctx), ctx)})
	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{group},
				Type:  streamType,
			}}},
			Results: &ast.FieldList{List: []*ast.Field{{
				Type: javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx),
			}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{applied}}}},
	}, resultType
}

// collectorFactoryName returns the Collectors method a node calls, e.g.
// "groupingBy" for `Collectors.groupingBy(...)`.
func collectorFactoryName(collector *sitter.Node, source []byte) (string, bool) {
	if collector == nil || collector.Type() != "method_invocation" {
		return "", false
	}
	object := collector.ChildByFieldName("object")
	name := collector.ChildByFieldName("name")
	if object == nil || name == nil {
		return "", false
	}
	if stripJavaQualifier(object.Content(source)) != "Collectors" {
		return "", false
	}
	return name.Content(source), true
}

// parseCollectorLambda parses one of a collector's lambda arguments with its
// parameter types seeded, then applies the declared result type.
func parseCollectorLambda(collector *sitter.Node, argIndex int, paramJavaTypes []string, resultJavaType string, ctx Ctx, source []byte) ast.Expr {
	argNode := invocationArgumentNode(collector, argIndex)
	if argNode == nil {
		return nil
	}
	argCtx := ctx.Clone()
	argCtx.lambdaParameterJavaTypes = append([]string(nil), paramJavaTypes...)
	parsed := ParseExpr(argNode, source, argCtx)

	typeParams := inScopeTypeParameters(ctx)
	paramTypes := make([]ast.Expr, 0, len(paramJavaTypes))
	for _, javaType := range paramJavaTypes {
		paramTypes = append(paramTypes, javaTypeStringToGoTypeExpr(javaType, typeParams, ctx))
	}
	var resultType ast.Expr
	if resultJavaType != "" {
		resultType = javaTypeStringToGoTypeExpr(resultJavaType, typeParams, ctx)
	}
	return retypeLambdaWithTypes(parsed, paramTypes, resultType)
}

// collectorLambdaResultJavaType infers what a collector's lambda argument
// evaluates to, given the element type its parameter takes.
func collectorLambdaResultJavaType(collector *sitter.Node, argIndex int, elementJavaType string, ctx Ctx, source []byte) string {
	resultType, ok := inferLambdaResultJavaType(
		invocationArgumentNode(collector, argIndex), []string{elementJavaType}, ctx, source)
	if !ok {
		return ""
	}
	return resultType
}

// numericCollectorJavaType maps a summing/averaging collector to the Java type
// its extractor returns.
func numericCollectorJavaType(name string) string {
	switch {
	case len(name) > 4 && name[len(name)-4:] == "Long":
		return "long"
	case len(name) > 6 && name[len(name)-6:] == "Double":
		return "double"
	}
	return "int"
}

func emptyStringLit() ast.Expr {
	return &ast.BasicLit{Kind: token.STRING, Value: `""`}
}
