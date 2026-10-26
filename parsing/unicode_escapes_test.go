package parsing

import "testing"

func TestCanonicalJavaUnicodeEscapeEligibility(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{`"\uu0041"`, `"\u0041"`},
		{`"\\uu0041"`, `"\\uu0041"`},
		{`"\\\uu0041"`, `"\\\u0041"`},
		{`"\u005c\uu0041"`, `"\u005c\u0041"`},
		{`"\u005c\\uu0041"`, `"\u005c\\u0041"`},
		{`"\uuD835\uuuDFD9"`, `"\uD835\uDFD9"`},
		{`"\uu00xz"`, `"\uu00xz"`},
	} {
		t.Run(test.source, func(t *testing.T) {
			got, mapping := canonicalJavaUnicodeEscapes([]byte(test.source))
			if string(got) != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
			if mapping != nil {
				for index, value := range got {
					if mapping.Original[mapping.OriginalOffset(uint32(index))] != value {
						t.Fatalf("incorrect original byte mapping at %d", index)
					}
				}
				if mapping.OriginalOffset(uint32(len(got))) != uint32(len(test.source)) {
					t.Fatal("incorrect EOF mapping")
				}
			}
		})
	}
}

func TestRepeatedUnicodeEscapeParseSourceMap(t *testing.T) {
	original := []byte("class Example { /* π😀 */\r\n String s = \"\\uuD835\\uuuDFD9\";\r\n}")
	file := SourceFile{Name: "Example.java", Source: original}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	if file.Ast.HasError() || file.UnicodeSource == nil {
		t.Fatal("eligible repeated-u literal did not parse with a source map")
	}
	if string(file.UnicodeSource.Original) != string(original) {
		t.Fatal("original source was changed")
	}
	mapping := file.UnicodeSource
	if err := file.ParseAST(); err != nil || file.UnicodeSource != mapping {
		t.Fatal("reparsing discarded the original source mapping")
	}
}
