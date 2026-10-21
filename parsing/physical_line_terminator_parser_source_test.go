package parsing

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

type physicalTerminatorFixture struct {
	name, source, sourceSHA256 string
	valid, assignment          bool
}

var physicalTerminatorFixtures = []physicalTerminatorFixture{
	{name: "pinned-physical-CR", source: "class Probe { void run() throws Exception { // comment\rjava.nio.file.Files.readString(null); } }\n", valid: true, assignment: false, sourceSHA256: "b2d6ecf0b8958ae1ce334ebedb737207d6a458f2363f9b7b8e34fd670c421027"},
	{name: "pinned-physical-CRLF", source: "class Probe { void run() throws Exception { // comment\r\njava.nio.file.Files.readString(null); } }\n", valid: true, assignment: false, sourceSHA256: "7e9aa091517a017102a1cdfb665f7b1d09fe507c34b55bf8389b1d3915188b58"},
	{name: "CR-whitespace", source: "class Probe {\r int value;\r}\r", valid: true, assignment: false, sourceSHA256: "b92db467205928121deb78ef942da1e188c3a99c7cc079c377185811df7687c6"},
	{name: "CR-block-comment", source: "class Probe { /* first\rsecond */ int value; }", valid: true, assignment: false, sourceSHA256: "73ca92933e6117dbad68155508612b826501a90f8c1ea786bee6d5e45e2145e8"},
	{name: "CR-textblock", source: "class Probe { String text = \"\"\"\r    alpha\r    beta\r    \"\"\"; }", valid: true, assignment: false, sourceSHA256: "7c1044cb340381a6dbbe8b5639717d17b90320fa6446a5702997a2e40741e052"},
	{name: "CRLF-textblock", source: "class Probe { String text = \"\"\"\r\n    alpha\r\n    beta\r\n    \"\"\"; }", valid: true, assignment: false, sourceSHA256: "8fe7fe674774bc67002f0252b2497945156a82ee22f5cbd3420d2fdd5f4c8e2b"},
	{name: "CR-textblock-continuation", source: "class Probe { String text = \"\"\"\r    alpha\\\r    beta\r    \"\"\"; }", valid: true, assignment: false, sourceSHA256: "e20240d6e7e58427a97167d5e61960c7e99b4f98eac69a4456b7cff13766fce7"},
	{name: "eligible-escape-and-CR", source: "class Probe { void run() { // \\u000d int n=0;\r int m=1; } }", valid: true, assignment: false, sourceSHA256: "8d4cbfbfb0a3d1675336127fded39a187f902bff940363fe91f3d4e004eb14aa"},
	{name: "ineligible-escape-and-CR", source: "class Probe { void run() { // \\\\u000a Missing.call();\r int m=1; } }", valid: true, assignment: false, sourceSHA256: "96a6b372eb4518a019038a1ad3a4212c48aab1f3b58632416536779d08497e3e"},
	{name: "double-parenthesized-after-CR", source: "class Probe { int value; void run() { // comment\r ((value)) = 7; } }", valid: true, assignment: true, sourceSHA256: "d2b19711f1dc362d6fd0bde511c6bb1aeae8bee1976660533d1a8637cf48c691"},
	{name: "double-parenthesized-after-CRLF", source: "class Probe { int value; void run() { // comment\r\n ((value)) = 7; } }", valid: true, assignment: true, sourceSHA256: "cde70dc8487552dc081d73068c76bd5c23b703e43a817b2085f8ebd050842466"},
	{name: "package-CR", source: "// comment\rpackage rowprobe;\rclass Probe {}", valid: true, assignment: false, sourceSHA256: "ffb8a0f0d6a04b949da722d66696ba05fea28527d442889bd14ddb718e6fffab"},
	{name: "package-repeated-u-UTF8-CR", source: "// café \\uuuu000a package rowprobe;\rclass Probe {}", valid: true, assignment: false, sourceSHA256: "2d6b91fa7fad96509643847b0cb9014a5da64c565e46061d975bff67af8af683"},
	{name: "escaped-literal-control", source: "class Probe { String text=\"\\r\"; char value='\\r'; }", valid: true, assignment: false, sourceSHA256: "2c68fe54d8b960e667542ccd6da92d7ce98999d9ddcfc68a12091eef1988998d"},
	{name: "CR-comment-at-EOF", source: "class Probe {} // comment\r", valid: true, assignment: false, sourceSHA256: "bff9f3a49651ede046b0bc0bf7be3b0b404b9f9d1ebd796e52e4af4952e3e2ba"},
	{name: "CR-before-ineligible-literal", source: "class Probe { void run() { // comment\r String text=\"\\\\u000a\"; } }", valid: true, assignment: false, sourceSHA256: "7903395a8ee772d12d2014c87fd3620d240dd6e4c4fb6b54cf317bf66df18a04"},
	{name: "invalid-raw-CR-string", source: "class Probe { String text=\"alpha\rbeta\"; }", valid: false, assignment: false, sourceSHA256: "1aa56c7b88dc07a2c374a8baf1839be2d56599867de312332ed8dd2450466c65"},
	{name: "invalid-raw-CRLF-string", source: "class Probe { String text=\"alpha\r\nbeta\"; }", valid: false, assignment: false, sourceSHA256: "1dce291189f12831b22c6607ecfd47c3fbe1bc8854aeba13859d4d3a59fc0164"},
	{name: "invalid-raw-LF-string", source: "class Probe { String text=\"alpha\nbeta\"; }", valid: false, assignment: false, sourceSHA256: "38dab24ce2b1da0c6ac0547bfefd87c71843728a915a6dcdc84c241b4d150b73"},
	{name: "invalid-raw-CR-char", source: "class Probe { char text='\r'; }", valid: false, assignment: false, sourceSHA256: "f7d27156d7a9500609e627e0dc4a9ab6882e433bd5a59056607742eca85b1ff2"},
	{name: "invalid-unclosed-block", source: "class Probe { /* comment\r }", valid: false, assignment: false, sourceSHA256: "c25428b11833e0257fae5c45935f02cd36dd51c1802b5e72b58066e0e27dc1fe"},
	{name: "invalid-textblock-opening", source: "class Probe { String text=\"\"\"alpha\rbeta\r\"\"\"; }", valid: false, assignment: false, sourceSHA256: "f12478086226c3f474621e1e8abcc8ebfdd9b4f484b3630465af05827a04984d"},
}

