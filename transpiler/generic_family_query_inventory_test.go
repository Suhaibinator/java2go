package transpiler

import (
	"bytes"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenericFamilyDiscoveryOwnsStructuralInventory(t *testing.T) {
	helper := setupParseHelper(t, `class Cell<T>{} class Factory{<T>Cell<T> make(){return null;}}`)
	foreign := &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	outer := &callableSubclassSourceInventory{graph: foreign, ready: true, syntheticReady: true, selectorsReady: true, selectors: map[string]struct{}{"sentinel": {}}}
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = outer
	for sample := 0; sample < 2; sample++ {
		if plans := discoverCanonicalGenericFamilies(ctx); len(plans) != 0 {
			t.Fatal("leaf family admission changed")
		}
		if ctx.callableSubclasses != outer || outer.graph != foreign || !outer.ready || !outer.syntheticReady || !outer.selectorsReady || len(outer.selectors) != 1 || outer.nodes != nil || outer.ownership != nil || len(outer.events) != 0 || len(outer.syntheticEvents) != 0 {
			t.Fatal("discovery borrowed or mutated caller structural inventory")
		}
	}
}

func TestGenericFamilyCallableEventsKeepMatchesFresh(t *testing.T) {
	helper := setupParseHelper(t, `class Left{} class Rght{} class Use{Object make(){return new Left(){};}}`)
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{graph: symbol.GlobalScope}
	left, right := helper.File.Symbols.FindClassScope("Left"), helper.File.Symbols.FindClassScope("Rght")
	if !classHasUnmodeledCallableSubclass(left, ctx) || classHasUnmodeledCallableSubclass(right, ctx) {
		t.Fatal("initial source target mismatch")
	}
	index := bytes.Index(helper.File.Symbols.Source, []byte("new Left()"))
	if index < 0 {
		t.Fatal("missing anonymous source")
	}
	copy(helper.File.Symbols.Source[index:index+len("new Left()")], []byte("new Rght()"))
	if classHasUnmodeledCallableSubclass(left, ctx) || !classHasUnmodeledCallableSubclass(right, ctx) {
		t.Fatal("cached structure retained target admission after source change")
	}
}
