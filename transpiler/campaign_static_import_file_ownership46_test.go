package transpiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func TestStaticMethodImportsUsesFileDeclaration46(t *testing.T) {
	parse := func(name, imports string) parsing.SourceFile {
		source := "package probe;\n" + imports
		// Both source buffers are equally long and large enough for either tree.
		// Incorrect AST ownership must fail on import identity, not just bounds.
		source += strings.Repeat(" ", 256-len(source)) + "class " + name + " {}"
		file := parsing.SourceFile{Name: name + ".java", Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		return file
	}
	left := parse("Left", "import static alpha.Owner.read;\nimport static alpha.Owner.more;\n")
	right := parse("Rght", "import static beta.Owner.*;\n")
	if len(left.Source) != len(right.Source) {
		t.Fatal("guard source lengths differ")
	}
	for _, fixture := range []struct {
		name  string
		file  parsing.SourceFile
		other parsing.SourceFile
		want  []staticMethodImport
	}{
		{"explicit", left, right, []staticMethodImport{{owner: "alpha.Owner", member: "read"}, {owner: "alpha.Owner", member: "more"}}},
		{"wildcard", right, left, []staticMethodImport{{owner: "beta.Owner", wildcard: true}}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			ctx := Ctx{currentFile: fixture.file.Symbols, currentClass: fixture.other.Symbols.BaseClass}
			if got := staticMethodImports(ctx); !reflect.DeepEqual(got, fixture.want) {
				t.Fatalf("file imports = %#v; want %#v", got, fixture.want)
			}
		})
	}
}
