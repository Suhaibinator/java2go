package transpiler

import sitter "github.com/smacker/go-tree-sitter"

// knownArrayMethodIsAbsent proves only that a selected name is not a member
// of an array. Known Object names retain their existing signature/access
// routing; this is not a replacement for Java overload or access checking.
func knownArrayMethodIsAbsent(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || node.Type() != "method_invocation" {
		return false
	}
	object, name := node.ChildByFieldName("object"), node.ChildByFieldName("name")
	if object == nil || name == nil {
		return false
	}
	// The receiver's static type cannot come from its consumer's assignment
	// target. Keep the declaration/binder context while removing target typing.
	proofCtx := ctx.Clone()
	proofCtx.expectedType, proofCtx.expectedTypeRoot = "", nil
	javaType, known := inferExprJavaType(object, proofCtx, source)
	if !known {
		return false
	}
	if _, rank := javaArrayTypeParts(javaType); rank == 0 {
		return false
	}
	switch name.Content(source) {
	case "clone", "equals", "finalize", "getClass", "hashCode", "notify", "notifyAll", "toString", "wait":
		return false
	}
	return true
}
