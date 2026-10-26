package transpiler

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// anonymousExternalInterfaceOwner proves the allocation's Java TYPE in its
// executable lexical context. Unlike a named class header, that context includes
// method and captured binders; their source spellings can own the first prefix.
func anonymousExternalInterfaceOwner(node, objectType *sitter.Node, source []byte, ctx Ctx) string {
	if node == nil || node.Type() != "object_creation_expression" || objectType == nil || ctx.currentFile == nil || ctx.currentClass == nil {
		return ""
	}
	actual := node.ChildByFieldName("type")
	if actual == nil || actual.StartByte() != objectType.StartByte() || actual.EndByte() != objectType.EndByte() {
		return ""
	}
	written := actual.Content(source)
	base, _, valid := reflectiveNamedTypeParts(strings.TrimSpace(written))
	if !valid || base == "" {
		return ""
	}
	first := strings.SplitN(base, ".", 2)[0]
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		sourceName := parameter.Name
		if parameter.Declaration != nil {
			sourceName = parameter.Declaration.SourceName
		}
		if first == sourceName {
			return ""
		}
	}
	return declarationOwnedExternalInterface(written, ctx.currentClass, ctx)
}
