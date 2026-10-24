package stdjava

import "testing"

type arrayListBasePublicOverrideHost struct {
	*List[any]
	sizeCalls, addCalls, getCalls int
}

func (host *arrayListBasePublicOverrideHost) Size() int32 {
	host.sizeCalls++
	return 137
}
func (host *arrayListBasePublicOverrideHost) Add(any) bool {
	host.addCalls++
	return false
}
func (host *arrayListBasePublicOverrideHost) Get(int32) any {
	host.getCalls++
	return "outer override"
}

func TestArrayListBaseSizeNonvirtualExistingAPI(t *testing.T) {
	e := NewExecution()
	base := NewArrayList[any]()
	base.Add(NewObject())
	host := &arrayListBasePublicOverrideHost{List: base}
	if !host.arrayListNativeStorage() || base.Size() != 1 {
		t.Fatal("invalid native-storage/size premise")
	}
	got := ArrayListBaseSizeExecution(e, host)
	if got != 1 || host.sizeCalls != 0 {
		t.Fatalf("nonvirtual base Size dispatched host override: got=%d native=1 overrideCalls=%d", got, host.sizeCalls)
	}
	if base.Size() != 1 || base.modCount != 1 {
		t.Fatal("size service mutated native storage")
	}
}

func TestArrayListBaseAddGetNonvirtualExistingAPIControls(t *testing.T) {
	e := NewExecution()
	base := NewArrayList[any]()
	host := &arrayListBasePublicOverrideHost{List: base}
	marker := NewObject()
	if !ArrayListBaseAddExecution(e, host, marker) || ArrayListBaseGetExecution(e, host, 0) != marker {
		t.Fatal("nonvirtual add/get result lost native storage")
	}
	if host.addCalls != 0 || host.getCalls != 0 || host.sizeCalls != 0 || base.Size() != 1 || base.GetObject(0) != marker || base.modCount != 1 {
		t.Fatal("base add/get dispatched public override or changed native writes")
	}
	if ArrayListBaseSizeExecution(e, base) != 1 {
		t.Fatal("plain native base size control")
	}
}

func arrayListBaseProbeFailure(call func()) (failure any) {
	defer func() { failure = recover() }()
	call()
	return nil
}

func TestArrayListBaseNonvirtualValidationExistingAPIControls(t *testing.T) {
	e := NewExecution()
	host := &arrayListBasePublicOverrideHost{List: NewArrayList[any]()}
	for _, call := range []func(){
		func() { ArrayListBaseSizeExecution(nil, host) },
		func() { ArrayListBaseGetExecution(nil, host, 0) },
		func() { ArrayListBaseAddExecution(nil, host, nil) },
	} {
		if failure := arrayListBaseProbeFailure(call); !CaughtAs(failure, "IllegalArgumentException") {
			t.Fatalf("nil Execution validation got %T", failure)
		}
	}
	for _, input := range []any{NewLinkedList[any](), NewList[any](), host.SubList(0, 0), AsList[any](nil)} {
		for _, call := range []func(){
			func() { ArrayListBaseSizeExecution(e, input) },
			func() { ArrayListBaseGetExecution(e, input, 0) },
			func() { ArrayListBaseAddExecution(e, input, nil) },
		} {
			if failure := arrayListBaseProbeFailure(call); !CaughtAs(failure, "UnsupportedOperationException") {
				t.Fatalf("invalid storage validation got %T", failure)
			}
		}
	}
	if failure := arrayListBaseProbeFailure(func() { ArrayListBaseSizeExecution(e, nil) }); !CaughtAs(failure, "NullPointerException") {
		t.Fatalf("null base validation got %T", failure)
	}
	if host.sizeCalls != 0 || host.addCalls != 0 || host.getCalls != 0 || host.List.Size() != 0 || host.modCount != 0 || host.erasedStorage {
		t.Fatal("validation reached outer behavior or mutated storage")
	}
}
