package stdjava

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// Keep the original measured ceiling, sample count and run count. The fixture
// is explicit nominal metadata, not a generated source object or frame probe.
const fifoAssignableAllocationCeiling = 6.0

var fifoAssignableFixtureSequence atomic.Uint64

func fifoAssignableFreshNominalOwner(t *testing.T) TypeID {
	t.Helper()
	owner := TypeID(fmt.Sprintf("campaign.assignability.fifo207.NominalOwner.%d", fifoAssignableFixtureSequence.Add(1)))
	javaTypeRegistry.RLock()
	_, existed := javaTypeRegistry.types[owner]
	javaTypeRegistry.RUnlock()
	if existed {
		t.Fatal("allocation fixture must not overwrite an existing nominal type")
	}
	RegisterJavaType(owner, "java.util.ArrayList")
	RegisterJavaSourceType(owner)
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		descriptor, present := javaTypeRegistry.types[owner]
		delete(javaTypeRegistry.types, owner)
		javaTypeRegistry.Unlock()
		if !present || descriptor.super != "java.util.ArrayList" || len(descriptor.interfaces) != 0 || !descriptor.source {
			t.Error("owned nominal allocation fixture changed before cleanup")
		}
	})
	if !JavaTypeAssignable(owner, "java.util.ArrayList") || !JavaTypeAssignable(owner, "java.util.Collection") {
		t.Fatal("nominal owner must use the actual native ArrayList hierarchy")
	}
	return owner
}

func TestJavaTypeAssignableFIFOAllocationBudget(t *testing.T) {
	owner := fifoAssignableFreshNominalOwner(t)
	for _, tc := range []struct {
		name   string
		actual TypeID
	}{
		{"native_ArrayList_negative", "java.util.ArrayList"},
		{"registered_nominal_owner_negative", owner},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for sample := 1; sample <= 3; sample++ {
				var result, unexpected bool
				count := testing.AllocsPerRun(200, func() {
					result = JavaTypeAssignable(tc.actual, ThrowableTypeID)
					unexpected = unexpected || result
				})
				if unexpected || result {
					t.Fatal("unrelated nominal type became assignable to Throwable")
				}
				t.Logf("FIFO_PUBLIC207_ALLOC|case=%s|sample=%d|runs=200|allocations_per_call=%.6f|ceiling=6.0|result=false", tc.name, sample, count)
				if count > fifoAssignableAllocationCeiling {
					t.Errorf("FIFO_PUBLIC207_ALLOCATION_CEILING|case=%s|sample=%d|got=%.6f|ceiling=6.0", tc.name, sample, count)
				}
			}
		})
	}
}
