package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestGenericFamilyCacheRenderLifetimeAndSyntheticIsolation(t *testing.T) {
	helper := setupParseHelper(t, `abstract class Adapter<T>{abstract T read();} interface Factory{<S> Adapter<S> create();} class Use{Adapter<String> make(){return new Adapter<String>(){String read(){return "ok";}};}}`)
	seed := helper.File.Symbols.FindClassScope("Adapter")
	ctx := helper.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{}
	plan := canonicalGenericFamily(seed, ctx)
	if plan == nil {
		t.Fatal("missing eligible named plan")
	}
	if again := canonicalGenericFamily(seed, ctx.Clone()); again != plan {
		t.Fatal("clone recomputed immutable named inventory")
	}
	if a, b := canonicalGenericFamily(seed, helper.Ctx), canonicalGenericFamily(seed, helper.Ctx); a == nil || b == nil || a == b {
		t.Fatal("uncached callers unexpectedly shared a plan")
	}
	var anonymous genericFamilyAnonymous
	for _, entry := range plan.anonymous {
		anonymous = entry
		break
	}
	if anonymous.node == nil {
		t.Fatal("anonymous source inventory missing")
	}
	first := &symbol.ClassScope{Class: &symbol.Definition{DeclarationNode: anonymous.node}, Enclosing: anonymous.owner}
	second := &symbol.ClassScope{Class: &symbol.Definition{DeclarationNode: anonymous.node}, Enclosing: anonymous.owner}
	firstPlan := canonicalGenericFamily(first, ctx)
	secondPlan := canonicalGenericFamily(second, ctx)
	if firstPlan == nil || secondPlan == nil {
		t.Fatal("anonymous source membership missing")
	}
	if _, ok := plan.members[first]; ok {
		t.Fatal("synthetic scope mutated cached named membership")
	}
	if _, ok := secondPlan.members[first]; ok {
		t.Fatal("one anonymous lowering leaked into another")
	}
	if _, ok := firstPlan.members[first]; !ok {
		t.Fatal("synthetic instance not included in its own view")
	}

	// A new resolution/conversion boundary creates a fresh cache. Unsupported
	// later source declarations cannot inherit an earlier eligibility result.
	file := parsing.SourceFile{Name: "Bad.java", Source: []byte(`abstract class Bad extends Adapter<java.util.List<String>>{}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	ResolveFile(file)
	fresh := helper.Ctx.Clone()
	fresh.genericFamilies = &genericFamilyAnalysis{}
	if canonicalGenericFamily(seed, fresh) != nil {
		t.Fatal("fresh render reused stale eligibility after resolution")
	}
}