func TestPhysicalLineTerminatorParseASTJDKControls(t *testing.T) {
	for _, c := range physicalTerminatorFixtures {
		t.Run(c.name, func(t *testing.T) {
			raw := []byte(c.source)
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != c.sourceSHA256 {
				t.Fatal("fixture bytes changed")
			}
			f := SourceFile{Name: "Probe.java", Source: append([]byte(nil), raw...)}
			err := f.ParseAST()
			if f.Ast == nil {
				t.Fatalf("actual AST absent, error=%v", err)
			}
			if got := err == nil && !f.Ast.HasError(); got != c.valid {
				t.Fatalf("JDK lexical validity=%v actual parser validity=%v error=%v", c.valid, got, err)
			}
			if f.UnicodeSource != nil {
				if !bytes.Equal(f.UnicodeSource.Original, raw) {
					t.Fatal("original Unicode source mutated")
				}
			} else if !bytes.Equal(f.Source, raw) {
				t.Fatal("source bytes mutated")
			}
			if !c.valid {
				return
			}
			assignments := 0
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n.EndByte() > uint32(len(f.Source)) {
					t.Fatal("source span shifted")
				}
				if n.Type() == "assignment_expression" {
					assignments++
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					walk(n.NamedChild(i))
				}
			}
			walk(f.Ast)
			if c.assignment && assignments != 1 {
				t.Fatalf("parenthesized assignment not recovered: %d", assignments)
			}
			if c.name == "pinned-physical-CR" || c.name == "pinned-physical-CRLF" {
				calls := 0
				var find func(*sitter.Node)
				find = func(n *sitter.Node) {
					if n.Type() == "method_invocation" {
						calls++
						if n.StartByte() != 55 && c.name == "pinned-physical-CR" {
							t.Fatal("original JDK call byte offset changed")
						}
						if n.Content(f.Source) != "java.nio.file.Files.readString(null)" {
							t.Fatal("original invocation source span changed")
						}
					}
					for i := 0; i < int(n.NamedChildCount()); i++ {
						find(n.NamedChild(i))
					}
				}
				find(f.Ast)
				if calls != 1 {
					t.Fatal("JDK-valid call hidden")
				}
			}
		})
	}
}

