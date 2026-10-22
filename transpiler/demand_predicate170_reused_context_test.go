package transpiler

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDemandPredicate170ReusedContextMutation(t *testing.T) {
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java":  `package p; public class Cell<T>{T value;}`,
		"q/Cell.java":  `package q; public class Cell<T>{T value;}`,
		"app/Use.java": `package app; import p.Cell; class Use{<Cell> Cell read(){return null;}}`,
	}, true)
	p, q := findQualifiedSourceClass("p.Cell"), findQualifiedSourceClass("q.Cell")
	use := findQualifiedSourceClass("app.Use")
	if p == nil || q == nil || use == nil {
		t.Fatal("missing live mutation declarations")
	}
	file := findFileScopeForClassScope(use, f.ctx)
	var method *symbol.Definition
	for _, candidate := range use.Methods {
		if candidate.OriginalName == "read" {
			method = candidate
		}
	}
	if method == nil || len(method.TypeParameters) != 1 {
		t.Fatal("missing parsed method binder")
	}
	ctx := f.ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	check := func(current demandPredicate170Fixture, want *symbol.ClassScope) {
		t.Helper()
		before := demandPredicate170Snapshot(current)
		// Keep exactly the same inventory pointer across both functions and all
		// later graph edits. This supplements the isolated cold-oracle cases.
		old := demandPredicate170Capture(demandPredicate170OldSeeds, ctx, false)
		got := demandPredicate170Capture(sourceGenericViewDemandSeeds, ctx, false)
		if !reflect.DeepEqual(old, got) || !reflect.DeepEqual(before, demandPredicate170Snapshot(current)) {
			t.Fatal("warm context oracle or input ownership changed")
		}
		if want == nil {
			if len(got.seeds) != 0 {
				t.Fatal("live binder shadow was ignored")
			}
		} else if len(got.seeds) != 1 || got.seeds[0] != want {
			t.Fatalf("live seed identity %#v instead of %p", got.seeds, want)
		}
	}
	check(f, nil)
	parameters := method.TypeParameters
	method.TypeParameters = nil
	check(f, p)
	file.Imports["Cell"] = "q"
	check(f, q)
	file.Imports["Cell"] = "p"
	check(f, p)
	method.TypeParameters = parameters
	check(f, nil)
	oldGraph := symbol.GlobalScope
	g := demandPredicate170Setup(t, map[string]string{
		"late/Late.java": `package late; class Late<T>{T value;} class Use{Late<?> field;}`,
	}, true)
	late := findQualifiedSourceClass("late.Late")
	if late == nil {
		t.Fatal("new graph declaration missing")
	}
	check(g, late)
	symbol.GlobalScope = oldGraph
	check(f, nil)
	t.Log("REUSED_CONTEXT170 binder/import/global mutations observed with unchanged AST owners")
}
