package stdjava

import (
	"slices"
	"testing"
)

func gseConstructorReferencePanic(t *testing.T, invoke func()) (failure any) {
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

func TestJavaGeneralSecurityExceptionReferenceMessageAndCauseSlots(t *testing.T) {
	execution := NewExecution()
	units := []uint16{114, 0, 0xd800, 'x', 0xdc00}
	message := NewJavaStringUTF16(units)
	cause := NewJavaExceptionMessage(nil)
	for _, test := range []struct {
		name      string
		failure   GeneralSecurityException
		message   *JavaString
		cause     any
		available bool
	}{
		{"empty", NewJavaGeneralSecurityExceptionMessage(nil), nil, nil, true},
		{"string", NewJavaGeneralSecurityExceptionMessage(message), message, nil, true},
		{"typed-null-string", NewJavaGeneralSecurityExceptionMessage((*JavaString)(nil)), nil, nil, true},
		{"both", NewJavaGeneralSecurityExceptionMessageCause(message, cause), message, cause, false},
		{"null-message-cause", NewJavaGeneralSecurityExceptionMessageCause(nil, cause), nil, cause, false},
		{"message-null-cause", NewJavaGeneralSecurityExceptionMessageCause(message, nil), message, nil, false},
		{"both-null", NewJavaGeneralSecurityExceptionMessageCause(nil, nil), nil, nil, false},
		{"typed-null-cause", NewJavaGeneralSecurityExceptionCauseExecution(execution, (*Exception)(nil)), nil, nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.failure.ThrowableTypeName() != "GeneralSecurityException" || JavaThrowableMessageDefault(test.failure) != test.message || GetCause(test.failure) != test.cause {
				t.Fatal("GeneralSecurityException changed exact message/cause references or overload identity")
			}
			if test.message != nil && !slices.Equal(JavaThrowableMessageDefault(test.failure).UTF16Copy(), units) {
				t.Fatal("GeneralSecurityException changed NUL/isolated-surrogate UTF16 message units")
			}
			replacement := NewJavaRuntimeExceptionMessage(nil)
			if test.available {
				returned := ThrowableInitCauseExecution(execution, test.failure, replacement)
				if !JavaReferenceEqual(returned, test.failure) || GetCause(test.failure) != replacement {
					t.Fatal("String constructor did not leave initCause available or preserve returned/cause identity")
				}
			} else {
				failure := gseConstructorReferencePanic(t, func() {
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
	self := NewJavaGeneralSecurityExceptionMessage(nil)
	if failure := gseConstructorReferencePanic(t, func() { _ = ThrowableInitCauseExecution(execution, self, self) }); !CaughtAs(failure, "IllegalArgumentException") || GetCause(self) != nil {
		t.Fatal("self cause rejection changed")
	}
}

type gseConstructorReferenceCause struct {
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

func (*gseConstructorReferenceCause) JavaDynamicTypeID() TypeID {
	return "campaign.gse.ConstructorCause"
}
func (*gseConstructorReferenceCause) String() string { panic("native cause rendering") }
func (cause *gseConstructorReferenceCause) DeclaredText(execution *Execution) *JavaString {
	if execution != cause.execution {
		panic("GeneralSecurityException lost caller execution")
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

func gseConstructorRegisterReferenceCause() {
	id := TypeID("campaign.gse.ConstructorCause")
	RegisterJavaType(id, BuiltinThrowableTypeID("Exception"))
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredText")
}

func TestJavaGeneralSecurityExceptionReferenceCauseCallerAndLazyPair(t *testing.T) {
	gseConstructorRegisterReferenceCause()
	execution := NewExecution()
	message := NewJavaStringUTF16([]uint16{114, 0, 0xd800, 'x', 0xdc00})
	trace := []string{}
	local := NewThreadLocal[*JavaString]()
	sibling := NewExecution()
	local.Set(sibling, message)
	cause := &gseConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), message: message, trace: &trace, local: local}
	var failure GeneralSecurityException
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		local.Set(execution, cause.message)
		defer func() { local.Remove(execution); trace = append(trace, "finally") }()
		trace = append(trace, "enter")
		failure = NewJavaGeneralSecurityExceptionCauseExecution(execution, cause)
		trace = append(trace, "built")
	}()
	if JavaThrowableMessageDefault(failure) != message || GetCause(failure) != cause || cause.calls != 1 || !cause.held || !cause.same || !cause.contextual || len(execution.threadLocals) != 0 || local.Get(sibling) != message || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("GeneralSecurityException cause callback changed reference, count, execution, thread, or monitor cleanup")
	}
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "built", "finally", "reacquire"}) {
		t.Fatal("GeneralSecurityException cause callback changed observed execution order")
	}
	lazyTrace := []string{}
	lazy := &gseConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, abrupt: &failure, trace: &lazyTrace}
	direct := NewJavaGeneralSecurityExceptionMessageCause(message, lazy)
	if JavaThrowableMessageDefault(direct) != message || GetCause(direct) != lazy || lazy.calls != 0 || len(lazyTrace) != 0 {
		t.Fatal("(String,Throwable) constructor rendered its cause eagerly")
	}
}

func TestJavaGeneralSecurityExceptionReferenceAbruptCauseIdentityAndCleanup(t *testing.T) {
	gseConstructorRegisterReferenceCause()
	execution := NewExecution()
	markerValue := NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{'s'}))
	marker := &markerValue
	trace := []string{}
	message := NewJavaStringUTF16([]uint16{114, 0, 0xd800})
	local := NewThreadLocal[*JavaString]()
	sibling := NewExecution()
	local.Set(sibling, message)
	cause := &gseConstructorReferenceCause{Exception: NewJavaExceptionMessage(nil), execution: execution, caller: ThreadCurrentThread(execution), message: message, abrupt: marker, trace: &trace, local: local}
	failure := gseConstructorReferencePanic(t, func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		local.Set(execution, cause.message)
		defer func() { local.Remove(execution); trace = append(trace, "finally") }()
		trace = append(trace, "enter")
		returned := NewJavaGeneralSecurityExceptionCauseExecution(execution, cause)
		trace = append(trace, "unreachable")
		t.Fatalf("expected abrupt cause callback, returned %T", returned)
	})
	if failure != marker || cause.calls != 1 || !cause.held || !cause.same || !cause.contextual || len(execution.threadLocals) != 0 || local.Get(sibling) != message || ThreadHoldsLockExecution(execution, cause) {
		t.Fatal("GeneralSecurityException changed abrupt callback identity or caller context/cleanup")
	}
	trace = append(trace, "catch")
	func() {
		guard := MonitorEnterExecution(execution, cause)
		defer MonitorExitExecution(guard)
		trace = append(trace, "reacquire")
	}()
	if !slices.Equal(trace, []string{"enter", "render", "finally", "catch", "reacquire"}) {
		t.Fatal("GeneralSecurityException abrupt callback changed observed order")
	}
}
