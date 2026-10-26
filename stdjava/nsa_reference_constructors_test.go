package stdjava

import (
	"slices"
	"testing"
)

func nsaConstructorReferencePanic(t *testing.T, invoke func()) (failure any) {
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

func TestJavaNoSuchAlgorithmExceptionReferenceMessageAndCauseSlots(t *testing.T) {
	execution := NewExecution()
	units := []uint16{114, 0, 0xd800, 'x', 0xdc00}
	message := NewJavaStringUTF16(units)
	cause := NewJavaExceptionMessage(nil)
	for _, test := range []struct {
		name      string
		failure   NoSuchAlgorithmException
		message   *JavaString
		cause     any
		available bool
	}{
		{"empty", NewJavaNoSuchAlgorithmExceptionMessage(nil), nil, nil, true},
		{"string", NewJavaNoSuchAlgorithmExceptionMessage(message), message, nil, true},
		{"typed-null-string", NewJavaNoSuchAlgorithmExceptionMessage((*JavaString)(nil)), nil, nil, true},
		{"both", NewJavaNoSuchAlgorithmExceptionMessageCause(message, cause), message, cause, false},
		{"null-message-cause", NewJavaNoSuchAlgorithmExceptionMessageCause(nil, cause), nil, cause, false},
		{"message-null-cause", NewJavaNoSuchAlgorithmExceptionMessageCause(message, nil), message, nil, false},
		{"both-null", NewJavaNoSuchAlgorithmExceptionMessageCause(nil, nil), nil, nil, false},
		{"typed-null-cause", NewJavaNoSuchAlgorithmExceptionCauseExecution(execution, (*Exception)(nil)), nil, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.failure.ThrowableTypeName() != "NoSuchAlgorithmException" || JavaThrowableMessageDefault(test.failure) != test.message || GetCause(test.failure) != test.cause {
				t.Fatal("NoSuchAlgorithmException changed exact message/cause references or overload identity")
			}
			if test.message != nil && !slices.Equal(JavaThrowableMessageDefault(test.failure).UTF16Copy(), units) {
				t.Fatal("NoSuchAlgorithmException changed NUL/isolated-surrogate UTF16 message units")
			}
			replacement := NewJavaRuntimeExceptionMessage(nil)
			if test.available {
				returned := ThrowableInitCauseExecution(execution, test.failure, replacement)
				if !JavaReferenceEqual(returned, test.failure) || GetCause(test.failure) != replacement {
					t.Fatal("String constructor did not leave initCause available or preserve returned/cause identity")
				}
			} else {
				failure := nsaConstructorReferencePanic(t, func() {
					returned := ThrowableInitCauseExecution(execution, test.failure, replacement)
					t.Fatalf("expected locked cause rejection, returned %T", returned)
				})
				if exception, ok := failure.(Throwable); !ok || exception.ThrowableTypeName() != "IllegalStateException" || GetCause(test.failure) != test.cause {
					t.Fatal("explicit cause overload failed to close the cause slot")
				}
			}
		})
	}
	if !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("original immutable message changed")
	}
	self := NewJavaNoSuchAlgorithmExceptionMessage(nil)
	if failure := nsaConstructorReferencePanic(t, func() { _ = ThrowableInitCauseExecution(execution, self, self) }); !CaughtAs(failure, "IllegalArgumentException") || GetCause(self) != nil {
		t.Fatal("self cause rejection changed")
	}
	if got := NewNoSuchAlgorithmException("native").Message(); got != "native" {
		t.Fatal("native Go NoSuchAlgorithmException constructor changed")
	}
}

type nsaConstructorReferenceCause struct {
	Exception
	execution  *Execution
	caller     *Thread
	message    *JavaString
	abrupt     any
	trace      *[]string
	calls      int
	held       bool
	same       bool
	local      *ThreadLocal[*JavaString]
	contextual bool
}

