package transpiler

import (
	"bytes"
	"os"
	"sort"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type syntaxIdentifierInterval struct{ first, end int }
type syntaxIdentifierFile struct {
	anchor        *sitter.Node
	source        []byte
	bodies        map[*sitter.Node]syntaxIdentifierInterval
	raw           map[string][]int
	methods       map[int]*sitter.Node
	exactMethods  map[string][]int
	prefixMethods map[string][]int
	declarations  map[*sitter.Node]string
}

// Build only copied syntax facts. Definitions, resolved owners and allocated
// execution names are deliberately absent: those are recomputed for each query.
func buildSyntaxIdentifierFile(file *symbol.FileScope) *syntaxIdentifierFile {
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
			identifier := sanitizeGoIdent(string(file.Source[start:end]))
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
			names, known := syntaxIdentifierMethodNames(node, file.Source)
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
func syntaxIdentifierMethodNames(method *sitter.Node, source []byte) ([2]string, bool) {
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
	raw := string(source[nameStart:nameEnd])
	if raw == "" {
		return [2]string{}, false
	}
	return [2]string{sanitizeGoIdent(symbol.HandleExportStatus(false, raw)), sanitizeGoIdent(symbol.HandleExportStatus(true, raw))}, true
}

// Validate the entire live query before any method synthesis or allocator work.
// New roots and byte edits fall back rather than repeating earlier side effects.
type syntaxIdentifierBody struct {
	entry *syntaxIdentifierFile
	file  *symbol.FileScope
	root  *sitter.Node
}

func (invocation *syntaxInvocation) queryBodies(name string) ([]syntaxIdentifierBody, bool) {
	if invocation == nil || name == "" || !invocation.indexable || !invocation.active || !invocation.session.Active() || invocation.graph != symbol.GlobalScope || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return nil, false
	}
	var bodies []syntaxIdentifierBody
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			entry := invocation.identifiers[file]
			if entry == nil || entry.anchor != file.ParsedRoot() || entry.anchor.HasChanges() || !bytes.Equal(entry.source, file.Source) {
				return nil, false
			}
			for _, top := range file.TopLevelClasses {
				if top != nil && top.Class != nil && top.Class.DeclarationNode != nil {
					if _, ok := entry.bodies[top.Class.DeclarationNode]; !ok {
						return nil, false
					}
					bodies = append(bodies, syntaxIdentifierBody{entry, file, top.Class.DeclarationNode})
				}
			}
		}
	}
	return bodies, true
}

func syntaxIdentifierFirst(positions []int, interval syntaxIdentifierInterval) int {
	at := sort.SearchInts(positions, interval.first)
	if at < len(positions) && positions[at] < interval.end {
		return positions[at]
	}
	return interval.end
}

func (entry *syntaxIdentifierFile) bodyContains(name string, root *sitter.Node, source []byte) bool {
	interval := entry.bodies[root]
	raw := syntaxIdentifierFirst(entry.raw[name], interval)
	var methods []int
	add := func(positions []int) {
		for at := sort.SearchInts(positions, interval.first); at < len(positions) && positions[at] < raw; at++ {
			methods = append(methods, positions[at])
		}
	}
	add(entry.exactMethods[name])
	// Check every byte prefix, including nested prefix stems, without scanning
	// unrelated declarations. This preserves strings.HasPrefix byte semantics.
	for end := 0; end <= len(name); end++ {
		add(entry.prefixMethods[name[:end]])
	}
	sort.Ints(methods)
	previous := -1
	for _, position := range methods {
		if position == previous {
			continue
		}
		previous = position
		node := entry.methods[position]
		if superclassEmbeddingMethodNameMayMatch(node, source, name) {
			method := synthAnonClassMethodDefinition(node, source, false)
			if method != nil && (method.Name == name ||
				(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, nil) == name) ||
				(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", nil) == name)) {
				return true
			}
		}
	}
	return raw < interval.end
}
