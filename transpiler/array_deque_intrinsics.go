package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func canonicalArrayDequeOwner(javaType string, typeParams []string, ctx Ctx) string {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || containsString(typeParams, base) || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return ""
	}
	switch owner {
	case "java.util.ArrayDeque", "java.util.Deque", "java.util.Queue":
		return owner
	}
	return ""
}
func arrayDequeRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	owner := canonicalArrayDequeOwner(javaType, typeParams, ctx)
	if owner == "" || len(typeArgs) > 1 {
		return nil, false
	}
	switch owner {
	case "java.util.Deque":
		return stdjavaQualifiedExpr("JavaDeque", ctx), true
	case "java.util.Queue":
		return stdjavaQualifiedExpr("JavaQueue", ctx), true
	}
	// Java E remains in source inference; every class view has one erased carrier.
	return &ast.StarExpr{X: stdjavaQualifiedExpr("ArrayDeque", ctx)}, true
}
func arrayDequeElement(invocation *sitter.Node, ctx Ctx, source []byte) string {
	types := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
	if len(types) == 1 {
		return readableWildcardProjection(types[0])
	}
	return "java.lang.Object"
}
func arrayDequeArgs(invocation *sitter.Node, expected []string, ctx Ctx, source []byte) []ast.Expr {
	var result []ast.Expr
	for i, node := range nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments")) {
		child := ctx.Clone()
		child.expectedType = ""
		child.expectedTypeRoot = node
		if i < len(expected) {
			child.expectedType = expected[i]
		}
		value := ParseExpr(node, source, child)
		if child.expectedType != "" {
			value = coerceArgumentToExpectedType(value, node, child.expectedType, ctx, source)
		}
		result = append(result, value)
	}
	return result
}
func arrayDequeResult(call ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if collectionResultIsErased(invocation, ctx, source) {
		return call
	}
	return arrayDequeElementResult(call, arrayDequeElement(invocation, ctx, source), ctx)
}
func arrayDequeElementResult(call ast.Expr, element string, ctx Ctx) ast.Expr {
	descriptor, known := javaTypeDescriptorExpr(element, ctx)
	if !known {
		descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{call, descriptor})
}
func init() {
	for _, owner := range []string{"java.util.ArrayDeque", "java.util.Deque", "java.util.Queue"} {
		registerIntrinsicOwner(owner, true)
	}
	registerConstructorNodeIntrinsic("ArrayDeque", func(types, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		written := node.ChildByFieldName("type")
		if written == nil && node.Type() == "method_reference" {
			children := nodeutil.SemanticNamedChildrenOf(node)
			if len(children) > 0 {
				written = children[0]
			}
		}
		if written == nil || canonicalArrayDequeOwner(written.Content(source), inScopeTypeParameters(ctx), ctx) != "java.util.ArrayDeque" {
			return nil
		}
		if len(types) == 0 {
			types = []ast.Expr{ast.NewIdent("any")}
		}
		if len(types) != 1 {
			return nil
		}
		if len(args) == 0 {
			return stdjavaGenericCall(ctx, "NewArrayDeque", types, nil)
		}
		if len(args) != 1 {
			return nil
		}
		actual, _ := inferExprJavaType(invocationArgumentNode(node, 0), ctx, source)
		primitive, primitiveKnown := javaPrimitiveType(actual)
		if !primitiveKnown {
			primitive, primitiveKnown = builtinJavaWrapperPrimitive(actual, ctx)
		}
		if primitiveKnown && (primitive == "byte" || primitive == "short" || primitive == "char" || primitive == "int") {
			value := args[0]
			if converted, ok := convertJavaValue(value, actual, "int", ctx); ok {
				value = converted
			}
			return stdjavaGenericCall(ctx, "NewArrayDequeWithCapacity", types, []ast.Expr{value})
		}
		return stdjavaGenericCall(ctx, "NewArrayDequeFromCollection", types, []ast.Expr{args[0], intrinsicExecutionExpr(ctx)})
	})
	for _, owner := range []string{"ArrayDeque", "Deque", "Queue"} {
		registerArrayDequeOperations(owner)
	}
}
func arrayDequeReceiverSupported(node *sitter.Node, owner string, ctx Ctx, source []byte) bool {
	actual, known := inferExprJavaType(node.ChildByFieldName("object"), ctx, source)
	return known && canonicalArrayDequeOwner(actual, inScopeTypeParameters(ctx), ctx) == "java.util."+owner
}
func registerArrayDequeOperations(owner string) {
	queue := map[string]string{"add": "QueueAddExecution", "offer": "QueueOfferExecution", "remove": "QueueRemoveExecution", "poll": "QueuePollExecution", "element": "QueueElementExecution", "peek": "QueuePeekExecution"}
	if owner != "Queue" {
		for _, end := range []string{"First", "Last"} {
			for _, verb := range []string{"add", "offer", "remove", "poll", "get", "peek"} {
				queue[verb+end] = "Deque" + strings.ToUpper(verb[:1]) + verb[1:] + end + "Execution"
			}
		}
		for java, runtime := range map[string]string{"push": "DequePushExecution", "pop": "DequePopExecution", "removeFirstOccurrence": "DequeRemoveFirstOccurrenceExecution", "removeLastOccurrence": "DequeRemoveLastOccurrenceExecution", "descendingIterator": "DequeDescendingIteratorExecution"} {
			queue[java] = runtime
		}
	}
	for java, runtime := range queue {
		java, runtime := java, runtime
		input := java == "add" || java == "offer" || java == "push" || strings.HasPrefix(java, "add") || strings.HasPrefix(java, "offer") || strings.HasSuffix(java, "Occurrence")
		arity := 0
		if input {
			arity = 1
		}
		read := !input && java != "descendingIterator"
		result := "void"
		if strings.HasPrefix(java, "offer") || java == "add" || strings.HasSuffix(java, "Occurrence") {
			result = "boolean"
		}
		if read {
			result = "java.lang.Object"
		}
		if java == "descendingIterator" {
			result = "java.util.Iterator"
		}
		arrayDequeReferenceOperations[intrinsicKey{owner, java}] = arrayDequeReferenceOperation{runtime, result, arity, read}
		registerInstanceNodeIntrinsic(owner, java, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !arrayDequeReceiverSupported(node, owner, ctx, source) {
				return nil
			}
			count := invocationArgumentCount(node)
			if java == "remove" && count == 1 {
				args := arrayDequeArgs(node, []string{"java.lang.Object"}, ctx, source)
				return stdjavaCall(ctx, "CollectionRemoveExecution", intrinsicExecutionExpr(ctx), recv, args[0])
			}
			if count != arity {
				return nil
			}
			var expected []string
			if input {
				element := arrayDequeElement(node, ctx, source)
				if strings.HasSuffix(java, "Occurrence") {
					element = "java.lang.Object"
				}
				expected = []string{element}
			}
			args := arrayDequeArgs(node, expected, ctx, source)
			call := stdjavaCall(ctx, runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
			if read {
				return arrayDequeResult(call, node, ctx, source)
			}
			return call
		})
		registerInstanceIntrinsicResultType(owner, java, result)
		if read || java == "descendingIterator" {
			registerInstanceIntrinsicDerivedResultType(owner, java, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
				if java == "remove" && invocationArgumentCount(node) == 1 {
					return "boolean", true
				}
				element := arrayDequeElement(node, ctx, source)
				if java == "descendingIterator" {
					return "java.util.Iterator<" + element + ">", true
				}
				return element, true
			})
		}
	}
	for _, spec := range []struct {
		java, runtime, result string
		arity                 int
	}{
		{"size", "CollectionSizeExecution", "int", 0}, {"isEmpty", "CollectionIsEmptyExecution", "boolean", 0}, {"clear", "CollectionClearExecution", "void", 0}, {"contains", "CollectionContainsExecution", "boolean", 1}, {"addAll", "ArrayDequeCollectionAddAllExecution", "boolean", 1}, {"containsAll", "CollectionContainsAllExecution", "boolean", 1},
		{"removeAll", "ArrayDequeCollectionRemoveAllExecution", "boolean", 1}, {"retainAll", "ArrayDequeCollectionRetainAllExecution", "boolean", 1}, {"iterator", "IterableIteratorExecution", "java.util.Iterator", 0}, {"toString", "ArrayDequeCollectionStringExecution", "java.lang.String", 0},
	} {
		spec := spec
		arrayDequeReferenceOperations[intrinsicKey{owner, spec.java}] = arrayDequeReferenceOperation{spec.runtime, spec.result, spec.arity, false}
		registerInstanceNodeIntrinsic(owner, spec.java, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !arrayDequeReceiverSupported(node, owner, ctx, source) {
				return nil
			}
			if invocationArgumentCount(node) != spec.arity {
				return nil
			}
			var expected []string
			if spec.arity == 1 {
				expected = []string{"java.lang.Object"}
				if spec.java != "contains" {
					expected = []string{"java.util.Collection<?>"}
				}
			}
			args := arrayDequeArgs(node, expected, ctx, source)
			return stdjavaCall(ctx, spec.runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
		})
		registerInstanceIntrinsicResultType(owner, spec.java, spec.result)
		if spec.java == "iterator" {
			registerInstanceIntrinsicDerivedResultType(owner, spec.java, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
				return "java.util.Iterator<" + arrayDequeElement(node, ctx, source) + ">", true
			})
		}
	}
	registerInstanceNodeIntrinsic(owner, "toArray", func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if !arrayDequeReceiverSupported(node, owner, ctx, source) {
			return nil
		}
		if invocationArgumentCount(node) == 0 {
			return stdjavaCall(ctx, "ArrayDequeCollectionToArrayExecution", intrinsicExecutionExpr(ctx), recv)
		}
		if invocationArgumentCount(node) == 1 {
			actual, known := inferExprJavaType(invocationArgumentNode(node, 0), ctx, source)
			_, rank := javaArrayTypeParts(actual)
			if !known || rank == 0 {
				return unsupportedIntrinsicOwnerValue("java.util."+owner+".toArray(IntFunction)", node, source, ctx)
			}
			args := arrayDequeArgs(node, nil, ctx, source)
			return stdjavaCall(ctx, "ArrayDequeCollectionToArrayIntoExecution", intrinsicExecutionExpr(ctx), recv, args[0])
		}
		return nil
	})
	registerInstanceIntrinsicResultType(owner, "toArray", "java.lang.Object[]")
	registerInstanceIntrinsicDerivedResultType(owner, "toArray", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if invocationArgumentCount(node) == 0 {
			return "java.lang.Object[]", true
		}
		if invocationArgumentCount(node) != 1 {
			return "", false
		}
		actual, known := inferExprJavaType(invocationArgumentNode(node, 0), ctx, source)
		_, rank := javaArrayTypeParts(actual)
		return actual, known && rank > 0
	})
	if owner == "ArrayDeque" {
		registerInstanceNodeIntrinsic(owner, "clone", func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !arrayDequeReceiverSupported(node, owner, ctx, source) || invocationArgumentCount(node) != 0 {
				return nil
			}
			return selectorCall(recv, "Clone", nil)
		})
		registerInstanceIntrinsicDerivedResultType(owner, "clone", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			return inferExprJavaType(node.ChildByFieldName("object"), ctx, source)
		})
	}
}

