package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func listVirtualReceiverSupported(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	object := node.ChildByFieldName("object")
	if object == nil {
		return false
	}
	var origin inferredJavaTypeOrigin
	typ, known := inferExprJavaType(object, ctx, source, &origin)
	if !known || origin.parameter != nil || origin.nominalScope != nil || origin.unresolved {
		return false
	}
	base, _ := parseJavaTypeString(typ)
	owner, owned := canonicalIntrinsicOwner(base, ctx)
	return owned && containsString([]string{"java.util.List", "java.util.ArrayList", "java.util.LinkedList", "java.util.AbstractList"}, owner)
}
func listVirtualRead(raw ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if collectionResultIsErased(node, ctx, source) {
		return raw
	}
	elements := receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, source)
	if len(elements) == 0 {
		return raw
	}
	element := readableWildcardProjection(elements[0])
	descriptor, known := javaTypeDescriptorExpr(element, ctx)
	if !known {
		descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{raw, descriptor})
}

// The only producers of AbstractList projections are attested source parent
// constructors. In a conversion with no such producer, retain the existing
// native List lowering. Where a producer exists, every List alias must carry
// the caller execution, regardless of the receiver's initial allocation.
func sourceAbstractListProjectionPossible(ctx Ctx) bool {
	files := map[*symbol.FileScope]bool{}
	if ctx.currentFile != nil {
		files[ctx.currentFile] = true
	}
	for _, scope := range allSourceClassScopes() {
		if sourceAbstractListDirectRoot(scope, ctx) {
			return true
		}
		if file := findFileScopeForClassScope(scope, ctx); file != nil {
			files[file] = true
		}
	}
	// Named local classes are hoisted during emission. Their still-unhoisted
	// headers are a conservative possibility, never permission to bind a view.
	for file := range files {
		if file.ParsedRoot() == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil {
			return true
		}
		possible := false
		var visit func(*sitter.Node)
		visit = func(node *sitter.Node) {
			if node == nil || possible {
				return
			}
			if node.Type() == "class_declaration" {
				if super := node.ChildByFieldName("superclass"); super != nil {
					for _, typ := range supplierHeaderTypeNodes(super) {
						written := typ.Content(file.Source)
						base, _ := parseJavaTypeString(written)
						if stripJavaQualifier(base) == "AbstractList" || strings.Contains(written, "\\u") || strings.Contains(written, "@") {
							possible = true
							return
						}
					}
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				visit(child)
			}
		}
		visit(file.ParsedRoot())
		if possible {
			return true
		}
	}
	return false
}

func registerListVirtualOperation(owner, name string, generator nodeIntrinsicGenerator) {
	key := intrinsicKey{owner, name}
	nativeNode, nativeCall := instanceNodeIntrinsics[key], instanceIntrinsics[key]
	registerInstanceNodeIntrinsic(owner, name, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		safeNative := containsString([]string{"get", "set", "size", "clear", "isEmpty", "contains", "indexOf", "remove"}, name)
		if safeNative && !sourceAbstractListProjectionPossible(ctx) {
			if nativeNode != nil {
				if result := nativeNode(receiver, node, ctx, source); result != nil {
					return result
				}
			}
			if nativeCall != nil {
				args := intrinsicArgs(node.ChildByFieldName("object"), name, source, ctx)
				if result := nativeCall(receiver, args, ctx); result != nil {
					return result
				}
			}
		}
		return generator(receiver, node, ctx, source)
	})
}

