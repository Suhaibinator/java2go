package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func fileChannelSourceDispatchCall(receiver ast.Expr, resolution *methodResolution, args []ast.Expr, ctx Ctx) ast.Expr {
	if resolution == nil || executionExpr(ctx) == nil {
		return nil
	}
	slot := fileChannelSourceProtocolSlot(resolution.def, resolution.owner, ctx)
	if slot == "" {
		return nil
	}
	return methodCall(receiver, slot, append([]ast.Expr{executionExpr(ctx)}, args...)...)
}

// Source receivers use their certified allocation carrier for ordinary virtual
// calls. Explicit super selects the source ancestor's body without redispatch.
func fileChannelSourceInvocation(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(source)
	if name != "read" && name != "write" && name != "implCloseChannel" && name != "close" && name != "isOpen" {
		return nil
	}
	object := node.ChildByFieldName("object")
	argsNode := node.ChildByFieldName("arguments")
	var scope *symbol.ClassScope
	var receiver ast.Expr
	explicitSuper := object != nil && object.Type() == "super"
	if object == nil || explicitSuper {
		if ctx.currentClass == nil || ctx.localScope == nil || ctx.localScope.IsStatic || fileChannelSourceRoot(ctx.currentClass, ctx) == nil {
			return nil
		}
		scope = ctx.currentClass
		receiver = ast.NewIdent(sourceInstanceReceiverName(ctx))
		if explicitSuper {
			scope = resolveSuperclassScopeInDeclaringContext(ctx, scope)
			if scope != nil {
				receiver = ParseExpr(object, source, ctx)
			}
		}
	} else {
		var origin inferredJavaTypeOrigin
		_, known := inferExprJavaType(object, ctx, source, &origin)
		if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope == nil || fileChannelSourceRoot(origin.nominalScope, ctx) == nil {
			return nil
		}
		scope = origin.nominalScope
		receiver = ParseExpr(object, source, ctx)
	}
	if name == "close" || name == "isOpen" {
		if nodeutil.SemanticNamedChildCount(argsNode) != 0 {
			return nil
		}
		if scope != nil {
			if selected := findBestMethodInHierarchy(scope, name, argsNode, true, false, ctx, source); selected != nil && selected.def != nil && selected.def.DeclarationNode != nil {
				return nil
			}
		}
		target := &ast.StarExpr{X: stdjavaQualifiedExpr("FileChannel", ctx)}
		view := fileChannelSourceProjection(receiver, target, ctx)
		if name == "isOpen" {
			return stdjavaCall(ctx, "FileChannelIsOpenVirtual", view)
		}
		if executionExpr(ctx) == nil {
			return nil
		}
		return stdjavaCall(ctx, "FileChannelCloseVirtualExecution", executionExpr(ctx), view)
	}
	if scope == nil || executionExpr(ctx) == nil {
		return nil
	}
	resolution := findBestMethodInHierarchy(scope, name, argsNode, true, false, ctx, source)
	if resolution == nil || resolution.def == nil {
		return nil
	}
	slot := fileChannelSourceProtocolSlot(resolution.def, resolution.owner, ctx)
	if slot == "" || len(resolution.def.Parameters) != nodeutil.SemanticNamedChildCount(argsNode) {
		return nil
	}
	args, _ := parseResolvedInvocationArguments(resolution, argsNode, source, ctx, nil, nil, node)
	// A ByteBuffer formal is selected by the declaring descriptor, not the caller.
	if len(args) > 0 && nodeutil.SemanticNamedChild(argsNode, 0).Type() == "null_literal" {
		args[0] = ast.NewIdent("nil")
	}
	if explicitSuper {
		return fileChannelSourceDispatchCall(receiver, resolution, args, ctx)
	}
	if name == "implCloseChannel" {
		if call := virtualDispatchMethodCall(receiver, resolution, args, false, ctx); call != nil {
			return call
		}
		return fileChannelSourceDispatchCall(receiver, resolution, args, ctx)
	}
	view := fileChannelSourceProjection(receiver, &ast.StarExpr{X: stdjavaQualifiedExpr("FileChannel", ctx)}, ctx)
	facade := "FileChannelReadVirtualExecution"
	switch slot {
	case fileChannelSourceReadAtSlot:
		facade = "FileChannelReadAtVirtualExecution"
	case fileChannelSourceWriteSlot:
		facade = "FileChannelWriteVirtualExecution"
	case fileChannelSourceWriteAtSlot:
		facade = "FileChannelWriteAtVirtualExecution"
	}
	return stdjavaCall(ctx, facade, append([]ast.Expr{executionExpr(ctx), view}, args...)...)
}
