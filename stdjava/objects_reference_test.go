package stdjava

import (
	"slices"
	"testing"
)

// These tests intentionally target canonical-only entrypoints. Native Objects
// entrypoints keep their existing Go string ABI in objects.go.
func objectsReferencePanic(t *testing.T, invoke func()) (failure any) {
	t.Helper()
	defer func() {
		failure = recover()
		if failure == nil {
			t.Fatal("expected abrupt completion")
		}
	}()
	invoke()
	return nil
}

func TestJavaObjectsReferenceIdentityAndStaticType(t *testing.T) {
	// Keep the API result representation checked by assignment.
	var one, two, three *JavaString
	value := NewJavaStringUTF16([]uint16{'v', 0xd800, 0})
	message := NewJavaStringUTF16([]uint16{'m'})
	one = ObjectsRequireNonNullReference(value)
	two = ObjectsRequireNonNullMessageReference(value, message)
	three = ObjectsRequireNonNullSupplierReference(NewExecution(), value, nil)
	if one != value || two != value || three != value {
		t.Fatal("requireNonNull changed the static result type or reference identity")
	}
	var erased any = value
	if ObjectsRequireNonNullReference[any](erased) != erased {
		t.Fatal("erased reference identity changed")
	}
}

func TestJavaObjectsReferenceNullMessagesAndCauseSlot(t *testing.T) {
	execution := NewExecution()
	units := []uint16{'M', 0, 0xd800, 'x', 0xdc00, 0xffff}
	message := NewJavaStringUTF16(units)
	emptyMessage := NewJavaStringUTF16(nil)
	for name, check := range map[string]struct {
		invoke  func()
		message *JavaString
	}{
		"untyped-null":      {func() { ObjectsRequireNonNullReference[any](nil) }, nil},
		"typed-null":        {func() { ObjectsRequireNonNullReference((*JavaString)(nil)) }, nil},
		"erased-typed-null": {func() { ObjectsRequireNonNullReference[any]((*JavaString)(nil)) }, nil},
		"String-message":    {func() { ObjectsRequireNonNullMessageReference[any](nil, message) }, message},
		"null-message":      {func() { ObjectsRequireNonNullMessageReference[any](nil, nil) }, nil},
		"empty-message":     {func() { ObjectsRequireNonNullMessageReference[any](nil, emptyMessage) }, emptyMessage},
	} {
		t.Run(name, func(t *testing.T) {
			failure := objectsReferencePanic(t, check.invoke)
			if exception, ok := failure.(NullPointerException); !ok || exception.ThrowableTypeName() != "NullPointerException" {
				t.Fatalf("panic type = %T, want canonical NullPointerException", failure)
			}
			got := JavaThrowableMessageDefault(failure)
			if got != check.message {
				t.Fatal("exception changed the exact String message reference")
			}
			if got != nil && !slices.Equal(got.UTF16Copy(), check.message.UTF16Copy()) {
				t.Fatal("exception changed UTF16 units")
			}
			cause := NewJavaExceptionMessage(nil)
			if GetCause(failure) != nil {
				t.Fatal("new NullPointerException unexpectedly has a cause")
			}
			if returned := ThrowableInitCauseExecution(execution, failure, cause); !JavaReferenceEqual(returned, failure) {
				t.Fatal("initCause did not return the original receiver")
			}
			if GetCause(failure) != cause {
				t.Fatal("canonical NullPointerException did not retain its initialized cause")
			}
		})
	}
	if !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("original UTF16 message was changed")
	}
}

type objectsReferenceSupplierProbe struct {
	execution *Execution
	message   *JavaString
	abrupt    any
	calls     int
	held      bool
}

func (*objectsReferenceSupplierProbe) Get() *JavaString {
	panic("supplier used fresh execution instead of caller execution")
}
func (p *objectsReferenceSupplierProbe) GetJava2goExecution(execution *Execution) *JavaString {
	if execution != p.execution {
		panic("supplier lost caller execution")
	}
	p.calls++
	p.held = ThreadHoldsLockExecution(execution, p)
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.message
}

