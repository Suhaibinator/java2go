package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// simpleReceiverHasLexicalValueBinding distinguishes a value receiver from a
// same-named source type at this expression. Method-wide local inventories do
// not retain declaration order or block boundaries, so inspect lexical scopes.
func simpleReceiverHasLexicalValueBinding(ctx Ctx, source []byte, node *sitter.Node, name string) bool {
	if node == nil || name == "" {
		return false
	}
	if ctx.localScope != nil && ctx.localScope.ParameterByName(name) != nil {
		return true
	}
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		if _, duplicate := seen[scope]; duplicate {
			return true
		}
		seen[scope] = struct{}{}
		if findFieldResolutionInHierarchy(scope, name, ctx) != nil {
			return true
		}
	}
	position := node.StartByte()
	for child, parent := node, node.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "block", "constructor_body":
			for _, sibling := range nodeutil.NamedChildrenOf(parent) {
				if sibling.StartByte() >= child.StartByte() {
					break
				}
				if localDeclarationNames(sibling, name, source) {
					return true
				}
			}
		case "local_variable_declaration":
			if localDeclarationNamesBefore(parent, name, source, position) {
				return true
			}
		case "for_statement":
			if localDeclarationNamesBefore(parent.ChildByFieldName("init"), name, source, position) {
				return true
			}
		case "enhanced_for_statement":
			if receiverNodeWithin(node, parent.ChildByFieldName("body")) && receiverDeclarationName(parent.ChildByFieldName("name"), name, source) {
				return true
			}
		case "lambda_expression":
			if receiverNodeWithin(node, parent.ChildByFieldName("body")) && receiverParameterNames(parent.ChildByFieldName("parameters"), name, source) {
				return true
			}
		case "catch_clause":
			if receiverNodeWithin(node, parent.ChildByFieldName("body")) {
				for _, parameter := range nodeutil.NamedChildrenOf(parent) {
					if parameter.Type() == "catch_formal_parameter" && receiverDeclarationName(parameter.ChildByFieldName("name"), name, source) {
						return true
					}
				}
			}
		case "try_with_resources_statement":
			resources := parent.ChildByFieldName("resources")
			if receiverNodeWithin(node, resources) || receiverNodeWithin(node, parent.ChildByFieldName("body")) {
				for _, resource := range nodeutil.NamedChildrenOf(resources) {
					ident := resource.ChildByFieldName("name")
					if receiverDeclarationName(ident, name, source) && ident.EndByte() <= position {
						return true
					}
				}
			}
		case "method_declaration", "constructor_declaration":
			if receiverParameterNames(parent.ChildByFieldName("parameters"), name, source) {
				return true
			}
		}
	}
	return false
}

func receiverNodeWithin(node, scope *sitter.Node) bool {
	return node != nil && scope != nil && node.StartByte() >= scope.StartByte() && node.EndByte() <= scope.EndByte()
}

func receiverDeclarationName(ident *sitter.Node, name string, source []byte) bool {
	return ident != nil && ident.Content(source) == name
}

func receiverParameterNames(parameters *sitter.Node, name string, source []byte) bool {
	if parameters == nil {
		return false
	}
	if parameters.Type() == "identifier" {
		return receiverDeclarationName(parameters, name, source)
	}
	for _, parameter := range nodeutil.NamedChildrenOf(parameters) {
		if parameter.Type() == "identifier" && receiverDeclarationName(parameter, name, source) || receiverDeclarationName(parameter.ChildByFieldName("name"), name, source) {
			return true
		}
	}
	return false
}

func localDeclarationNamesBefore(declaration *sitter.Node, name string, source []byte, position uint32) bool {
	if declaration == nil || declaration.Type() != "local_variable_declaration" {
		return false
	}
	for _, declarator := range nodeutil.NamedChildrenOf(declaration) {
		if declarator.Type() == "variable_declarator" {
			ident := declarator.ChildByFieldName("name")
			if receiverDeclarationName(ident, name, source) && ident.EndByte() <= position {
				return true
			}
		}
	}
	return false
}
