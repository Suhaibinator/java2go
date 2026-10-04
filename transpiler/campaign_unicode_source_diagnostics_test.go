package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestCampaignUnicodeSourceDiagnosticMapping(t *testing.T) {
	original := "class Example { /* π😀 */\r\n String value=\"\\uu0041😀\";\r\n}"
	file := parsing.SourceFile{Name: "Example.java", Source: []byte(original)}
	if err := file.ParseAST(); err != nil || file.Ast.HasError() {
		t.Fatalf("parse: %v", err)
	}
	scope := file.ParseSymbols()
	var literal *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "string_literal" {
			literal = node
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(file.Ast)
	if literal == nil || file.UnicodeSource == nil {
		t.Fatal("missing normalized literal/source map")
	}
	want := `"\uu0041😀"`
	start := file.UnicodeSource.OriginalOffset(literal.StartByte())
	end := file.UnicodeSource.OriginalOffset(literal.EndByte())
	if start != uint32(strings.Index(original, want)) || original[start:end] != want {
		t.Fatalf("incorrect original literal byte span: [%d,%d)", start, end)
	}
	diagnostic := reportUnsupported("test", literal, file.Source, Ctx{currentFile: scope, suppressUnsupportedDiagnostics: true})
	if diagnostic.Line != 2 || diagnostic.Message != want {
		t.Fatalf("diagnostic does not preserve original source: %#v", diagnostic)
	}
}
