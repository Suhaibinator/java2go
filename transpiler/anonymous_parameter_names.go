package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Anonymous methods bypass ordinary source-method symbol resolution. Allocate
// legal parameter spellings here and keep the original Java names for lookup.
// Reserve identifiers throughout the method so a renamed parameter cannot
// capture another parameter or local (for example type alongside type_).
func anonymousMethodParameters(method *sitter.Node, source []byte) []*symbol.Definition {
	used := map[string]struct{}{}
	var reserve func(*sitter.Node)
	reserve = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "identifier" || node.Type() == "type_identifier" {
			used[node.Content(source)] = struct{}{}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			reserve(child)
		}
	}
	reserve(method)
	var parameters []*symbol.Definition
	for _, parameter := range nodeutil.NamedChildrenOf(method.ChildByFieldName("parameters")) {
		javaType, nameNode := nodeutil.JavaParameterNodes(parameter)
		if javaType == nil || nameNode == nil {
			continue
		}
		original := nameNode.Content(source)
		name := sanitizeGoIdent(original)
		if name != original {
			name = synchronizedUniqueLocalName(name, used)
		}
		parameters = append(parameters, &symbol.Definition{OriginalName: original, Name: name, OriginalType: javaType.Content(source)})
	}
	return parameters
}
