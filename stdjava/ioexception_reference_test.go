package stdjava

import (
	"slices"
	"testing"
)

func ioExceptionReferencePanic(t *testing.T, invoke func()) (failure any) {
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

func TestJavaIOExceptionReferenceMessageAndCauseSlots(t *testing.T) {
	execution := NewExecution()
	units := []uint16{114, 0, 0xd800, 'x', 0xdc00}
	message := NewJavaStringUTF16(units)
	cause := NewJavaExceptionMessage(nil)
	for _, test := range []struct {
		name      string
		failure   IOException
		message   *JavaString
		cause     any
		available bool
	}{
		{"empty", NewJavaIOExceptionMessage(nil), nil, nil, true},
		{"string", NewJavaIOExceptionMessage(message), message, nil, true},
		{"typed-null-string", NewJavaIOExceptionMessage((*JavaString)(nil)), nil, nil, true},
		{"both", NewJavaIOExceptionMessageCause(message, cause), message, cause, false},
		{"null-message-cause", NewJavaIOExceptionMessageCause(nil, cause), nil, cause, false},
		{"message-null-cause", NewJavaIOExceptionMessageCause(message, nil), message, nil, false},
		{"both-null", NewJavaIOExceptionMessageCause(nil, nil), nil, nil, false},
		{"typed-null-cause", NewJavaIOExceptionCauseExecution(execution, (*Exception)(nil)), nil, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.failure.ThrowableTypeName() != "IOException" || JavaThrowableMessageDefault(test.failure) != test.message || GetCause(test.failure) != test.cause {
				t.Fatal("IOException changed exact message/cause references or overload identity")
			}
			if test.message != nil && !slices.Equal(JavaThrowableMessageDefault(test.failure).UTF16Copy(), units) {
				t.Fatal("IOException changed NUL/isolated-surrogate UTF16 message units")
			}
			replacement := NewJavaRuntimeExceptionMessage(nil)
			if test.available {
				returned := ThrowableInitCauseExecution(execution, test.failure, replacement)
				if !JavaReferenceEqual(returned, test.failure) || GetCause(test.failure) != replacement {
					t.Fatal("String constructor did not leave initCause available or preserve returned/cause identity")
				}
			} else {
				failure := ioExceptionReferencePanic(t, func() { ThrowableInitCauseExecution(execution, test.failure, replacement) })
				if exception, ok := failure.(Throwable); !ok || exception.ThrowableTypeName() != "IllegalStateException" || GetCause(test.failure) != test.cause {
					t.Fatal("explicit cause overload failed to close the cause slot")
				}
			}
		})
	}
	if !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("original immutable message changed")
	}
	if got := NewIOException("native").Message(); got != "native" {
		t.Fatal("native Go IOException constructor changed")
	}
}

type ioExceptionReferenceCause struct {
	Exception
	execution *Execution
	caller    *Thread
	message   *JavaString
	abrupt    any
	trace     *[]string
	calls     int
	held      bool
	same      bool
}

func (*ioExceptionReferenceCause) JavaDynamicTypeID() TypeID { return "path.ioexception.Cause" }
func (*ioExceptionReferenceCause) String() string            { panic("native cause rendering") }
func (cause *ioExceptionReferenceCause) DeclaredText(execution *Execution) *JavaString {
	if execution != cause.execution {
		panic("IOException lost caller execution")
	}
	cause.calls++
	cause.held = ThreadHoldsLockExecution(execution, cause)
	cause.same = ThreadCurrentThread(execution) == cause.caller
	*cause.trace = append(*cause.trace, "render")
	if cause.abrupt != nil {
		panic(cause.abrupt)
	}
	return cause.message
}

func ioExceptionRegisterReferenceCause() {
	id := TypeID("path.ioexception.Cause")
	RegisterJavaType(id, BuiltinThrowableTypeID("Exception"))
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredText")
}

func TestJavaIOExceptionReferenceCauseCallbackExecutionAndLazyPair(t *testing.T) {
	ioExceptionRegisterReferenceCause()
	execution := NewExecution()
	message := NewJavaStringUTF16([]uint16{114, 0, 0xd800, 'x', 0xdc00})
	trace := []string{}
	cause := &ioExceptionReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), message: message, trace: &trace}
	var failure IOException
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "enter")
		failure = NewJavaIOExceptionCauseExecution(execution, cause)
		trace = append(trace, "built")
	}()
	if JavaThrowableMessageDefault(failure) != message || GetCause(failure) != cause || cause.calls != 1 || !cause.held || !cause.same || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("IOException cause callback changed reference, count, execution, thread, or monitor cleanup")
	}
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "built", "reacquire"}) {
		t.Fatal("IOException cause callback changed observed execution order")
	}
	lazyTrace := []string{}
	lazy := &ioExceptionReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, abrupt: &failure, trace: &lazyTrace}
	direct := NewJavaIOExceptionMessageCause(message, lazy)
	if JavaThrowableMessageDefault(direct) != message || GetCause(direct) != lazy || lazy.calls != 0 || len(lazyTrace) != 0 {
		t.Fatal("(String,Throwable) constructor rendered its cause eagerly")
	}
}

func TestJavaIOExceptionReferenceAbruptCauseIdentityAndMonitorCleanup(t *testing.T) {
	ioExceptionRegisterReferenceCause()
	execution := NewExecution()
	markerValue := NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{'s'}))
	marker := &markerValue
	trace := []string{}
	cause := &ioExceptionReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), abrupt: marker, trace: &trace}
	failure := ioExceptionReferencePanic(t, func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "enter")
		NewJavaIOExceptionCauseExecution(execution, cause)
		trace = append(trace, "unreachable")
	})
	if failure != marker || cause.calls != 1 || !cause.held || !cause.same || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("IOException changed abrupt callback identity or caller context/cleanup")
	}
	trace = append(trace, "catch")
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "catch", "reacquire"}) {
		t.Fatal("IOException abrupt callback changed observed order")
	}
}
