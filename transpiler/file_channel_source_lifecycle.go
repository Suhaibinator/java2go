package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const fileChannelSourceViewMethod = "Java2goFileChannelView"

// Lifecycle methods are final zero-argument declarations. Actual source
// overloads keep ordinary source dispatch; source ancestry supplies only the
// inherited canonical declaration and its stored view.
func fileChannelLifecycleSelection(object *sitter.Node, method string, ctx Ctx, source []byte) (*symbol.ClassScope, bool) {
	if object == nil || (method != "close" && method != "isOpen") || invocationArgumentCount(object.Parent()) != 0 {
		return nil, false
	}
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(object, ctx, source, &origin)
	if !known || origin.parameter != nil || origin.unresolved {
		return nil, false
	}
	if scope := origin.nominalScope; scope != nil {
		if !fileChannelSourceHasCanonicalRoot(scope, ctx) {
			return nil, false
		}
		if selected := findBestMethodInHierarchy(scope, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source); selected != nil && selected.def != nil && selected.def.DeclarationNode != nil {
			return nil, false
		}
		return scope, true
	}
	return nil, fileChannelReadExternalExpression(object, "java.nio.channels.FileChannel", ctx, source)
}

func registerFileChannelSourceLifecycleIntrinsics() {
	registerInstanceIntrinsicResultType("FileChannel", "isOpen", "boolean")
	for _, method := range []string{"close", "isOpen"} {
		registerInstanceNodeIntrinsic("FileChannel", method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			scope, selected := fileChannelLifecycleSelection(node.ChildByFieldName("object"), method, ctx, source)
			if !selected || (method == "close" && executionExpr(ctx) == nil) {
				return nil
			}
			if scope != nil {
				receiver = methodCall(receiver, fileChannelSourceViewMethod)
			}
			if method == "close" {
				return stdjavaCall(ctx, "FileChannelCloseVirtualExecution", executionExpr(ctx), receiver)
			}
			return stdjavaCall(ctx, "FileChannelIsOpenVirtual", receiver)
		})
	}
}

// The source declaration does not contain inherited external final methods.
// Their result types follow the same ancestry and selection proof as lowering.
func fileChannelSourceInheritedLifecycle(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || node.Type() != "method_invocation" || ctx.localScope == nil || ctx.localScope.IsStatic {
		return false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(source) != "isOpen" || invocationArgumentCount(node) != 0 {
		return false
	}
	object := node.ChildByFieldName("object")
	if object != nil && object.Type() != "super" {
		return false
	}
	if fileChannelSourceRoot(ctx.currentClass, ctx) == nil {
		return false
	}
	scope := ctx.currentClass
	if object != nil {
		scope = resolveSuperclassScopeInDeclaringContext(ctx, scope)
	}
	if scope != nil {
		if selected := findBestMethodInHierarchy(scope, "isOpen", node.ChildByFieldName("arguments"), true, false, ctx, source); selected != nil && selected.def != nil && selected.def.DeclarationNode != nil {
			return false
		}
	}
	return true
}
