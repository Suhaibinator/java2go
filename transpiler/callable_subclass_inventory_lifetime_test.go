package transpiler

import (
	"strings"
	"testing"
)

func TestCallableSubclassInventory_CachedQueriesKeepSourceIdentityAndLifetime(t *testing.T) {
	source := `class Base<T>{} class Child extends Base<String>{} class Other{} class Use{Base<String> make(){return new Child(){};} void local(){class Local extends Child{}}} class Padding{void noise(){int value=0;` + strings.Repeat("value++;", 2048) + `}}`
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	base, child, other := helper.File.Symbols.FindClassScope("Base"), helper.File.Symbols.FindClassScope("Child"), helper.File.Symbols.FindClassScope("Other")
	if !classHasUnmodeledCallableSubclass(base, ctx) || !classHasUnmodeledCallableSubclass(child, ctx) || classHasUnmodeledCallableSubclass(other, ctx) || classHasUnmodeledCallableSubclass(nil, ctx) {
		t.Fatal("cached source events changed anonymous/local/ancestor queries")
	}
	// Named-family discovery clears its own cache while analyzing plans. Its
	// cloned context must retain independent structural source facts.
	clone := ctx.Clone()
	clone.genericFamilies = nil
	allocations := testing.AllocsPerRun(1, func() {
		if classHasUnmodeledCallableSubclass(other, clone) {
			t.Fatal("unrelated declaration became a superclass")
		}
	})
	if allocations > 256 {
		t.Fatalf("clone repeated source traversal: %.0f allocations >256", allocations)
	}
	if !classHasUnmodeledCallableSubclass(base, clone) || !classHasUnmodeledCallableSubclass(child, clone) {
		t.Fatal("clone lost source superclass identities")
	}
	fresh := setupParseHelper(t, `class Base<T>{} class Child extends Base<String>{} class Other{} class Use{static class Named extends Child{}}`)
	freshCtx := fresh.Ctx.Clone()
	freshCtx.callableSubclasses = ctx.callableSubclasses
	freshBase, freshChild := fresh.File.Symbols.FindClassScope("Base"), fresh.File.Symbols.FindClassScope("Child")
	if freshBase == base || freshChild == child {
		t.Fatal("separate source graph reused declaration identity")
	}
	if classHasUnmodeledCallableSubclass(freshBase, freshCtx) || classHasUnmodeledCallableSubclass(freshChild, freshCtx) {
		t.Fatal("new source graph inherited anonymous/local superclass events")
	}
	// Ctx{} callers remain independent of a render's structural cache.
	if classHasUnmodeledCallableSubclass(freshBase, fresh.Ctx) {
		t.Fatal("uncached named-only graph changed")
	}
}