func registerListVirtualIntrinsics() {
	for _, owner := range listTypeNames {
		for _, operation := range []struct {
			name, helper, result string
			arity                int
		}{
			{"get", "CollectionListGetExecution", "", 1}, {"set", "CollectionListSetExecution", "", 2},
			{"size", "CollectionSizeExecution", "int", 0}, {"clear", "CollectionClearExecution", "void", 0}, {"isEmpty", "CollectionIsEmptyExecution", "boolean", 0},
			{"contains", "CollectionContainsExecution", "boolean", 1}, {"indexOf", "CollectionListIndexOfExecution", "int", 1}, {"lastIndexOf", "CollectionListLastIndexOfExecution", "int", 1},
			{"removeAll", "CollectionRemoveAllExecution", "boolean", 1}, {"retainAll", "CollectionRetainAllExecution", "boolean", 1}, {"containsAll", "CollectionContainsAllExecution", "boolean", 1},
		} {
			registerListVirtualOperation(owner, operation.name, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if !listVirtualReceiverSupported(node, ctx, source) {
					return nil
				}
				if invocationArgumentCount(node) != operation.arity {
					return unsupportedIntrinsicValue(node, operation.result, source, ctx)
				}
				args := intrinsicArgs(node.ChildByFieldName("object"), operation.name, source, ctx)
				raw := stdjavaCall(ctx, operation.helper, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
				if operation.name == "get" || operation.name == "set" {
					return listVirtualRead(raw, node, ctx, source)
				}
				return raw
			})
			if operation.result != "" {
				registerInstanceIntrinsicResultType(owner, operation.name, operation.result)
			}
		}
		registerListVirtualOperation(owner, "add", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !listVirtualReceiverSupported(node, ctx, source) {
				return nil
			}
			count := invocationArgumentCount(node)
			helper := "CollectionAddExecution"
			if count == 2 {
				helper = "CollectionListAddAtExecution"
			} else if count != 1 {
				return unsupportedIntrinsicValue(node, "boolean", source, ctx)
			}
			args := intrinsicArgs(node.ChildByFieldName("object"), "add", source, ctx)
			return stdjavaCall(ctx, helper, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
		})
		registerListVirtualOperation(owner, "remove", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !listVirtualReceiverSupported(node, ctx, source) {
				return nil
			}
			if invocationArgumentCount(node) != 1 {
				return unsupportedIntrinsicValue(node, "java.lang.Object", source, ctx)
			}
			helper := "CollectionRemoveExecution"
			indexed := listRemoveUsesIndex(node, ctx, source)
			if indexed {
				helper = "CollectionListRemoveAtExecution"
			}
			args := intrinsicArgs(node.ChildByFieldName("object"), "remove", source, ctx)
			raw := stdjavaCall(ctx, helper, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
			if indexed {
				return listVirtualRead(raw, node, ctx, source)
			}
			return raw
		})
		registerListVirtualOperation(owner, "addAll", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !listVirtualReceiverSupported(node, ctx, source) {
				return nil
			}
			count := invocationArgumentCount(node)
			helper := "CollectionAddAllExecution"
			if count == 2 {
				helper = "CollectionListAddAllAtExecution"
			} else if count != 1 {
				return unsupportedIntrinsicValue(node, "boolean", source, ctx)
			}
			args := intrinsicArgs(node.ChildByFieldName("object"), "addAll", source, ctx)
			return stdjavaCall(ctx, helper, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
		})
		registerInstanceIntrinsicResultType(owner, "addAll", "boolean")
		registerListVirtualOperation(owner, "toArray", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !listVirtualReceiverSupported(node, ctx, source) {
				return nil
			}
			count := invocationArgumentCount(node)
			if count == 0 {
				return stdjavaCall(ctx, "CollectionToArrayExecution", intrinsicExecutionExpr(ctx), receiver)
			}
			if count != 1 {
				return unsupportedIntrinsicValue(node, "java.lang.Object[]", source, ctx)
			}
			argument := invocationArgumentNode(node, 0)
			actual, known := inferExprJavaType(argument, ctx, source)
			_, rank := javaArrayTypeParts(actual)
			if !known || rank == 0 {
				if argument.Type() != "null_literal" {
					return unsupportedIntrinsicValue(node, "java.util.List.toArray(IntFunction)", source, ctx)
				}
			}
			return stdjavaCall(ctx, "CollectionToArrayIntoExecution", intrinsicExecutionExpr(ctx), receiver, ParseExpr(argument, source, ctx))
		})
		registerInstanceIntrinsicResultType(owner, "toArray", "java.lang.Object[]")
		registerInstanceIntrinsicDerivedResultType(owner, "toArray", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if invocationArgumentCount(node) == 0 {
				return "java.lang.Object[]", true
			}
			if invocationArgumentCount(node) != 1 {
				return "", false
			}
			typ, known := inferExprJavaType(invocationArgumentNode(node, 0), ctx, source)
			_, rank := javaArrayTypeParts(typ)
			return typ, known && rank > 0
		})
		registerListVirtualOperation(owner, "listIterator", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !listVirtualReceiverSupported(node, ctx, source) {
				return nil
			}
			args := []ast.Expr{intrinsicExecutionExpr(ctx), receiver}
			switch invocationArgumentCount(node) {
			case 0:
				args = append(args, intLit(0))
			case 1:
				args = append(args, ParseExpr(invocationArgumentNode(node, 0), source, ctx))
			default:
				return unsupportedIntrinsicValue(node, "java.util.ListIterator", source, ctx)
			}
			return stdjavaCall(ctx, "CollectionListIteratorExecution", args...)
		})
		registerInstanceIntrinsicDerivedResultType(owner, "listIterator", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			args := receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, source)
			if len(args) == 1 {
				return "java.util.ListIterator<" + args[0] + ">", true
			}
			return "java.util.ListIterator", true
		})
	}
	registerListIteratorIntrinsics()
}
func registerListIteratorIntrinsics() {
	for _, operation := range []struct {
		name, helper, result string
		arity                int
	}{
		{"hasNext", "IteratorHasNextExecution", "boolean", 0}, {"next", "IteratorNextExecution", "", 0}, {"remove", "IteratorRemoveExecution", "void", 0},
		{"hasPrevious", "ListIteratorHasPreviousExecution", "boolean", 0}, {"previous", "ListIteratorPreviousExecution", "", 0},
		{"nextIndex", "ListIteratorNextIndexExecution", "int", 0}, {"previousIndex", "ListIteratorPreviousIndexExecution", "int", 0},
		{"set", "ListIteratorSetExecution", "void", 1}, {"add", "ListIteratorAddExecution", "void", 1},
	} {
		registerInstanceNodeIntrinsic("ListIterator", operation.name, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != operation.arity {
				return unsupportedIntrinsicValue(node, operation.result, source, ctx)
			}
			args := []ast.Expr{intrinsicExecutionExpr(ctx), receiver}
			if operation.arity == 1 {
				argument := invocationArgumentNode(node, 0)
				args = append(args, ParseExpr(argument, source, ctx))
			}
			raw := stdjavaCall(ctx, operation.helper, args...)
			if operation.name == "next" || operation.name == "previous" {
				return listVirtualRead(raw, node, ctx, source)
			}
			return raw
		})
		if operation.result != "" {
			registerInstanceIntrinsicResultType("ListIterator", operation.name, operation.result)
		}
	}
}
func sourceAbstractListProtocolReservedSelector(name string) bool {
	return name == "AbstractListStateJava2go" || strings.HasPrefix(name, "CollectionList") || name == "CollectionAddAllJava2goExecution" || name == "CollectionRemoveAllJava2goExecution" || name == "CollectionRetainAllJava2goExecution" || name == "CollectionToArrayJava2goExecution" || name == "CollectionToArrayIntoJava2goExecution"
}
