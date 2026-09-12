package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func isExternalCallableType(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	return stripJavaQualifier(base) == "Callable" && resolveClassScopeByQualifiedName(ctx, base) == nil
}

// executorSubmitType resolves the Runnable/Callable overload before parsing the
// callback, so its return expressions receive Java boxing conversions.
func executorSubmitType(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	result := "Object"
	target, args := parseJavaTypeString(ctx.expectedType)
	if stripJavaQualifier(target) == "Future" && len(args) == 1 && expectedTypeTargetsExpression(ctx, invocation) && args[0] != "?" {
		result = args[0]
	}
	if invocationArgumentCount(invocation) == 2 {
		if result == "Object" {
			if actual, ok := inferExprJavaType(invocationArgumentNode(invocation, 1), ctx, source); ok && actual != "null" {
				result = intrinsicReferenceJavaType(actual)
			}
		}
		return result, false
	}
	task := invocationArgumentNode(invocation, 0)
	if task == nil {
		return result, false
	}
	if task.Type() == "lambda_expression" {
		body := task.ChildByFieldName("body")
		if body == nil {
			return result, false
		}
		if body.Type() == "block" {
			returned := callableReturnedExpression(body)
			if returned == nil {
				// A block which always throws is both void- and value-compatible;
				// javac selects Callable as the more specific overload.
				return result, body.NamedChildCount() > 0 && body.NamedChild(int(body.NamedChildCount())-1).Type() == "throw_statement"
			}
			body = returned
		}
		if inferred, ok := inferExprJavaType(body, ctx, source); ok {
			if inferred == "void" {
				return "Object", false
			}
			if result == "Object" && inferred != "null" {
				result = intrinsicReferenceJavaType(inferred)
			}
		}
		return result, true
	}
	if task.Type() == "method_reference" {
		inferred, ok := inferMethodReferenceResultJavaType(task, nil, ctx, source)
		if ok && inferred == "void" {
			return "Object", false
		}
		if ok && result == "Object" {
			result = intrinsicReferenceJavaType(inferred)
		}
		return result, true
	}
	if actual, ok := inferExprJavaType(task, ctx, source); ok {
		if element, callable := callableResultJavaType(actual, ctx, map[string]bool{}); callable {
			return element, true
		}
	}
	return "Object", false
}

func registerFutureIntrinsics() {
	registerInstanceNodeIntrinsic("ExecutorService", "submit", func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		result, callable := executorSubmitType(invocation, ctx, source)
		args := intrinsicArgs(invocation.ChildByFieldName("object"), "submit", source, ctx)
		if len(args) == 2 {
			return stdjavaGenericCall(ctx, "SubmitRunnableResult", []ast.Expr{javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}, append([]ast.Expr{recv}, args...))
		}
		if len(args) != 1 {
			return nil
		}
		if !callable {
			return selectorCall(recv, "Submit", args)
		}
		return stdjavaGenericCall(ctx, "SubmitCallable", []ast.Expr{javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}, append([]ast.Expr{recv}, args...))
	})
	registerInstanceIntrinsic("Callable", "call", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "CallCallableExecution", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsic("Future", "get", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return selectorCall(recv, "Get", nil)
		}
		if len(args) == 2 {
			return selectorCall(recv, "GetTimed", args)
		}
		return nil
	})
	for java, goName := range map[string]string{"cancel": "Cancel", "isDone": "IsDone", "isCancelled": "IsCancelled"} {
		goName := goName
		registerInstanceIntrinsic("Future", java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr { return selectorCall(recv, goName, args) })
		registerInstanceIntrinsicResultType("Future", java, "boolean")
	}
	for _, method := range []string{"isShutdown", "isTerminated", "awaitTermination"} {
		registerInstanceIntrinsicResultType("ExecutorService", method, "boolean")
	}
	for _, method := range []string{"newFixedThreadPool", "newSingleThreadExecutor"} {
		registerStaticIntrinsicResultType("Executors", method, "ExecutorService")
	}
	for _, unit := range []string{"NANOSECONDS", "MICROSECONDS", "MILLISECONDS", "SECONDS", "MINUTES", "HOURS", "DAYS"} {
		unit := unit
		registerStaticFieldIntrinsic("TimeUnit", unit, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr(unit, ctx) })
		registerStaticFieldIntrinsicResultType("TimeUnit", unit, "TimeUnit")
	}
}

// Find returns in the callback itself, excluding nested callback/class bodies.
func callableReturnedExpression(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "return_statement" && node.NamedChildCount() == 1 {
		return node.NamedChild(0)
	}
	switch node.Type() {
	case "lambda_expression", "class_body", "method_declaration":
		return nil
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if result := callableReturnedExpression(node.NamedChild(i)); result != nil {
			return result
		}
	}
	return nil
}

func callableResultJavaType(javaType string, ctx Ctx, seen map[string]bool) (string, bool) {
	if seen[javaType] {
		return "", false
	}
	seen[javaType] = true
	base, args := parseJavaTypeString(javaType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if stripJavaQualifier(base) == "Callable" && scope == nil {
		if len(args) == 1 {
			return args[0], true
		}
		return "Object", true
	}
	if scope == nil {
		return "", false
	}
	bindings := map[string]string{}
	for i, parameter := range scope.TypeParameters {
		if i < len(args) {
			bindings[parameter.Name] = args[i]
		}
	}
	parents := append([]string{}, scope.ImplementedInterfaces...)
	if scope.Superclass != "" {
		parents = append(parents, scope.Superclass)
	}
	for _, parent := range parents {
		if element, ok := callableResultJavaType(substituteJavaTypeParams(parent, bindings), ctx, seen); ok {
			return element, true
		}
	}
	return "", false
}
