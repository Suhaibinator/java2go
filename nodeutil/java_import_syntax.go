package nodeutil

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// JavaImportKind records syntax only; it does not resolve a type or member.
type JavaImportKind uint8

const (
	JavaImportSingleType JavaImportKind = iota + 1
	JavaImportTypeOnDemand
	JavaImportSingleStatic
	JavaImportStaticOnDemand
)

// JavaImportSyntax reconstructs identifier tokens separated by grammar dots.
// Scope is the written prefix and Name the terminal name (or "*"). Neither
// field establishes a declaration identity or an original source namespace.
type JavaImportSyntax struct {
	Kind  JavaImportKind
	Scope string
	Name  string
}

// ReadJavaImportSyntax proves a complete supported import declaration. Unknown,
// recovered, missing and extra non-comment syntax cannot certify an absence.
func ReadJavaImportSyntax(node *sitter.Node, source []byte) (JavaImportSyntax, bool) {
	if node == nil || node.Type() != "import_declaration" {
		return JavaImportSyntax{}, false
	}
	children, ok := javaImportSyntaxChildren(node, source)
	if !ok || len(children) < 3 || !javaImportSyntaxToken(children[0], "import", source) || !javaImportSyntaxToken(children[len(children)-1], ";", source) {
		return JavaImportSyntax{}, false
	}
	children = children[1 : len(children)-1]
	static := false
	if len(children) > 0 && javaImportSyntaxToken(children[0], "static", source) {
		static = true
		children = children[1:]
	}
	wildcard := false
	if len(children) == 3 && javaImportSyntaxToken(children[1], ".", source) && children[2].Type() == "asterisk" {
		star, valid := javaImportSyntaxChildren(children[2], source)
		if !valid || len(star) != 1 || !javaImportSyntaxToken(star[0], "*", source) {
			return JavaImportSyntax{}, false
		}
		wildcard = true
		children = children[:1]
	}
	if len(children) != 1 {
		return JavaImportSyntax{}, false
	}
	parts, ok := javaImportSyntaxIdentifiers(children[0], source)
	if !ok || len(parts) == 0 || !wildcard && len(parts) < 2 {
		return JavaImportSyntax{}, false
	}
	kind := JavaImportSingleType
	if static {
		kind = JavaImportSingleStatic
	}
	if wildcard {
		if static {
			kind = JavaImportStaticOnDemand
		} else {
			kind = JavaImportTypeOnDemand
		}
		return JavaImportSyntax{Kind: kind, Scope: strings.Join(parts, "."), Name: "*"}, true
	}
	return JavaImportSyntax{Kind: kind, Scope: strings.Join(parts[:len(parts)-1], "."), Name: parts[len(parts)-1]}, true
}

func javaImportSyntaxChildren(node *sitter.Node, source []byte) ([]*sitter.Node, bool) {
	if node == nil || node.IsMissing() || node.IsError() || node.HasError() || node.IsExtra() || node.StartByte() > node.EndByte() || uint64(node.EndByte()) > uint64(len(source)) {
		return nil, false
	}
	children := make([]*sitter.Node, 0, node.ChildCount())
	for index := 0; index < int(node.ChildCount()); index++ {
		child := node.Child(index)
		if child == nil {
			return nil, false
		}
		if !semanticJavaChild(child) {
			continue
		}
		if child == nil || child.IsMissing() || child.IsError() || child.HasError() || child.IsExtra() || child.StartByte() < node.StartByte() || child.EndByte() > node.EndByte() {
			return nil, false
		}
		children = append(children, child)
	}
	return children, true
}

func javaImportSyntaxToken(node *sitter.Node, token string, source []byte) bool {
	children, ok := javaImportSyntaxChildren(node, source)
	return ok && len(children) == 0 && !node.IsNamed() && node.Type() == token && node.Content(source) == token
}

func javaImportSyntaxIdentifiers(node *sitter.Node, source []byte) ([]string, bool) {
	children, ok := javaImportSyntaxChildren(node, source)
	if !ok {
		return nil, false
	}
	switch node.Type() {
	case "identifier":
		if len(children) != 0 || !node.IsNamed() || node.Content(source) == "" {
			return nil, false
		}
		return []string{node.Content(source)}, true
	case "scoped_identifier":
		if len(children) != 3 || !javaImportSyntaxToken(children[1], ".", source) || children[2].Type() != "identifier" {
			return nil, false
		}
		scope, name := node.ChildByFieldName("scope"), node.ChildByFieldName("name")
		if scope == nil || name == nil || !scope.Equal(children[0]) || !name.Equal(children[2]) {
			return nil, false
		}
		prefix, valid := javaImportSyntaxIdentifiers(scope, source)
		if !valid {
			return nil, false
		}
		last, valid := javaImportSyntaxIdentifiers(name, source)
		if !valid || len(last) != 1 {
			return nil, false
		}
		return append(prefix, last[0]), true
	}
	return nil, false
}