func (*nsaConstructorReferenceCause) JavaDynamicTypeID() TypeID {
	return "campaign.nsa.ConstructorCause"
}
func (*nsaConstructorReferenceCause) String() string { panic("native cause rendering") }
func (cause *nsaConstructorReferenceCause) DeclaredText(execution *Execution) *JavaString {
	if execution != cause.execution {
		panic("NoSuchAlgorithmException lost caller execution")
	}
	cause.calls++
	cause.held = ThreadHoldsLockExecution(execution, cause)
	cause.same = ThreadCurrentThread(execution) == cause.caller
	cause.contextual = cause.local.Get(execution) == cause.message
	*cause.trace = append(*cause.trace, "render")
	if cause.abrupt != nil {
		panic(cause.abrupt)
	}
	return cause.message
}

func nsaConstructorRegisterReferenceCause() {
	id := TypeID("campaign.nsa.ConstructorCause")
	RegisterJavaType(id, BuiltinThrowableTypeID("Exception"))
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredText")
}

func TestJavaNoSuchAlgorithmExceptionReferenceCauseCallerAndLazyPair(t *testing.T) {
	nsaConstructorRegisterReferenceCause()
	execution := NewExecution()
	message := NewJavaStringUTF16([]uint16{114, 0, 0xd800, 'x', 0xdc00})
	trace := []string{}
	local := NewThreadLocal[*JavaString]()
	sibling := NewExecution()
	local.Set(sibling, message)
	cause := &nsaConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), message: message, trace: &trace, local: local}
	var failure NoSuchAlgorithmException
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		local.Set(execution, cause.message)
		defer func() { local.Remove(execution); trace = append(trace, "finally") }()
		trace = append(trace, "enter")
		failure = NewJavaNoSuchAlgorithmExceptionCauseExecution(execution, cause)
		trace = append(trace, "built")
	}()
	if JavaThrowableMessageDefault(failure) != message || GetCause(failure) != cause || cause.calls != 1 || !cause.held || !cause.same || !cause.contextual || len(execution.threadLocals) != 0 || local.Get(sibling) != message || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("NoSuchAlgorithmException cause callback changed reference, count, execution, thread, or monitor cleanup")
	}
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "built", "finally", "reacquire"}) {
		t.Fatal("NoSuchAlgorithmException cause callback changed observed execution order")
	}
	lazyTrace := []string{}
	lazy := &nsaConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, abrupt: &failure, trace: &lazyTrace}
	direct := NewJavaNoSuchAlgorithmExceptionMessageCause(message, lazy)
	if JavaThrowableMessageDefault(direct) != message || GetCause(direct) != lazy || lazy.calls != 0 || len(lazyTrace) != 0 {
		t.Fatal("(String,Throwable) constructor rendered its cause eagerly")
	}
}

func TestJavaNoSuchAlgorithmExceptionReferenceAbruptCauseIdentityAndCleanup(t *testing.T) {
	nsaConstructorRegisterReferenceCause()
	execution := NewExecution()
	markerValue := NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{'s'}))
	marker := &markerValue
	trace := []string{}
	message := NewJavaStringUTF16([]uint16{114, 0, 0xd800})
	local := NewThreadLocal[*JavaString]()
	sibling := NewExecution()
	local.Set(sibling, message)
	cause := &nsaConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), message: message, abrupt: marker, trace: &trace, local: local}
	failure := nsaConstructorReferencePanic(t, func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		local.Set(execution, cause.message)
		defer func() { local.Remove(execution); trace = append(trace, "finally") }()
		trace = append(trace, "enter")
		returned := NewJavaNoSuchAlgorithmExceptionCauseExecution(execution, cause)
		trace = append(trace, "unreachable")
		t.Fatalf("expected abrupt cause callback, returned %T", returned)
	})
	if failure != marker || cause.calls != 1 || !cause.held || !cause.same || !cause.contextual || len(execution.threadLocals) != 0 || local.Get(sibling) != message || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("NoSuchAlgorithmException changed abrupt callback identity or caller context/cleanup")
	}
	trace = append(trace, "catch")
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "finally", "catch", "reacquire"}) {
		t.Fatal("NoSuchAlgorithmException abrupt callback changed observed order")
	}
}
