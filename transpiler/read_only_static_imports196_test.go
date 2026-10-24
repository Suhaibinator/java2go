package transpiler

import (
	"reflect"
	"testing"
)

func TestReadOnlyStaticImports196BorrowAndCopyOwnership(t *testing.T) {
	ctx := campaignImportControlContext(t, campaignImportSource("alpha"))
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	want := staticMethodImportsFromFile(ctx)
	for i := 0; i < 8; i++ {
		view := readOnlyStaticMethodImports(ctx)
		if !reflect.DeepEqual(view, want) {
			t.Fatal("private iteration view changed ordered source tuples")
		}
		owned := staticMethodImports(ctx)
		owned[0].owner = "caller.poison"
		if !reflect.DeepEqual(view, want) || !reflect.DeepEqual(readOnlyStaticMethodImports(ctx), want) {
			t.Fatal("copying API mutation changed read-only source tuples")
		}
	}
	// Header replacement must invalidate the view through the existing guards.
	campaignReplaceImportTree(t, ctx, campaignImportSource("bravo"))
	if !reflect.DeepEqual(readOnlyStaticMethodImports(ctx), staticMethodImportsFromFile(ctx)) {
		t.Fatal("reparsed header retained stale borrowed tuples")
	}
	fallback := ctx
	fallback.callableSubclasses = nil
	if !reflect.DeepEqual(readOnlyStaticMethodImports(fallback), staticMethodImportsFromFile(fallback)) {
		t.Fatal("unowned context changed ordinary fallback tuples")
	}
	if readOnlyStaticMethodImports(Ctx{}) != nil {
		t.Fatal("empty context invented source tuples")
	}
}
