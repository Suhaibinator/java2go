package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// A canonical RAF declaration is virtual. Only an exact native allocation or a
// lexically selected immutable TWR binding to that allocation permits this
// finite native service. Source subclasses and unknown allocation flows need a
// source carrier and are intentionally not certified by their declared type.
func randomAccessFileNativeAllocation(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	node = unwrapParenthesizedExpressionNode(node)
	proof := ctx.Clone()
	proof.expectedType, proof.expectedTypeRoot = "", nil
	if node.Type() == "object_creation_expression" {
		return objectCreationClassBody(node) == nil && randomAccessFileStringConstructorApplicable(node, nil, proof, source)
	}
	if node.Type() != "identifier" || ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentFile == nil || findFileScopeForClassScope(ctx.currentClass, ctx) != ctx.currentFile || !receiverNodeWithin(node, ctx.currentClass.Class.DeclarationNode) {
		return false
	}
	binding, found := fileChannelReadCaptureDeclaration(node, source)
	if !found || binding.declaration.Type() != "resource" {
		return false
	}
	// The lexical resolver chooses this declaration at this occurrence, including
	// capture and shadow boundaries. A declared TWR variable cannot be reassigned.
	value := unwrapParenthesizedExpressionNode(binding.declaration.ChildByFieldName("value"))
	if value == nil || value.Type() != "object_creation_expression" || objectCreationClassBody(value) != nil || !randomAccessFileStringConstructorApplicable(value, nil, proof, source) {
		return false
	}
	return fileChannelReadExternalExpression(node, "java.io.RandomAccessFile", proof, source)
}
