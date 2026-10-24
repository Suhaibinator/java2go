package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

// An ordinary graph may have a synthesized lexical enclosing scope which is
// absent from TopLevelClasses/Subclasses and from both rendering registries.
// Its declaration name remains reachable through lexicalMemberType.
func TestGenericDemandSourceNamesUnregisteredEnclosing(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"Use.java": `class Use { String value; }`,
	})
	file := parsing.SourceFile{Name: "Synthetic.java", Source: []byte(`class String<T> {}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	target := file.Symbols.BaseClass
	owner := ctx.currentClass
	if owner == nil || target == nil {
		t.Fatal("missing witness declarations")
	}
	owner.Enclosing = target
	lexical := classScopeCtx(owner, ctx)
	if got := resolveClassScopeByQualifiedName(lexical, "String"); got != target {
		t.Fatal("ordinary lexical resolver did not reach enclosing declaration")
	}
	if names := sourceGenericDemandSourceNames(allSourceClassScopes(), ctx, nil); names != nil {
		t.Error("absence inventory admitted an open lexical enclosing graph")
	}
	got := sourceGenericViewDemandSeeds(ctx)
	if len(got) != 1 || got[0] != target {
		t.Fatal("unregistered lexical enclosing generic demand was lost")
	}
}
