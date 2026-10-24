package parsing

import (
	"bytes"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestUnicodeLineTerminatorParserSource(t *testing.T) {
	cases := []struct {
		name, escape string
		exposed      bool
	}{
		{"LF", `\u000a`, true}, {"CR", `\u000d`, true}, {"CRLF", `\u000d\u000a`, true},
		{"repeated-u", `\uuuu000a`, true}, {"even-backslash", `\\u000a`, false}, {"odd-backslash", `\\\u000a`, true},
		{"four-backslash", `\\\\u000a`, false}, {"produced-backslash-then-escape", `\u005c\u000a`, true}, {"nonrecursive-produced-backslash", `\u005cu000a`, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			raw := []byte("class Before {} // 😀 comment " + test.escape + " class Exposed {}\nclass Last {}")
			file := SourceFile{Name: "LineFacts.java", Source: append([]byte(nil), raw...)}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			original := file.Source
			if file.UnicodeSource != nil {
				original = file.UnicodeSource.Original
			}
			if !bytes.Equal(original, raw) {
				t.Fatal("original source bytes changed")
			}
			if file.Ast.HasError() {
				t.Fatalf("valid lexical source retained parser errors: %s", file.Ast.String())
			}
			exposed := false
			for i := 0; i < int(file.Ast.NamedChildCount()); i++ {
				node := file.Ast.NamedChild(i)
				if node.Type() != "class_declaration" {
					continue
				}
				name := node.ChildByFieldName("name")
				if name == nil || name.Content(file.Source) != "Exposed" {
					continue
				}
				exposed = true
				start, end := node.StartByte(), node.EndByte()
				if file.UnicodeSource != nil {
					start = file.UnicodeSource.OriginalOffset(start)
					end = file.UnicodeSource.OriginalOffset(end)
				}
				if string(raw[start:end]) != "class Exposed {}" {
					t.Fatalf("source span moved: %q", raw[start:end])
				}
			}
			if exposed != test.exposed {
				t.Fatalf("Exposed declaration=%t want %t", exposed, test.exposed)
			}
		})
	}
}

func TestUnicodeLineTerminatorParenthesizedRepair(t *testing.T) {
	raw := []byte(`class Before {} // \u000a class Exposed { int value; void bump(){ (value) += 1; } }`)
	file := SourceFile{Name: "Repair.java", Source: append([]byte(nil), raw...)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(file.Source, raw) {
		t.Fatal("raw source rewritten")
	}
	if file.Ast.HasError() {
		t.Fatalf("composition retained parser error: %s", file.Ast.String())
	}
	found := false
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "assignment_expression" && strings.Contains(node.Content(file.Source), "value") {
			found = true
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}
	walk(file.Ast)
	if !found {
		t.Fatal("real exposed assignment body was not parsed")
	}
}
