package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestExecutionNamingPreservesRenderFamilyCache(t *testing.T) {
	helper := setupParseHelper(t, `abstract class Adapter<T>{abstract T read();} interface Factory{<S> Adapter<S> create();} class Use{Adapter<String> make(){return new Adapter<String>(){String read(){return "ok";}};}}`)
	owner := overrideBridgeTestScope(t, helper, "Adapter")
	method := overrideBridgeTestMethod(t, owner, "read")
	ctx := helper.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{}
	name := executionImplementationName(method, owner, ctx)
	if !ctx.genericFamilies.ready {
		t.Fatal("execution naming discarded the render-context family cache")
	}
	plan := canonicalGenericFamily(owner, ctx)
	if plan == nil {
		t.Fatal("execution naming failed to retain eligible source-family inventory")
	}
	clone := ctx.Clone()
	if got := executionImplementationName(method, owner, clone); got != name {
		t.Fatalf("clone changed execution selector: %q != %q", got, name)
	}
	if got := canonicalGenericFamily(owner, clone); got != plan {
		t.Fatal("cloned render context recomputed the source-family inventory")
	}
	fresh := helper.Ctx.Clone()
	fresh.genericFamilies = &genericFamilyAnalysis{}
	if got := executionImplementationName(method, owner, fresh); got != name {
		t.Fatalf("fresh context changed selector: %q != %q", got, name)
	}
	if !fresh.genericFamilies.ready || canonicalGenericFamily(owner, fresh) == plan {
		t.Fatal("fresh render context shared an earlier inventory")
	}
	if got := executionImplementationName(method, owner, helper.Ctx); got != name {
		t.Fatalf("uncached naming semantics changed: %q != %q", got, name)
	}

	// Resolving additional source changes eligibility. A new conversion context
	// must observe that change; an earlier immutable render inventory stays local.
	file := parsing.SourceFile{Name: "Bad.java", Source: []byte(`abstract class Bad extends Adapter<java.util.List<String>>{}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	ResolveFile(file)
	later := helper.Ctx.Clone()
	later.genericFamilies = &genericFamilyAnalysis{}
	_ = executionImplementationName(method, owner, later)
	if !later.genericFamilies.ready {
		t.Fatal("fresh resolution naming bypassed its cache")
	}
	if canonicalGenericFamily(owner, later) != nil {
		t.Fatal("later resolution reused stale source-family eligibility")
	}
	if canonicalGenericFamily(owner, ctx) != plan {
		t.Fatal("later resolution mutated an earlier render inventory")
	}
}

func TestExecutionNamingCachedAndUncachedBridgeSelectorsAgree(t *testing.T) {
	helper := setupParseHelper(t, `abstract class Adapter<T>{abstract T read();abstract void write(T value);} interface Factory{<S> Adapter<S> create();} final class Specific extends Adapter<String>{String read(){return "ok";} void write(String value){}} class Plain{int value(){return 1;} int valueJava2goExecution(){return 2;}}`)
	ctx := helper.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{}
	for _, item := range []struct{ owner, method string }{{"Adapter", "read"}, {"Adapter", "write"}, {"Specific", "read"}, {"Specific", "write"}, {"Plain", "value"}} {
		owner := overrideBridgeTestScope(t, helper, item.owner)
		method := overrideBridgeTestMethod(t, owner, item.method)
		expected := executionImplementationName(method, owner, helper.Ctx)
		got := executionImplementationName(method, owner, ctx)
		if got == "" || got != expected {
			t.Fatalf("%s.%s selector cached=%q uncached=%q", item.owner, item.method, got, expected)
		}
		if item.owner == "Plain" && got == "ValueJava2goExecution" {
			t.Fatal("execution selector captured a source method")
		}
	}
	if !ctx.genericFamilies.ready {
		t.Fatal("name-selection paths did not retain render cache")
	}
	owner := overrideBridgeTestScope(t, helper, "Adapter")
	method := overrideBridgeTestMethod(t, owner, "read")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(owner, method, ctx)
	if !ok || plan == nil {
		t.Fatal("missing eligible erased family")
	}
	if got, want := directOwnerOverrideBridgeErasedExecutionName(plan, ctx), executionImplementationName(method, owner, ctx); got != want {
		t.Fatalf("erased bridge selector=%q want %q", got, want)
	}
}
