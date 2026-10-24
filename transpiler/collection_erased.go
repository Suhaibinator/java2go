package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Collection is physically erased. Its Java type arguments remain available
// to binding; runtime reads perform the consumer's required element conversion.
func registerErasedCollectionIntrinsics() {
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
	} {
		registerInstanceIntrinsic("Collection", operation.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != operation.arity {
				return nil
			}
			return stdjavaCall(ctx, operation.runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
		})
		registerInstanceIntrinsicResultType("Collection", operation.java, operation.result)
	}
}

func erasedCollectionExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	typ, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return false
	}
	base, _ := parseJavaTypeString(typ)
	return resolveClassScopeByQualifiedName(ctx, base) == nil && stripJavaQualifier(strings.TrimSpace(base)) == "Collection"
}

// Canonical and source Iterable values use the same live iterator protocol.
func erasedIterableExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	typ, known := inferExprJavaType(node, ctx, source)
	if !known {
		return false
	}
	_, iterable := iterationElementType(typ, "java.lang.Iterable", ctx)
	return iterable
}

// No checkcast is emitted by Java when the erased result is discarded or used
// as Object. Keep the raw return at those consumer boundaries.
func collectionResultIsErased(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	if parent := invocation.Parent(); parent != nil && parent.Type() == "cast_expression" && parent.NamedChildCount() > 0 {
		if collectionObjectConsumer(parent.NamedChild(0).Content(source), ctx) {
			return true
		}
	}
	if parent := invocation.Parent(); parent != nil && parent.Type() == "expression_statement" {
		return true
	}
	return expectedTypeTargetsExpression(ctx, invocation) && collectionObjectConsumer(ctx.expectedType, ctx)
}

func collectionObjectConsumer(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	if base == "java.lang.Object" {
		return true
	}
	if base != "Object" || containsString(inScopeTypeParameters(ctx), base) || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	if ctx.currentFile != nil {
		if owner, imported := ctx.currentFile.Imports[base]; imported {
			return owner == "java.lang"
		}
	}
	return true
}

func registerListErasedResults() {
	for _, name := range listTypeNames {
		for _, operation := range []struct {
			java, typed, erased string
			arity               int
		}{
			{"get", "Get", "GetObject", 1}, {"set", "Set", "SetObject", 2},
		} {
			registerInstanceNodeIntrinsic(name, operation.java, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if invocationArgumentCount(invocation) != operation.arity {
					return nil
				}
				method := operation.typed
				if collectionResultIsErased(invocation, ctx, source) {
					method = operation.erased
				}
				args := intrinsicArgs(invocation.ChildByFieldName("object"), operation.java, source, ctx)
				return methodCall(recv, method, args...)
			})
		}
	}
}

func rawListReceiver(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	typ, known := inferExprJavaType(invocation.ChildByFieldName("object"), ctx, source)
	base, args := parseJavaTypeString(typ)
	return known && len(args) == 0 && containsString(listTypeNames, stripJavaQualifier(base)) && resolveClassScopeByQualifiedName(ctx, base) == nil
}

func registerRawListIntrinsics() {
	for _, name := range listTypeNames {
		for _, operation := range []struct {
			java, typed, erased string
			arity               int
		}{
			{"add", "Add", "CollectionAddExecution", 1},
			{"get", "Get", "CollectionListGetExecution", 1},
			{"set", "Set", "CollectionListSetExecution", 2},
			{"size", "Size", "CollectionSizeExecution", 0},
			{"clear", "Clear", "CollectionClearExecution", 0},
			{"contains", "Contains", "CollectionContainsExecution", 1},
			{"isEmpty", "IsEmpty", "CollectionIsEmptyExecution", 0},
		} {
			registerInstanceNodeIntrinsic(name, operation.java, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if invocationArgumentCount(invocation) != operation.arity {
					return nil
				}
				args := intrinsicArgs(invocation.ChildByFieldName("object"), operation.java, source, ctx)
				if rawListReceiver(invocation, ctx, source) {
					return stdjavaCall(ctx, operation.erased, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
				}
				method := operation.typed
				if (operation.java == "get" || operation.java == "set") && collectionResultIsErased(invocation, ctx, source) {
					method += "Object"
				}
				if operation.java == "contains" {
					args = append(args, intrinsicExecutionExpr(ctx))
				}
				return methodCall(recv, method, args...)
			})
		}
	}
}

func registerCollectionIterators() {
	for _, receiver := range append(append([]string{"Collection", "Iterable"}, listTypeNames...), setTypeNames...) {
		registerInstanceIntrinsic(receiver, "iterator", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "IterableIteratorExecution", intrinsicExecutionExpr(ctx), recv)
		})
	}
	registerInstanceIntrinsic("Iterator", "hasNext", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "IteratorHasNextExecution", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsicResultType("Iterator", "hasNext", "boolean")
	registerInstanceIntrinsic("Iterator", "remove", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "IteratorRemoveExecution", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsicResultType("Iterator", "remove", "void")
	registerInstanceNodeIntrinsic("Iterator", "next", func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if invocationArgumentCount(invocation) != 0 {
			return nil
		}
		raw := stdjavaCall(ctx, "IteratorNextExecution", intrinsicExecutionExpr(ctx), recv)
		if collectionResultIsErased(invocation, ctx, source) {
			return raw
		}
		elements := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
		if len(elements) == 0 {
			return raw
		}
		element := readableWildcardProjection(elements[0])
		physical := javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)
		descriptor, known := javaTypeDescriptorExpr(element, ctx)
		if !known {
			descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
		}
		return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{raw, descriptor})
	})
}
