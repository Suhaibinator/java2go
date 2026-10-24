package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func syntaxBounds214ReferenceBuild(file *symbol.FileScope) *syntaxIdentifierFile {
	root := file.ParsedRoot()
	if root == nil || root.HasError() || root.IsMissing() || root.HasChanges() {
		return nil
	}
	entry := &syntaxIdentifierFile{anchor: root, source: append([]byte(nil), file.Source...),
		bodies: map[*sitter.Node]syntaxIdentifierInterval{}, raw: map[string][]int{},
		methods: map[int]*sitter.Node{}, exactMethods: map[string][]int{}, prefixMethods: map[string][]int{},
		declarations: map[*sitter.Node]string{}}
	targets := map[*sitter.Node]bool{}
	for _, top := range file.TopLevelClasses {
		if top != nil && top.Class != nil && top.Class.DeclarationNode != nil {
			targets[top.Class.DeclarationNode] = true
		}
	}
	ordinal := 0
	var visit func(*sitter.Node) bool
	visit = func(node *sitter.Node) bool {
		if node == nil {
			return true
		}
		first := ordinal
		kind := node.Type()
		switch kind {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			entry.declarations[node] = kind
		}
		if kind == "identifier" || kind == "type_identifier" {
			start, end := node.StartByte(), node.EndByte()
			if start > end || uint64(end) > uint64(len(file.Source)) {
				return false
			}
			identifier := sanitizeGoIdent(node.Content(file.Source))
			names := [3]string{identifier, symbol.Uppercase(identifier), symbol.Lowercase(identifier)}
			for n, name := range names {
				duplicate := false
				for _, previous := range names[:n] {
					if previous == name {
						duplicate = true
						break
					}
				}
				if !duplicate {
					entry.raw[name] = append(entry.raw[name], ordinal)
				}
			}
			ordinal++
		}
		if kind == "method_declaration" {
			names, known := syntaxBounds214ReferenceMethodNames(node, file.Source)
			if !known {
				return false
			}
			entry.methods[ordinal] = node
			for n, name := range names {
				if n > 0 && names[0] == name {
					continue
				}
				entry.exactMethods[name] = append(entry.exactMethods[name], ordinal)
				for _, suffix := range [2]string{executionMethodSuffix, "Java2goExactExecution"} {
					stem := name + suffix
					entry.prefixMethods[stem] = append(entry.prefixMethods[stem], ordinal)
				}
			}
			ordinal++
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if !visit(child) {
				return false
			}
		}
		if targets[node] {
			entry.bodies[node] = syntaxIdentifierInterval{first, ordinal}
		}
		return true
	}
	if !visit(root) || len(entry.bodies) != len(targets) {
		return nil
	}
	return entry
}

// This is the same conservative header proof as the existing MayMatch guard.
// Unknown syntax disables this index; it never supplies a negative conclusion.
func syntaxBounds214ReferenceMethodNames(method *sitter.Node, source []byte) ([2]string, bool) {
	if method == nil || method.HasError() || method.IsMissing() || method.HasChanges() {
		return [2]string{}, false
	}
	nameNode := method.ChildByFieldName("name")
	typeNode := method.ChildByFieldName("type")
	if nameNode == nil || typeNode == nil || nameNode.IsMissing() || typeNode.IsMissing() || nameNode.HasError() || typeNode.HasError() || nameNode.HasChanges() || typeNode.HasChanges() || nameNode.Type() != "identifier" {
		return [2]string{}, false
	}
	start, end := method.StartByte(), method.EndByte()
	if start > end || uint64(end) > uint64(len(source)) {
		return [2]string{}, false
	}
	nameStart, nameEnd := nameNode.StartByte(), nameNode.EndByte()
	typeStart, typeEnd := typeNode.StartByte(), typeNode.EndByte()
	if nameStart < start || nameStart > nameEnd || nameEnd > end || typeStart < start || typeStart > typeEnd || typeEnd > end {
		return [2]string{}, false
	}
	raw := nameNode.Content(source)
	if raw == "" {
		return [2]string{}, false
	}
	return [2]string{sanitizeGoIdent(symbol.HandleExportStatus(false, raw)), sanitizeGoIdent(symbol.HandleExportStatus(true, raw))}, true
}
