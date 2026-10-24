package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

func syntax175OldSourceIdentifierExists(name string, ctx Ctx) bool {
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			var visit func(*sitter.Node) bool
			visit = func(node *sitter.Node) bool {
				if node == nil {
					return false
				}
				kind := node.Type()
				if kind == "identifier" || kind == "type_identifier" {
					identifier := sanitizeGoIdent(node.Content(file.Source))
					if identifier == name || symbol.Uppercase(identifier) == name || symbol.Lowercase(identifier) == name {
						return true
					}
				}
				if kind == "method_declaration" && superclassEmbeddingMethodNameMayMatch(node, file.Source, name) {
					method := synthAnonClassMethodDefinition(node, file.Source, false)
					if method != nil && (method.Name == name ||
						(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, nil) == name) ||
						(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", nil) == name)) {
						return true
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					if visit(child) {
						return true
					}
				}
				return false
			}
			for _, top := range file.TopLevelClasses {
				if top != nil && top.Class != nil && visit(top.Class.DeclarationNode) {
					return true
				}
			}
		}
	}
	return false
}
