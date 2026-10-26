package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

const erasedContractTestDescriptor = "compareTo(Ljava/lang/Object;)I"

type erasedContractTestLeaf struct {
	id     TypeID
	want   *Execution
	calls  int32
	right  any
	result int32
	abrupt any
}

func (v *erasedContractTestLeaf) JavaDynamicTypeID() TypeID { return v.id }
func (v *erasedContractTestLeaf) ExactSelected(e *Execution, right any) int32 {
	if e == nil || v.want != nil && e != v.want {
		panic("execution changed")
	}
	atomic.AddInt32(&v.calls, 1)
	v.right = right
	if v.abrupt != nil {
		panic(v.abrupt)
	}
	return v.result
}
func (v *erasedContractTestLeaf) WrongABI(e *Execution, right any) int64 { panic("wrong ABI invoked") }

type erasedContractTestProvider struct{ info *ObjectInfo }

func (v *erasedContractTestProvider) JavaObjectInfo() *ObjectInfo { return v.info }
func registerErasedContractTest(owner TypeID, selector string) {
	RegisterJavaType(owner, ObjectTypeID, ComparableTypeID)
	RegisterSourceErasedContract(owner, ComparableTypeID, erasedContractTestDescriptor, selector)
}
func erasedContractTestPanic(t *testing.T, invoke func()) any {
	t.Helper()
	var recovered any
	func() { defer func() { recovered = recover() }(); invoke() }()
	if recovered == nil {
		t.Fatal("expected abrupt completion")
	}
	return recovered
}
func TestNativeContractExactLeafNullExecutionAndAbruptIdentity(t *testing.T) {
	id := TypeID("test.contract.leaf")
	registerErasedContractTest(id, "ExactSelected")
	execution := NewExecution()
	leaf := &erasedContractTestLeaf{id: id, want: execution, result: 17}
	right := &erasedContractHostRightA{}
	if result, ok := compareViaCompareToExecution(execution, leaf, right); !ok || result != 17 || leaf.calls != 1 || leaf.right != right {
		t.Fatal("exact leaf invocation changed")
	}
	var typedNull *erasedContractHostRightA
	if _, ok := compareViaCompareToExecution(execution, leaf, typedNull); !ok || leaf.right != any(typedNull) {
		t.Fatal("typed null representation changed")
	}
	if _, ok := compareViaCompareToExecution(execution, leaf, nil); !ok || leaf.right != nil {
		t.Fatal("null not passed")
	}
	leaf.want = nil
	if _, ok := compareViaCompareToExecution(nil, leaf, right); !ok {
		t.Fatal("nil execution did not create invocation context")
	}
	abrupt := &struct{ code int }{9}
	leaf.abrupt = abrupt
	if got := erasedContractTestPanic(t, func() { compareViaCompareToExecution(execution, leaf, right) }); got != abrupt {
		t.Fatal("body abrupt identity changed")
	}
}
func TestNativeContractOwnerProjectionNearestClassAndReentry(t *testing.T) {
	base := TypeID("test.contract.base")
	child := TypeID("test.contract.child")
	iface := TypeID("test.contract.interface")
	registerErasedContractTest(base, "ExactSelected")
	RegisterJavaType(child, base, ComparableTypeID, iface)
	RegisterJavaType(iface, ObjectTypeID, ComparableTypeID)
	RegisterSourceErasedContract(iface, ComparableTypeID, erasedContractTestDescriptor, "WrongABI")
	execution := NewExecution()
	baseView := &erasedContractTestLeaf{id: child, want: execution, result: 31}
	childView := &erasedContractTestLeaf{id: child, want: execution, result: 47}
	views := 0
	info := NewObjectInfo(child, func(owner TypeID) any {
		views++
		RegisterSourceErasedContract(base, ComparableTypeID, erasedContractTestDescriptor, "ExactSelected")
		if owner == base {
			return baseView
		}
		if owner == child {
			return childView
		}
		return nil
	})
	alias := &erasedContractTestProvider{info: info}
	if n, ok := compareViaCompareToExecution(execution, alias, nil); !ok || n != 31 || views != 1 {
		t.Fatalf("ancestor view: %d %t %d", n, ok, views)
	}
	RegisterSourceErasedContract(child, ComparableTypeID, erasedContractTestDescriptor, "ExactSelected")
	if n, ok := compareViaCompareToExecution(execution, alias, nil); !ok || n != 47 || views != 2 {
		t.Fatalf("same dynamic owner must project: %d %t %d", n, ok, views)
	}
	if baseView.calls != 1 || childView.calls != 1 {
		t.Fatal("wrong owner or duplicate body")
	}
}
func TestNativeContractPublicationAndInvariantFailures(t *testing.T) {
	id := TypeID("test.contract.late")
	RegisterJavaType(id, ObjectTypeID, ComparableTypeID)
	leaf := &erasedContractTestLeaf{id: id, result: 19}
	if _, ok := compareViaCompareToExecution(nil, leaf, nil); ok {
		t.Fatal("unregistered arbitrary selector discovered")
	}
	RegisterSourceErasedContract(id, ComparableTypeID, erasedContractTestDescriptor, "ExactSelected")
	RegisterSourceErasedContract(id, ComparableTypeID, erasedContractTestDescriptor, "ExactSelected")
	if n, ok := compareViaCompareToExecution(nil, leaf, nil); !ok || n != 19 {
		t.Fatal("late registration invisible")
	}
	erasedContractTestPanic(t, func() { RegisterSourceErasedContract(id, ComparableTypeID, erasedContractTestDescriptor, "WrongABI") })
	if n, ok := compareViaCompareToExecution(nil, leaf, nil); !ok || n != 19 {
		t.Fatal("conflict changed prior registration")
	}
	bad := TypeID("test.contract.badabi")
	registerErasedContractTest(bad, "WrongABI")
	badLeaf := &erasedContractTestLeaf{id: bad}
	if _, ok := erasedContractTestPanic(t, func() { compareViaCompareToExecution(nil, badLeaf, nil) }).(sourceErasedContractInvariantError); !ok {
		t.Fatal("bad ABI must be typed invariant failure")
	}
	if badLeaf.calls != 0 {
		t.Fatal("bad ABI invoked fallback")
	}
	missing := TypeID("test.contract.missingview")
	RegisterJavaType(missing, id, ComparableTypeID)
	plain := &erasedContractTestLeaf{id: missing}
	if _, ok := erasedContractTestPanic(t, func() { compareViaCompareToExecution(nil, plain, nil) }).(sourceErasedContractInvariantError); !ok {
		t.Fatal("ancestor without provider must fail invariant")
	}
	noView := &erasedContractTestProvider{info: NewObjectInfo(id, func(TypeID) any { return nil })}
	if _, ok := erasedContractTestPanic(t, func() { compareViaCompareToExecution(nil, noView, nil) }).(sourceErasedContractInvariantError); !ok {
		t.Fatal("missing provider view must fail invariant")
	}
	RegisterJavaType(id, ObjectTypeID)
	if _, ok := erasedContractTestPanic(t, func() { compareViaCompareToExecution(nil, leaf, nil) }).(ClassCastException); !ok {
		t.Fatal("live nominal inconsistency must be Java ClassCastException")
	}
}
func TestNativeContractMissCycleAndConcurrentPublication(t *testing.T) {
	a := TypeID("test.contract.cycle.a")
	b := TypeID("test.contract.cycle.b")
	RegisterJavaType(a, b)
	RegisterJavaType(b, a)
	views := int32(0)
	miss := &erasedContractTestProvider{info: NewObjectInfo(a, func(TypeID) any { atomic.AddInt32(&views, 1); return nil })}
	if _, ok := compareViaCompareToExecution(nil, miss, nil); ok || views != 0 {
		t.Fatal("miss projected receiver or selected interface")
	}
	id := TypeID("test.contract.concurrent")
	registerErasedContractTest(id, "ExactSelected")
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			execution := NewExecution()
			leaf := &erasedContractTestLeaf{id: id, want: execution, result: 23}
			for j := 0; j < 20; j++ {
				RegisterSourceErasedContract(id, ComparableTypeID, erasedContractTestDescriptor, "ExactSelected")
				if n, ok := compareViaCompareToExecution(execution, leaf, nil); !ok || n != 23 {
					panic("concurrent lookup changed")
				}
			}
			if leaf.calls != 20 {
				panic("body count")
			}
		}()
	}
	group.Wait()
}
