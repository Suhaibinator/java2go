package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// Enum source scopes omit their implicit java.lang.Enum superclass. The
// omission is not evidence that super.clone selects java.lang.Object.clone.
func TestObjectCloneRejectsEnumOwnerAndAncestor(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	file := parsing.SourceFile{Name: "EnumGuard.java", Source: []byte(`package cloneguard;
enum Plain { ONE }
enum Marked implements java.lang.Cloneable { ONE }
class Ordinary {}
class Enum {}
`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	scope := file.ParseSymbols()
	plain := scope.FindClassScope("Plain")
	marked := scope.FindClassScope("Marked")
	if plain == nil || marked == nil || !plain.IsEnum || !marked.IsEnum || plain.Superclass != "" || marked.Superclass != "" {
		t.Fatal("enum omitted-superclass precondition missing")
	}
	// A constant-specific implementation is represented as a generated source
	// subclass. Probe that metadata shape without inventing an explicit Java enum
	// subclass, which javac prohibits in written source.
	body := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "GeneratedConstantBody", Name: "GeneratedConstantBody"}, Superclass: "Marked"}
	scope.TopLevelClasses = append(scope.TopLevelClasses, body)
	symbol.AddSymbolsToPackage(scope)
	for _, test := range []struct {
		name       string
		owner      *symbol.ClassScope
		objectBody bool
	}{
		{"plain_enum", plain, false},
		{"cloneable_enum", marked, false},
		{"constant_body_ancestor", body, false},
		{"ordinary_implicit_Object", scope.FindClassScope("Ordinary"), true},
		{"source_class_named_Enum", scope.FindClassScope("Enum"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := Ctx{currentFile: scope, currentClass: test.owner}
			if got := sourceSuperSelectsObjectClone(test.owner, ctx); got != test.objectBody {
				t.Fatalf("Object.clone selected=%v, want %v", got, test.objectBody)
			}
		})
	}
}
