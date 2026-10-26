package transpiler

import (
	"reflect"
	"testing"
)

func TestStaticImportInventory_ReturnedTuplesRemainCallerOwned(t *testing.T) {
	ctx := staticImportInventoryLegalContext(t, 0)
	first := staticMethodImports(ctx)
	if !reflect.DeepEqual(first, staticImportInventoryExpectedTuples()) {
		t.Fatal("first import query changed source tuples")
	}
	first[0].owner = "mutated.by.caller"
	second := staticMethodImports(ctx)
	if !reflect.DeepEqual(second, staticImportInventoryExpectedTuples()) {
		t.Fatal("first returned slice mutated shared source tuples")
	}
	second[0].member = "mutatedByCaller"
	if !reflect.DeepEqual(staticMethodImports(ctx), staticImportInventoryExpectedTuples()) {
		t.Fatal("cached returned slice mutated shared source tuples")
	}
	standalone := ctx.Clone()
	standalone.callableSubclasses = nil
	if !reflect.DeepEqual(staticMethodImports(standalone), staticImportInventoryExpectedTuples()) || standalone.callableSubclasses != nil {
		t.Fatal("standalone import query retained external source facts")
	}
}
