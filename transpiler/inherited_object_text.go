package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// Object's fixed-arity zero-argument method remains inherited when a source
// declaration only supplies a variable-arity overload. Actual source overrides
// continue through ordinary method resolution and its virtual dispatch bridges.
func inheritedObjectTextSelected(object *sitter.Node, method string, ctx Ctx, source []byte) bool {
	if object == nil || method != "toString" || invocationArgumentCount(object.Parent()) != 0 {
		return false
	}
	target := resolveInvocationTarget(object, ctx, source)
	if target == nil || target.classScope == nil || target.classScope.IsEnum {
		return false
	}
	selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source)
	return selected == nil || selected.def == nil || len(selected.def.Parameters) != 0
}

func inheritedObjectTextInvocation(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if !inheritedObjectTextSelected(object, method, ctx, source) {
		return nil
	}
	if object.Type() == "super" {
		return stdjavaCall(ctx, "ObjectDefaultJavaStringExecution", intrinsicExecutionExpr(ctx), ast.NewIdent(ShortName(ctx.className)))
	}
	receiver := stdjavaCall(ctx, "ReferenceRequireNonNull", ParseExpr(object, source, ctx))
	return stdjavaCall(ctx, "JavaStringValueOfExecution", intrinsicExecutionExpr(ctx), receiver)
}

// Implicit lookup observes the same fixed-arity phase as an explicit receiver;
// lexical enclosing static lookup occurs only if no inherited member applies.
func implicitInheritedObjectTextSelected(node *sitter.Node, selected *methodResolution, ctx Ctx, source []byte) bool {
	if node == nil || node.ChildByFieldName("object") != nil || ctx.currentClass == nil || ctx.currentClass.IsEnum || ctx.localScope == nil || ctx.localScope.IsStatic {
		return false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(source) != "toString" || invocationArgumentCount(node) != 0 {
		return false
	}
	return selected == nil || selected.def == nil || len(selected.def.Parameters) != 0
}