// Method references use the same runtime operation signatures as direct calls.
// Admission retains the complete Java owner before a simple dispatch key is used.
type arrayDequeReferenceOperation struct {
	runtime, result string
	arity           int
	elementResult   bool
}

var arrayDequeReferenceOperations = map[intrinsicKey]arrayDequeReferenceOperation{}

func arrayDequeMethodReferenceIntrinsic(javaType, method string, unbound bool, node *sitter.Node, source []byte, ctx Ctx) (intrinsicGenerator, string, bool) {
	owner := canonicalArrayDequeOwner(javaType, inScopeTypeParameters(ctx), ctx)
	if owner == "" {
		return nil, "", false
	}
	parameters, samResult := methodReferenceJavaSignature(ctx)
	// For a raw unbound qualifier, Java's second search obtains E from the
	// target SAM's first parameter. Explicit qualifier arguments still win.
	_, elements := parseJavaTypeString(javaType)
	if unbound && len(parameters) > 0 && len(elements) == 0 {
		receiverOwner := canonicalArrayDequeOwner(parameters[0], inScopeTypeParameters(ctx), ctx)
		compatible := receiverOwner == owner || receiverOwner == "java.util.ArrayDeque" || owner == "java.util.Queue" && receiverOwner == "java.util.Deque"
		if receiverOwner != "" && compatible {
			_, elements = parseJavaTypeString(parameters[0])
		}
	}
	if unbound && len(parameters) > 0 {
		parameters = parameters[1:]
	}
	operation, supported := arrayDequeReferenceOperations[intrinsicKey{stripJavaQualifier(owner), method}]
	if method == "remove" && len(parameters) == 1 {
		operation = arrayDequeReferenceOperation{"CollectionRemoveExecution", "boolean", 1, false}
		supported = true
	}
	if method == "clone" && owner == "java.util.ArrayDeque" && len(parameters) == 0 {
		return func(recv ast.Expr, args []ast.Expr, body Ctx) ast.Expr { return selectorCall(recv, "Clone", nil) }, javaType, true
	}
	if method == "toArray" {
		if len(parameters) == 0 {
			operation = arrayDequeReferenceOperation{"ArrayDequeCollectionToArrayExecution", "java.lang.Object[]", 0, false}
			supported = true
		} else if len(parameters) == 1 {
			if _, rank := javaArrayTypeParts(parameters[0]); rank == 0 {
				return func(ast.Expr, []ast.Expr, Ctx) ast.Expr {
					return unsupportedIntrinsicOwnerValue(owner+".toArray(IntFunction)", node, source, ctx)
				}, "", true
			}
			operation = arrayDequeReferenceOperation{"ArrayDequeCollectionToArrayIntoExecution", parameters[0], 1, false}
			supported = true
		}
	}
	if !supported || len(parameters) != operation.arity {
		return nil, "", false
	}
	result := operation.result
	element := "java.lang.Object"
	if len(elements) == 1 {
		element = readableWildcardProjection(elements[0])
	}
	logicalElement := element
	if operation.elementResult {
		// A discarded result has no consumer checkcast. Reference consumers
		// check their adapted return type; primitive consumers first check E
		// and then use the ordinary unboxing conversion.
		if samResult == "void" {
			element = "java.lang.Object"
		} else if _, primitive := javaPrimitiveType(samResult); !primitive {
			element = samResult
		}
		result = element
	}
	if method == "iterator" || method == "descendingIterator" {
		result = "java.util.Iterator<" + element + ">"
	}
	return func(recv ast.Expr, args []ast.Expr, body Ctx) ast.Expr {
		if len(args) != operation.arity {
			return nil
		}
		if len(args) == 1 {
			expected := parameters[0]
			switch {
			case method == "add" || method == "push" || strings.HasPrefix(method, "addFirst") || strings.HasPrefix(method, "addLast") || strings.HasPrefix(method, "offer"):
				expected = logicalElement
			case method == "remove" || method == "contains" || strings.HasSuffix(method, "Occurrence"):
				expected = "java.lang.Object"
			case method == "addAll" || method == "containsAll" || method == "removeAll" || method == "retainAll":
				expected = "java.util.Collection<?>"
			}
			args[0] = methodReferenceConvertedArgument(args[0], parameters[0], expected, body)
		}
		call := stdjavaCall(body, operation.runtime, append([]ast.Expr{intrinsicExecutionExpr(body), recv}, args...)...)
		if operation.elementResult && samResult != "void" {
			return arrayDequeElementResult(call, element, body)
		}
		return call
	}, result, true
}
