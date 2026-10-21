package transpiler

import (
	"crypto/sha256"
	"fmt"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

type physicalRowFixture struct {
	name, source string
	line         uint32
	module       bool
	sourceSHA256 string
}

var physicalRowFixtures = []physicalRowFixture{
	{name: "escape-LF", source: "class Probe { void run() throws Exception { // \\u000a java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "d61da223e267ae4aeea200d0ced373b55accf75eb35271ed5fb275c942a4ed8b"},
	{name: "escape-CR", source: "class Probe { void run() throws Exception { // \\u000d java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "e7300213ce9389f529943c170807a4ff3ce073a2324f7ce7acc50cf6f78b13c6"},
	{name: "escape-CRLF", source: "class Probe { void run() throws Exception { // \\u000d\\u000a java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "6bae4e94d49ce3f71876b0ba8232e097b646317c6c5ddb837ba39496db93888b"},
	{name: "escape-repeated-u", source: "class Probe { void run() throws Exception { // \\uuuu000a java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "37e85a34626534fa87c3b7cf730b7873d98eb85ceba1896fde31d104ddaef795"},
	{name: "escape-UTF8", source: "class Probe { void run() throws Exception { // 😀 \\u000a java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "fb56ecc61046d5d56eb776c592f0b8bae077054a7cf42a3e2321b76a7f665788"},
	{name: "physical-LF", source: "class Probe { void run() throws Exception { // comment\njava.nio.file.Files.readString(null); } }\n", line: 2, module: false, sourceSHA256: "76e92c8ea616974afba51fb1301c40f98d8a7a093976f0255bc20d6f624af8c7"},
	{name: "physical-CRLF", source: "class Probe { void run() throws Exception { // comment\r\njava.nio.file.Files.readString(null); } }\n", line: 2, module: false, sourceSHA256: "7e9aa091517a017102a1cdfb665f7b1d09fe507c34b55bf8389b1d3915188b58"},
	{name: "physical-CR", source: "class Probe { void run() throws Exception { // comment\rjava.nio.file.Files.readString(null); } }\n", line: 2, module: false, sourceSHA256: "b2d6ecf0b8958ae1ce334ebedb737207d6a458f2363f9b7b8e34fd670c421027"},
	{name: "ordinary-noescape", source: "class Probe { void run() throws Exception { java.nio.file.Files.readString(null); } }\n", line: 1, module: false, sourceSHA256: "7e67b0818271baf1424234fe0027989799e752aeeee16c95e8ee0c1764c0070b"},
	{name: "module-escaped-LF", source: "// \\u000a module rowprobe {}\n", line: 1, module: true, sourceSHA256: "a9086d680885ce5aeeeda27114b32cfeafeb63ca58959923ce624c2206bb8d02"},
}

func physicalRowParse(t *testing.T, source string) parsing.SourceFile {
	t.Helper()
	f := parsing.SourceFile{Name: "Probe.java", Source: []byte(source)}
	if err := f.ParseAST(); err != nil {
		t.Fatal(err)
	}
	if f.Ast == nil || f.Ast.HasError() {
		t.Fatalf("actual Java parser rejected fixture: %s", source)
	}
	return f
}
func physicalRowInvocation(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "method_invocation" {
		return node
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if got := physicalRowInvocation(node.NamedChild(i)); got != nil {
			return got
		}
	}
	return nil
}
func physicalRowReport(t *testing.T, f parsing.SourceFile, current *symbol.FileScope) Diagnostic {
	t.Helper()
	n := physicalRowInvocation(f.Ast)
	if n == nil {
		t.Fatal("actual method invocation absent")
	}
	resetDiagnostics()
	setStrictMode(false)
	t.Cleanup(func() { resetDiagnostics(); setStrictMode(false) })
	return reportUnsupported("expression", n, f.Source, Ctx{className: "Probe", currentFile: current})
}
func TestPhysicalSourceRowsActualConsumers(t *testing.T) {
	for _, c := range physicalRowFixtures {
		t.Run(c.name, func(t *testing.T) {
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(c.source))); got != c.sourceSHA256 {
				t.Fatalf("native fixture bytes changed: got %s want %s", got, c.sourceSHA256)
			}
			f := physicalRowParse(t, c.source)
			if c.module {
				_, err := classifyCompilationUnit(f)
				want := fmt.Sprintf("unsupported compilation unit %q in Probe.java at line %d", "module_declaration", c.line)
				if err == nil || err.Error() != want {
					t.Fatalf("physical module refusal: got %v want %s", err, want)
				}
				return
			}
			d := physicalRowReport(t, f, f.ParseSymbols())
			if d.Line != c.line {
				t.Fatalf("physical original row: got %d want %d, diagnostic=%s", d.Line, c.line, d)
			}
			if d.Kind != "expression" || d.NodeType != "method_invocation" || d.ClassName != "Probe" || d.Message != "java.nio.file.Files.readString(null)" {
				t.Fatalf("diagnostic contract changed: %+v", d)
			}
			want := fmt.Sprintf("line %d: in Probe: unsupported expression %q: java.nio.file.Files.readString(null)", c.line, "method_invocation")
			if d.String() != want {
				t.Fatalf("diagnostic string changed: %s", d)
			}
		})
	}
}
func TestPhysicalSourceRowsParserAndOffsetShields(t *testing.T) {
	fixtures := []struct {
		name, source string
		line         uint32
	}{
		{"physical-before-escape", "class Probe { void run() throws Exception {\r\n// \\u000a java.nio.file.Files.readString(null); } }", 2},
		{"multiple-escape", "class Probe { void run() throws Exception { // \\u000a \\u000a java.nio.file.Files.readString(null); } }", 1},
		{"repeated-u-UTF8-before-physical", "class Probe { void run() throws Exception { // café \\uuuu000a\r\njava.nio.file.Files.readString(null); } }", 2},
		{"parenthesized-assignment", "class Probe { void run() throws Exception { int v=0; // \\u000a ((v)) = java.nio.file.Files.readString(null).length(); } }", 1},
	}
	for _, c := range fixtures {
		t.Run(c.name, func(t *testing.T) {
			f := physicalRowParse(t, c.source)
			n := physicalRowInvocation(f.Ast)
			if n == nil {
				t.Fatal("call absent")
			}
			d := physicalRowReport(t, f, f.ParseSymbols())
			if d.Line != c.line {
				t.Fatalf("got row%d want%d", d.Line, c.line)
			}
			original := []byte(c.source)
			if f.UnicodeSource != nil && string(f.UnicodeSource.Original) != c.source {
				t.Fatal("original source mutated")
			}
			if f.UnicodeSource == nil && string(f.Source) != string(original) {
				t.Fatal("no-map source mutated")
			}
		})
	}
	// An ineligible escape keeps the following token in the physical comment.
	f := physicalRowParse(t, "class Probe { void run() { // \\\\u000a Missing.call();\n} }")
	if physicalRowInvocation(f.Ast) != nil {
		t.Fatal("even backslashes exposed a comment token")
	}
	for _, c := range physicalRowFixtures {
		if c.module {
			continue
		}
		f := physicalRowParse(t, c.source)
		for label, current := range map[string]*symbol.FileScope{"nil": nil, "missing-map": {OriginalSource: []byte("unused")}, "invalid-original-offset": {OriginalSource: []byte("x"), SourceOffsets: make([]uint32, len(f.Source)+1)}} {
			if label == "invalid-original-offset" {
				for i := range current.SourceOffsets {
					current.SourceOffsets[i] = 99
				}
			}
			d := physicalRowReport(t, f, current)
			if d.Line != c.line {
				t.Fatalf("%s/%s fallback row%d want%d", c.name, label, d.Line, c.line)
			}
		}
	}
	resetDiagnostics()
	setStrictMode(false)
	d := reportUnsupported("node", nil, nil, Ctx{})
	if d.Line != 0 || d.NodeType != "" || d.Message != "" {
		t.Fatal("nil node changed")
	}
}
func TestPhysicalSourceRowsStrictRefusalUnchanged(t *testing.T) {
	f := physicalRowParse(t, physicalRowFixtures[0].source)
	n := physicalRowInvocation(f.Ast)
	resetDiagnostics()
	setStrictMode(true)
	defer func() { resetDiagnostics(); setStrictMode(false) }()
	defer func() {
		r := recover()
		e, ok := r.(strictModeError)
		if !ok {
			t.Fatalf("strict failure changed: %T", r)
		}
		if e.diagnostic.Line != 1 || e.diagnostic.Kind != "expression" || e.diagnostic.Message != "java.nio.file.Files.readString(null)" {
			t.Fatalf("strict diagnostic changed: %+v", e.diagnostic)
		}
	}()
	reportUnsupported("expression", n, f.Source, Ctx{className: "Probe", currentFile: f.ParseSymbols()})
	t.Fatal("strict construct admitted")
}

func TestPhysicalSourceLineBoundsAndTerminators(t *testing.T) {
	cases := []struct {
		name, source string
		rows         []uint32
	}{
		{"empty", "", []uint32{1}},
		{"LF", "a\nb", []uint32{1, 1, 2, 2}},
		{"CR", "a\rb", []uint32{1, 1, 2, 2}},
		{"CRLF", "a\r\nb", []uint32{1, 1, 1, 2, 2}},
		{"mixed", "\r\n\r\n\n\r", []uint32{1, 1, 2, 2, 3, 4, 5}},
		{"escape-spelling", `\u000a`, []uint32{1, 1, 1, 1, 1, 1, 1}},
		{"UTF8", "é\n", []uint32{1, 1, 1, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for offset, want := range c.rows {
				if got := physicalSourceLine([]byte(c.source), uint32(offset)); got != want {
					t.Fatalf("offset%d row%d want%d", offset, got, want)
				}
			}
			for _, bad := range []uint32{uint32(len(c.source) + 1), ^uint32(0)} {
				if got := physicalSourceLine([]byte(c.source), bad); got != 0 {
					t.Fatalf("invalid offset%d invented row%d", bad, got)
				}
			}
		})
	}
	if physicalSourceLine(nil, 0) != 1 || physicalSourceLine(nil, 1) != 0 {
		t.Fatal("nil byte input bounds changed")
	}
}

func TestPhysicalSourceRowsModuleOriginalOffsets(t *testing.T) {
	for _, prefix := range []string{"", "\r\n"} {
		raw := prefix + "// café \\uuuu000a module rowprobe {}"
		f := physicalRowParse(t, raw)
		if f.UnicodeSource == nil || string(f.UnicodeSource.Original) != raw {
			t.Fatal("actual repeated-u original mapping absent")
		}
		_, err := classifyCompilationUnit(f)
		want := fmt.Sprintf("unsupported compilation unit %q in Probe.java at line %d", "module_declaration", 1+len(prefix)/2)
		if err == nil || err.Error() != want {
			t.Fatalf("mapped module refusal got%v want%s", err, want)
		}
		f.Source = nil
		_, err = classifyCompilationUnit(f)
		if err == nil || err.Error() != "unsupported compilation unit \"module_declaration\" in Probe.java at line 0" {
			t.Fatalf("bad canonical offset invented row: %v", err)
		}
	}
}
