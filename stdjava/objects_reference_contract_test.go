package stdjava

import (
	"slices"
	"testing"
)

// These additive contracts come from the frozen actual JDK21 Objects oracle:
// seed 17's message has NUL and separated high/low surrogate code units.
// Existing focused tests and Java oracle sources remain unchanged.
func objectsReferenceContractPanic(t *testing.T, invoke func()) (failure any) {
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

func objectsReferenceContractNPE(t *testing.T, execution *Execution, failure any, message *JavaString, units []uint16) {
	t.Helper()
	exception, ok := failure.(NullPointerException)
	if !ok || exception.ThrowableTypeName() != "NullPointerException" {
		t.Fatalf("panic type = %T, want exact canonical NullPointerException", failure)
	}
	if got := JavaThrowableMessageDefault(failure); got != message {
		t.Fatal("NullPointerException changed the exact message reference")
	} else if got != nil && !slices.Equal(got.UTF16Copy(), units) {
		t.Fatalf("message UTF16 = %x, want %x", got.UTF16Copy(), units)
	}
	if GetCause(failure) != nil {
		t.Fatal("new NullPointerException unexpectedly has a cause")
	}
	causeValue := NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{'s'}))
	cause := &causeValue
	returned := ThrowableInitCauseExecution(execution, failure, cause)
	if !JavaReferenceEqual(returned, failure) {
		t.Fatal("initCause did not return the original NullPointerException reference")
	}
	if GetCause(failure) != cause {
		t.Fatal("initCause did not retain the exact cause reference")
	}
}

func TestJavaObjectsReferenceContractInitCauseReturnsReceiver(t *testing.T) {
	units := []uint16{114, 0, 104, 55296, 120, 56320}
	message := NewJavaStringUTF16(units)
	empty := NewJavaStringUTF16(nil)
	execution := NewExecution()
	cases := []struct {
		name    string
		invoke  func()
		message *JavaString
		units   []uint16
	}{
		{"plain", func() { ObjectsRequireNonNullReference[any](nil) }, nil, nil},
		{"null-message", func() { ObjectsRequireNonNullMessageReference[any](nil, nil) }, nil, nil},
		{"empty-message", func() { ObjectsRequireNonNullMessageReference[any](nil, empty) }, empty, nil},
		{"UTF16-message", func() { ObjectsRequireNonNullMessageReference[any](nil, message) }, message, units},
		{"null-supplier", func() { ObjectsRequireNonNullSupplierReference[any](execution, nil, nil) }, nil, nil},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			failure := objectsReferenceContractPanic(t, test.invoke)
			objectsReferenceContractNPE(t, execution, failure, test.message, test.units)
		})
	}
	if !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("original UTF16 message changed")
	}
}

type objectsReferenceContractSupplier struct {
	execution  *Execution
	caller     *Thread
	message    *JavaString
	trace      *[]string
	calls      int
	held       bool
	sameThread bool
}

func (*objectsReferenceContractSupplier) Get() *JavaString {
	panic("supplier lost caller execution")
}

func (supplier *objectsReferenceContractSupplier) GetJava2goExecution(execution *Execution) *JavaString {
	if execution != supplier.execution {
		panic("supplier changed caller execution")
	}
	supplier.calls++
	supplier.held = ThreadHoldsLockExecution(execution, supplier)
	supplier.sameThread = ThreadCurrentThread(execution) == supplier.caller
	*supplier.trace = append(*supplier.trace, "get")
	return supplier.message
}

func TestJavaObjectsReferenceContractSupplierNPEUnwindsCallerMonitor(t *testing.T) {
	units := []uint16{114, 0, 104, 55296, 120, 56320}
	for _, name := range []string{"UTF16-message", "null-message"} {
		t.Run(name, func(t *testing.T) {
			execution := NewExecution()
			var message *JavaString
			if name == "UTF16-message" {
				message = NewJavaStringUTF16(units)
			}
			trace := []string{}
			supplier := &objectsReferenceContractSupplier{
				execution: execution,
				caller:    ThreadCurrentThread(execution),
				message:   message,
				trace:     &trace,
			}
			// Recovery is outside the entire monitor scope, so the NPE must
			// unwind through MonitorExit before the catch observation.
			failure := objectsReferenceContractPanic(t, func() {
				guard := MonitorEnterExecution(execution, supplier)
				defer MonitorExitExecution(guard)
				trace = append(trace, "enter")
				ObjectsRequireNonNullSupplierReference[any](execution, nil, supplier)
				trace = append(trace, "after")
			})
			if ThreadHoldsLockExecution(execution, supplier) {
				t.Fatal("caller monitor remained held when the NPE reached catch")
			}
			trace = append(trace, "catch")
			objectsReferenceContractNPE(t, execution, failure, message, units)
			if supplier.calls != 1 || !supplier.held || !supplier.sameThread {
				t.Fatalf("supplier calls=%d held=%v sameThread=%v", supplier.calls, supplier.held, supplier.sameThread)
			}
			func() {
				guard := MonitorEnterExecution(execution, supplier)
				defer MonitorExitExecution(guard)
				if !ThreadHoldsLockExecution(execution, supplier) {
					t.Fatal("caller could not reacquire the released monitor")
				}
				trace = append(trace, "reacquire")
			}()
			if ThreadHoldsLockExecution(execution, supplier) {
				t.Fatal("reacquired monitor leaked after leaving its scope")
			}
			if want := []string{"enter", "get", "catch", "reacquire"}; !slices.Equal(trace, want) {
				t.Fatalf("supplier cleanup trace = %v, want %v", trace, want)
			}
		})
	}
}