func TestJavaObjectsReferenceSupplierLazyNullAndOnce(t *testing.T) {
	execution := NewExecution()
	value := NewJavaStringUTF16([]uint16{'v'})
	message := NewJavaStringUTF16([]uint16{0xd800, 0, 0xdc00})
	probe := &objectsReferenceSupplierProbe{execution: execution, message: message}
	if ObjectsRequireNonNullSupplierReference(execution, value, probe) != value || probe.calls != 0 {
		t.Fatal("present value changed identity or invoked supplier eagerly")
	}
	var typedNull *objectsReferenceSupplierProbe
	if ObjectsRequireNonNullSupplierReference(execution, value, typedNull) != value {
		t.Fatal("present value with typed-null supplier changed identity")
	}
	for name, supplier := range map[string]Supplier[*JavaString]{"nil": nil, "typed-nil": typedNull} {
		t.Run(name, func(t *testing.T) {
			failure := objectsReferencePanic(t, func() { ObjectsRequireNonNullSupplierReference[any](execution, nil, supplier) })
			if _, ok := failure.(NullPointerException); !ok || JavaThrowableMessageDefault(failure) != nil {
				t.Fatalf("null supplier panic = %T; expected null-message NullPointerException", failure)
			}
		})
	}
	func() {
		guard := MonitorEnterExecution(execution, probe)
		defer MonitorExitExecution(guard)
		failure := objectsReferencePanic(t, func() { ObjectsRequireNonNullSupplierReference[any](execution, nil, probe) })
		if JavaThrowableMessageDefault(failure) != message || probe.calls != 1 || !probe.held {
			t.Fatal("supplier changed message reference, call count, or caller monitor")
		}
		probe.message = nil
		failure = objectsReferencePanic(t, func() { ObjectsRequireNonNullSupplierReference[any](execution, nil, probe) })
		if JavaThrowableMessageDefault(failure) != nil || probe.calls != 2 || !probe.held {
			t.Fatal("supplier null result was replaced or callback repeated")
		}
	}()
	if ThreadHoldsLockExecution(execution, probe) {
		t.Fatal("caller monitor leaked after supplier completion")
	}
}

func TestJavaObjectsReferenceSupplierAbruptIdentityCauseAndCleanup(t *testing.T) {
	execution := NewExecution()
	cause := NewJavaExceptionMessage(nil)
	markerValue := NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{'b', 0xd800, 0}))
	marker := &markerValue
	if returned := ThrowableInitCauseExecution(execution, marker, cause); !JavaReferenceEqual(returned, marker) {
		t.Fatal("initCause did not retain abrupt marker receiver identity")
	}
	probe := &objectsReferenceSupplierProbe{execution: execution, abrupt: marker}
	failure := objectsReferencePanic(t, func() {
		guard := MonitorEnterExecution(execution, probe)
		defer MonitorExitExecution(guard)
		ObjectsRequireNonNullSupplierReference[any](execution, nil, probe)
	})
	if failure != marker || GetCause(failure) != cause || probe.calls != 1 || !probe.held {
		t.Fatal("supplier abrupt completion changed exception identity, cause, call count, or caller execution")
	}
	if ThreadHoldsLockExecution(execution, probe) {
		t.Fatal("caller monitor leaked after abrupt supplier completion")
	}
	other := NewExecution()
	guard := MonitorEnterExecution(other, probe)
	if !ThreadHoldsLockExecution(other, probe) {
		t.Fatal("a later execution could not own the cleaned-up monitor")
	}
	MonitorExitExecution(guard)
}

func TestJavaObjectsReferencePreservesNativeEntrypoints(t *testing.T) {
	failure := objectsReferencePanic(t, func() { ObjectsRequireNonNullMessage[any](nil, "native") })
	if failure.(NullPointerException).Message() != "native" {
		t.Fatal("legacy native String message changed")
	}
	calls := 0
	supplier := NewPlainSupplierFuncAdapter(func() string { calls++; return "native supplier" })
	failure = objectsReferencePanic(t, func() { ObjectsRequireNonNullSupplier[any](NewExecution(), nil, supplier) })
	if failure.(NullPointerException).Message() != "native supplier" || calls != 1 {
		t.Fatal("legacy native supplier entrypoint changed")
	}
}
