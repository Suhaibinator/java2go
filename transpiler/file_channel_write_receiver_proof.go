package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type fileChannelWriteReceiverProofKind uint8

const (
	fileChannelWriteReceiverUnavailable fileChannelWriteReceiverProofKind = iota
	fileChannelWriteReceiverNullable
	fileChannelWriteReceiverSourceResource
)

// A completed allocation certificate is distinct from nullable invocation
// provenance. In particular, an empty description never admits a receiver.
type fileChannelWriteReceiverProof struct {
	kind        fileChannelWriteReceiverProofKind
	description string
	resource    *sitter.Node
	allocation  *symbol.ClassScope
}

func fileChannelWriteReceiverProofFor(node *sitter.Node, ctx Ctx, source []byte) (fileChannelWriteReceiverProof, bool) {
	if description, known := fileChannelWriteReceiverDescription(node, ctx, source); known && description != "" {
		return fileChannelWriteReceiverProof{kind: fileChannelWriteReceiverNullable, description: description}, true
	}
	return fileChannelWriteSourceResourceProof(node, ctx, source)
}

func fileChannelWriteSourceResourceProof(node *sitter.Node, ctx Ctx, source []byte) (fileChannelWriteReceiverProof, bool) {
	refuse := fileChannelWriteReceiverProof{}
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || node.Type() != "identifier" || ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || ctx.currentFile == nil {
		return refuse, false
	}
	file := findFileScopeForClassScope(ctx.currentClass, ctx)
	method := ctx.localScope.DeclarationNode
	if file == nil || file != ctx.currentFile || (method.Type() != "method_declaration" && method.Type() != "constructor_declaration") {
		return refuse, false
	}
	// Require both the occurrence and method to belong to these actual trees.
	// Same byte ranges in a detached scope or another source file are insufficient.
	owned := false
	for n := method.Parent(); n != nil; n = n.Parent() {
		if ctx.currentClass.Class.DeclarationNode != nil && n.Equal(ctx.currentClass.Class.DeclarationNode) {
			owned = true
			break
		}
		switch n.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "object_creation_expression":
			return refuse, false
		}
	}
	if !owned {
		return refuse, false
	}
	binding, found := fileChannelReadCaptureDeclaration(node, source)
	if !found || binding.declaration == nil || binding.declaration.Type() != "resource" {
		return refuse, false
	}
	var resource *sitter.Node
	insideMethod := false
	for n := node.Parent(); n != nil; n = n.Parent() {
		if n.Equal(method) {
			insideMethod = true
			break
		}
		switch n.Type() {
		case "lambda_expression", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "object_creation_expression", "method_declaration", "constructor_declaration":
			return refuse, false
		case "try_with_resources_statement":
			resources := n.ChildByFieldName("resources")
			if receiverNodeWithin(node, resources) {
				return refuse, false
			}
			if receiverNodeWithin(node, n.ChildByFieldName("body")) {
				for _, candidate := range nodeutil.SemanticNamedChildrenOf(resources) {
					if candidate != nil && candidate.Equal(binding.declaration) {
						resource = candidate
					}
				}
			}
		}
	}
	if !insideMethod || resource == nil {
		return refuse, false
	}
	typ := resource.ChildByFieldName("type")
	if typ == nil || isVarKeywordType(typ.Content(source)) {
		return refuse, false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(typ.Content(source)))
	_, arguments := parseJavaTypeString(base)
	if rank != 0 || len(arguments) != 0 {
		return refuse, false
	}
	if owner, known := fileChannelDeclarationOwner(typ.Content(source), ctx); !known || owner != "java.nio.channels.FileChannel" {
		return refuse, false
	}
	allocation, known := fileChannelWriteAllocationViewProof(resource.ChildByFieldName("value"), ctx, source)
	if !known {
		return refuse, false
	}
	// Declaration resources are implicitly final in Java. Do not grant malformed
	// parser inputs a certificate after a write to this particular binding.
	mutated := false
	var inspect func(*sitter.Node)
	inspect = func(n *sitter.Node) {
		if n == nil || mutated {
			return
		}
		var target *sitter.Node
		if n.Type() == "assignment_expression" {
			target = n.ChildByFieldName("left")
		} else if n.Type() == "update_expression" && n.NamedChildCount() > 0 {
			target = n.NamedChild(0)
		}
		if target != nil && target.Type() == "identifier" {
			if actual, found := fileChannelReadCaptureDeclaration(target, source); found && actual.declaration != nil && actual.declaration.Equal(resource) {
				mutated = true
				return
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
			inspect(child)
		}
	}
	inspect(method.ChildByFieldName("body"))
	if mutated {
		return refuse, false
	}
	return fileChannelWriteReceiverProof{kind: fileChannelWriteReceiverSourceResource, resource: resource, allocation: allocation}, true
}

func fileChannelWriteAllocationViewProof(creation *sitter.Node, ctx Ctx, source []byte) (*symbol.ClassScope, bool) {
	creation = unwrapParenthesizedExpressionNode(creation)
	if creation == nil || creation.Type() != "object_creation_expression" || objectCreationClassBody(creation) != nil {
		return nil, false
	}
	inference := ctx.Clone()
	inference.expectedType, inference.expectedTypeRoot = "", nil
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(creation, inference, source, &origin)
	if !known || !completeJavaTypeOrigin(origin) || origin.parameter != nil || origin.nominalScope == nil || origin.arrayRank != 0 || origin.wildcard {
		return nil, false
	}
	typ := creation.ChildByFieldName("type")
	if typ == nil {
		return nil, false
	}
	_, arguments := parseJavaTypeString(typ.Content(source))
	if len(arguments) != 0 {
		return nil, false
	}
	scope := resolveObjectCreationTargetScope(creation, typ, typ.Content(source), source, inference)
	if scope == nil || scope != origin.nominalScope || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.IsInterface || scope.IsAbstract || scope.IsEnum || scope.IsInner || len(scope.TypeParameters) != 0 {
		return nil, false
	}
	declaration := scope.Class.DeclarationNode
	parent := declaration.Parent()
	if declaration.Type() != "class_declaration" || parent == nil || findFileScopeForClassScope(scope, ctx) == nil {
		return nil, false
	}
	if parent.Type() != "program" && (scope.Enclosing == nil || (parent.Type() != "class_body" && parent.Type() != "interface_body") || (!declarationHasModifier(declaration, "static") && !scope.Enclosing.IsInterface)) {
		return nil, false
	}
	// These unchanged generators install the direct root's storage and reserve
	// its canonical-view selector. Successful construction returns that same
	// source allocation; a thrown constructor never enters the resource body.
	root := fileChannelSourceRoot(scope, inference)
	if root == nil || !fileChannelSourceDirectRoot(root, inference) || !fileChannelSourceReservedSelector(root, fileChannelSourceViewMethod, inference) {
		return nil, false
	}
	return scope, true
}
