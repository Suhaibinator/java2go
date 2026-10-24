package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"testing"
)

func TestGenericDemandSourceNamesUnregisteredEnclosingLegacy(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"Use.java": `class Use{String value;}`})
	file := parsing.SourceFile{Name: "Synthetic.java", Source: []byte(`class String<T>{}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	target := file.Symbols.BaseClass
	owner := ctx.currentClass
	owner.Enclosing = target
	ctx.localClasses = map[string]*localClassInfo{"force-legacy": nil}
	got := sourceGenericViewDemandSeeds(ctx)
	if len(got) != 1 || got[0] != target {
		t.Fatal("legacy full-fallback lost enclosing seed")
	}
}