var physicalLiteralV2Fixtures = []physicalTerminatorFixture{
	{name: "opening-Unicode-tab", source: "class Probe { String text = \"\"\"\\u0009\n    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "c0a0d9f6ff70844b43b55aefa703547d5c6097b1bb2e226f59cf1aec1312f66d"},
	{name: "opening-Unicode-space-LF", source: "class Probe { String text = \"\"\"\\u0020\\u000a    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "0ab70da0f36b81f2c35c3a55271db74e63dd794a89529f07574ff21ebc905902"},
	{name: "opening-Unicode-formfeed", source: "class Probe { String text = \"\"\"\\u000c\n    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "239774e2485fe814ce5bfdb21c4bdec8718eacf1935211769f25d84382d83bca"},
	{name: "opening-repeated-u-whitespace", source: "class Probe { String text = \"\"\"\\uuuu0009\\uu000C \t\f\n    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "71c174fbfe927f6a8d330723165e964d565299e1dc24fcc8ee7e2b290edbd7c3"},
	{name: "opening-Unicode-CR", source: "class Probe { String text = \"\"\"\\u000d    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "1fa3f0719c2176ae1506f7c066f1e800c943d2e34eb531d5390d40647030842c"},
	{name: "opening-Unicode-CRLF", source: "class Probe { String text = \"\"\"\\u000d\\u000a    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "a9efbe3c19aaff93961f371fe14584f43f57fa1144201df79735f5c0601c8d16"},
	{name: "opening-raw-horizontal", source: "class Probe { String text = \"\"\" \t\f\n    alpha\n    \"\"\"; }", valid: true, sourceSHA256: "6b95e601f2758c899ef1734310deb04cb71f19de09fa9cd02628444f26b2f077"},
	{name: "valid-backslash-n", source: "class Probe { String text=\"\\n\"; char value='\\n'; }", valid: true, sourceSHA256: "94a005a0c29e71fee10331e63ad1d2528fa26d8080c13a23634ed85b172d12ee"},
	{name: "valid-ineligible-Unicode-literal", source: "class Probe { String text=\"\\\\u000a\"; }", valid: true, sourceSHA256: "946e946d31a631420a978e8420241a528fdeb5b84a3e6d378da3089e237c30c3"},
	{name: "valid-Unicode-backslash-n", source: "class Probe { String text=\"\\u005cn\"; }", valid: true, sourceSHA256: "3e7e778efeccf66d379b4f0b39de03d50c9981cd1bfeab3b9735dc7c8438cb21"},
	{name: "textblock-LF-continuation", source: "class Probe { String text = \"\"\"\n    alpha\\\n    beta\n    \"\"\"; }", valid: true, sourceSHA256: "eb76a2e3ee43f1cb5a963008eb18bc413a1d47be1dce7887da9c6835c0f8534b"},
	{name: "textblock-ineligible-Unicode-body", source: "class Probe { String text = \"\"\"\n    \\\\u000a\n    \"\"\"; }", valid: true, sourceSHA256: "a0c3cd931ccd8515aef43eaac14ada1eb3482cc12521a3fa08821deef08c3af2"},
	{name: "invalid-opening-Unicode-VT", source: "class Probe { String text = \"\"\"\\u000b\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "463e8ef74fae9fc63f370b81b902278098cd99c194f6107db2b8eddd7187fa93"},
	{name: "invalid-opening-Unicode-NBSP", source: "class Probe { String text = \"\"\"\\u00a0\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "46b3a6e994fc5cc9be2ad5919b7bce07b934904b0d4e8bdbb6a52f4cb4d153d3"},
	{name: "invalid-opening-ineligible-Unicode-tab", source: "class Probe { String text = \"\"\"\\\\u0009\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "8d5b6d0cb0a47d93efd1083ec1a7fc7c4722590a41bfba0cbaf820e18b1dd56c"},
	{name: "invalid-opening-backslash-t", source: "class Probe { String text = \"\"\"\\t\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "002af3ba1d5128d86b0935b8d5089895a9b850368eeed997561786a5db63b1f9"},
	{name: "invalid-opening-Unicode-letter", source: "class Probe { String text = \"\"\"\\u0061\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "a88fb22e4eca17a0d64c9485f57b5de533d5578236bff9ce1b506d986404503f"},
	{name: "invalid-ordinary-line-continuation", source: "class Probe { String text=\"alpha\\\nbeta\"; }", valid: false, sourceSHA256: "e98d5859f6ef6fa6d638df2a586db0058b38e19ebfcb3146ded5d3592556b59e"},
	{name: "invalid-ordinary-Unicode-LF", source: "class Probe { String text=\"\\u000a\"; }", valid: false, sourceSHA256: "25d1614969eefbb29e772692b33818f0347e7965fe3173d8e345347dc8bce359"},
	{name: "invalid-char-Unicode-CR", source: "class Probe { char text='\\u000d'; }", valid: false, sourceSHA256: "f478277f7eef279318c8170152332fd493ac21764028f5b36240ae2c221f0c58"},
	{name: "invalid-Unicode-backslash-Unicode-LF", source: "class Probe { String text=\"\\u005c\\u000a\"; }", valid: false, sourceSHA256: "a1170bb40b71154d0ca6ad3488ddaa87a82444324c038474f2a8770944be94ef"},
	{name: "invalid-textblock-opening-no-line", source: "class Probe { String text=\"\"\"   \"\"\"; }", valid: false, sourceSHA256: "143ea15d3feb5419d1f62453bdf32956bb38ec5c0e09cbec61ba92d5948ea5f3"},
	{name: "Unicode-backslash-raw-u", source: "class Probe { String text = \"\"\"\\u005cu0009\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "df3ae02bcaa386a1a5194fdd244606e842093a9b60f3b3d17bb9f54fcb309da9"},
	{name: "Unicode-backslash-Unicode-tab", source: "class Probe { String text = \"\"\"\\u005c\\u0009\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "8eab6edacc13f4704a97afbff36d1bfbd6aa1aa0ea61b3bb31dc75a6726e7577"},
	{name: "Unicode-backslash-raw-tab", source: "class Probe { String text = \"\"\"\\u005c\t\n    alpha\n    \"\"\"; }", valid: false, sourceSHA256: "d8aecaf6437bdd5840adfcaf403699bf9948a68144553273ab095b663dc92c59"},
}

func TestPhysicalLineLiteralV2JDKBaseline(t *testing.T) {
	for _, c := range physicalLiteralV2Fixtures {
		t.Run(c.name, func(t *testing.T) {
			raw := []byte(c.source)
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != c.sourceSHA256 {
				t.Fatal("fixture bytes changed")
			}
			f := SourceFile{Name: "Probe.java", Source: append([]byte(nil), raw...)}
			err := f.ParseAST()
			if f.Ast == nil {
				if c.valid || err == nil {
					t.Fatalf("missing AST/error: %v", err)
				}
				return
			}
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n.Type() == "string_literal" || n.Type() == "character_literal" {
					children := []string{}
					for i := 0; i < int(n.ChildCount()); i++ {
						children = append(children, n.Child(i).Type())
					}
					t.Logf("actual kind=%s bytes=[%d,%d) children=%q canonical=%q parserInput=%q", n.Type(), n.StartByte(), n.EndByte(), children, n.Content(f.Source), n.Content(physicalLineTerminatorParserSource(unicodeLineTerminatorParserSource(f.Source))))
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					walk(n.NamedChild(i))
				}
			}
			walk(f.Ast)
			if got := err == nil && !f.Ast.HasError(); got != c.valid {
				t.Fatalf("JDK lexical validity=%v actual parser validity=%v error=%v", c.valid, got, err)
			}
		})
	}
}

func TestPhysicalLineTerminatorPrivateViewContracts(t *testing.T) {
	for _, c := range []struct{ source, want string }{
		{"", ""}, {"plain\n", "plain\n"}, {"a\r\nb\rc\n", "a \nb\nc\n"}, {"é\r😀", "é\n😀"}, {`\u000d\u000a`, `\u000d\u000a`},
	} {
		source := []byte(c.source)
		original := append([]byte(nil), source...)
		view := physicalLineTerminatorParserSource(source)
		if len(view) != len(source) || string(view) != c.want {
			t.Fatalf("view got%q want%q", view, c.want)
		}
		if !bytes.Equal(source, original) {
			t.Fatal("view mutated source")
		}
		if bytes.IndexByte(source, '\r') >= 0 && len(view) > 0 {
			view[0] = 'x'
			if !bytes.Equal(source, original) {
				t.Fatal("mutable view aliases input")
			}
		} else if len(source) > 0 && &view[0] != &source[0] {
			t.Fatal("no-CR view unnecessarily copied")
		}
	}
	source := []byte("plain\n")
	if got := testing.AllocsPerRun(100, func() { physicalLineTerminatorParserSource(source) }); got != 0 {
		t.Fatalf("no-CR allocation=%v", got)
	}
	if physicalLineTerminatorParserSource(nil) != nil {
		t.Fatal("nil input changed")
	}
}
func TestPhysicalTextBlockOpeningPrefixContracts(t *testing.T) {
	for _, c := range []struct {
		prefix string
		valid  bool
	}{
		{" \t\f\n", true}, {"\r\n", true}, {`\u0020\u0009\u000C` + "\n", true}, {`\uuuu0009` + "\n", true},
		{`\u000b` + "\n", false}, {`\u00a0` + "\n", false}, {`\\u0009` + "\n", false}, {`\t` + "\n", false}, {`\u005cu0009` + "\n", false}, {`\u005c\u0009` + "\n", false}, {`\u005c` + "\t\n", false}, {`\u000`, false}, {"   ", false}, {"", false},
	} {
		if got := physicalTextBlockOpening([]byte(c.prefix)); got != c.valid {
			t.Fatalf("prefix%q got%v want%v", c.prefix, got, c.valid)
		}
	}
}
func TestPhysicalLineTerminatorLiteralErrorContracts(t *testing.T) {
	raw := []byte("class Probe { String text=\"alpha\nbeta\"; }")
	f := SourceFile{Name: "Probe.java", Source: append([]byte(nil), raw...)}
	err := f.ParseAST()
	if err == nil || err.Error() != "invalid Java string literal: line terminator" || f.Ast == nil {
		t.Fatalf("literal error contract: %v", err)
	}
	if !bytes.Equal(f.Source, raw) {
		t.Fatal("literal rejection mutated source")
	}
	if err := physicalLineTerminatorLiteralError(f.Ast, nil); err == nil || err.Error() != "invalid Java literal source span" {
		t.Fatalf("unknown literal span accepted: %v", err)
	}
	invalid := SourceFile{Name: "Probe.java", Source: []byte("class Probe { /* comment\r }")}
	if err := invalid.ParseAST(); err != nil || invalid.Ast == nil || !invalid.Ast.HasError() {
		t.Fatalf("existing syntax-error channel changed: %v", err)
	}
	if err := physicalLineTerminatorLiteralError(nil, nil); err != nil {
		t.Fatal("nil helper input changed")
	}
}
