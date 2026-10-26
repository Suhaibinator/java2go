package transpiler

import sitter "github.com/smacker/go-tree-sitter"

// These inherited final, fixed-arity methods precede source varargs overloads.
// The same resolution feeds emission and return inference. Explicit source
// methods with other arities continue through the ordinary overload resolver.
func implicitRuntimeClassMethod(node *sitter.Node, selected *methodResolution, ctx Ctx, source []byte) string {
	if node == nil || node.ChildByFieldName("object") != nil || ctx.currentClass == nil || ctx.localScope == nil || ctx.localScope.IsStatic || invocationArgumentCount(node) != 0 {
		return ""
	}
	if selected != nil && selected.def != nil && len(selected.def.Parameters) == 0 {
		return ""
	}
	name := node.ChildByFieldName("name")
	if name == nil {
		return ""
	}
	switch name.Content(source) {
	case "getClass":
		return "ObjectGetClass"
	case "getDeclaringClass":
		if ctx.currentClass.IsEnum {
			return "EnumGetDeclaringClass"
		}
	}
	return ""
}
