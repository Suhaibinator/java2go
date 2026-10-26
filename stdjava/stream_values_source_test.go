package stdjava

import "testing"

func TestStreamFromValuesOwnsOuterCopyAndRetainsReferences(t *testing.T) {
	first, last := NewObject(), NewObject()
	values := []any{first, nil, last}
	input := StreamFromValues(values...)
	values[0], values[1] = last, first
	product := NewArrayList[any]()
	got := StreamCollectPlanExecution(NewExecution(), input, CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return product })))
	if got != product || product.Size() != 3 || product.GetObject(0) != first || product.GetObject(1) != nil || product.GetObject(2) != last {
		t.Fatal("outer source copy, identity, null or encounter order changed")
	}
}

func TestStreamFromValuesFreshEmptyAndSingleConsumption(t *testing.T) {
	e := NewExecution()
	first, second := StreamFromValues[any](), StreamFromValues[any]()
	calls := 0
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { calls++; return NewArrayList[any]() }))
	if StreamCollectPlanExecution(e, first, plan).(*List[any]).Size() != 0 || StreamCollectPlanExecution(e, second, plan).(*List[any]).Size() != 0 || calls != 2 {
		t.Fatal("empty source missing initialized fresh state")
	}
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(e, first, plan) })
	if calls != 2 {
		t.Fatal("consumed source invoked supplier again")
	}
}

func TestStreamFromValuesUsesTerminalExecutionAndStopsAbruptly(t *testing.T) {
	creator, operation := NewExecution(), NewExecution()
	base := NewArrayList[any]()
	source := newArrayListSourceControl("stream.values.Source", base)
	calls, gets := 0, 0
	abrupt := NewIllegalArgumentException("values abrupt")
	source.add = func(e *Execution, value any) bool {
		if e != operation || e == creator {
			t.Fatal("source add used creation/fresh Execution")
		}
		calls++
		if calls == 2 {
			panic(abrupt)
		}
		return ArrayListBaseAddExecution(e, base, value)
	}
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	input := StreamFromValues[any](NewObject(), nil, NewObject())
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(e *Execution) any {
		if e != operation {
			t.Fatal("supplier lost terminal Execution")
		}
		gets++
		return source
	}))
	if calls != 0 || gets != 0 {
		t.Fatal("source construction eagerly ran callback")
	}
	func() {
		defer func() {
			if recover() != abrupt {
				t.Fatal("terminal replaced abrupt object")
			}
		}()
		StreamCollectPlanExecution(operation, input, plan)
		t.Fatal("abrupt terminal returned")
	}()
	if gets != 1 || calls != 2 || base.Size() != 1 {
		t.Fatal("abrupt traversal continued or supplier/product changed")
	}
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(operation, input, plan) })
}

func TestStreamFromValuesGeneralSDKScalarSource(t *testing.T) {
	input := StreamFromValues[int32](7, 11)
	result := StreamCollectPlanExecution(NewExecution(), input, CollectorToCollectionPlan[int32](NewSupplierFuncAdapter(func(*Execution) any { return NewArrayList[any]() }))).(*List[any])
	if result.Size() != 2 || result.GetObject(0) != int32(7) || result.GetObject(1) != int32(11) {
		t.Fatal("general typed SDK source lost native values")
	}
}
