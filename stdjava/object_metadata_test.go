package stdjava

import (
	"reflect"
	"runtime"
	"testing"
)

func objectMetadataPanic(call func()) (caught any) {
	defer func() { caught = recover() }()
	call()
	return nil
}

func TestCanonicalObjectAllocationMetadataAndIdentity(t *testing.T) {
	t.Run("actualObjectDescriptor", func(t *testing.T) {
		value := NewObject()
		id, known := ObjectDynamicType(value)
		if !known || id != ObjectTypeID {
			t.Fatalf("actual NewObject allocation metadata = %q/%v, want java.lang.Object/true", id, known)
		}
		if ObjectGetClass(value) != ClassLiteral(ObjectTypeID) {
			t.Fatal("canonical Object allocation has wrong Class identity")
		}
		if id, known := ObjectDynamicType(new(byte)); known || id != "" {
			t.Fatal("unrelated host pointer was universally reinterpreted as Object")
		}
	})
	t.Run("freshLivePointerEqualityAndHash", func(t *testing.T) {
		var values []any
		seen := make(map[uintptr]bool)
		for range 64 {
			value := NewObject()
			pointer := reflect.ValueOf(value)
			if pointer.Kind() != reflect.Pointer || pointer.Type().Elem().Size() == 0 || pointer.IsNil() {
				t.Fatal("Object allocation lost its nonzero-sized pointer representation")
			}
			if address := pointer.Pointer(); address == 0 || seen[address] {
				t.Fatal("distinct live Object allocations share pointer identity")
			} else {
				seen[address] = true
			}
			values = append(values, value)
		}
		first, second := values[0], values[1]
		alias := first
		execution := NewExecution()
		if !JavaReferenceEqual(first, alias) || JavaReferenceEqual(first, second) ||
			!ObjectEqualsExecution(execution, first, alias) || ObjectEqualsExecution(execution, first, second) {
			t.Fatal("Object alias or distinct-allocation equality changed")
		}
		hash := ObjectHashCodeExecution(execution, first)
		if hash != ObjectHashCodeExecution(execution, alias) || hash != SystemIdentityHashCode(first) ||
			SystemIdentityHashCode(alias) != SystemIdentityHashCode(first) || SystemIdentityHashCode(nil) != 0 {
			t.Fatal("Object default/System identity hash lost stability or alias identity")
		}
		runtime.KeepAlive(values)
	})
	t.Run("reentrantAndDistinctMonitorIdentity", func(t *testing.T) {
		first, second := NewObject(), NewObject()
		execution := NewExecution()
		if ThreadHoldsLockExecution(execution, first) {
			t.Fatal("new Object monitor already held")
		}
		outer := MonitorEnterExecution(execution, first)
		if !ThreadHoldsLockExecution(execution, first) || ThreadHoldsLockExecution(execution, second) {
			t.Fatal("different Object allocations share a monitor")
		}
		inner := MonitorEnterExecution(execution, first)
		MonitorExitExecution(inner)
		if !ThreadHoldsLockExecution(execution, first) {
			t.Fatal("reentrant inner exit released outer monitor")
		}
		MonitorExitExecution(outer)
		if ThreadHoldsLockExecution(execution, first) {
			t.Fatal("outer monitor was not released")
		}
	})
	t.Run("nominalViewsAndTypedNull", func(t *testing.T) {
		value := NewObject()
		if ObjectView[any](value, ObjectTypeID) != value {
			t.Fatal("Object view changed exact allocation reference")
		}
		failure := objectMetadataPanic(func() { ObjectView[*StringBuilder](value, TypeID("java.lang.StringBuilder")) })
		if !CaughtAs(failure, "ClassCastException") {
			t.Fatalf("Object-to-Builder nominal rejection = %T", failure)
		}
		null := reflect.Zero(reflect.TypeOf(value)).Interface()
		if id, known := ObjectDynamicType(null); known || id != "" {
			t.Fatal("typed null advertised Object allocation metadata")
		}
		if carrier, ok := null.(JavaDynamicTypeCarrier); ok && carrier.JavaDynamicTypeID() != "" {
			t.Fatal("nil metadata carrier advertised a live Object allocation")
		}
		if ObjectView[any](null, ObjectTypeID) != nil || SystemIdentityHashCode(null) != 0 {
			t.Fatal("typed null view/hash contract changed")
		}
		for name, call := range map[string]func(){
			"getClass": func() { ObjectGetClass(null) },
			"monitor":  func() { MonitorEnterExecution(NewExecution(), null) },
		} {
			if failure := objectMetadataPanic(call); !CaughtAs(failure, "NullPointerException") {
				t.Errorf("typed-null %s rejection = %T", name, failure)
			}
		}
	})
}
