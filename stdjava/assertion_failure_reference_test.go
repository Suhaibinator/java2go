package stdjava

import (
	"slices"
	"testing"
)

func assertionFailureReferencePanic(t *testing.T, invoke func()) (failure any) {
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

type assertionFailureReferenceDetail struct {
	value                  *JavaString
	execution              *Execution
	caller                 *Thread
	local                  *ThreadLocal[*JavaString]
	trace                  *[]string
	calls                  int
	held, same, contextual bool
	abrupt                 any
}

func (*assertionFailureReferenceDetail) JavaDynamicTypeID() TypeID {
	return "campaign.assertion.statement.Detail"
}
func (*assertionFailureReferenceDetail) String() string { panic("legacy native detail renderer") }
func (detail *assertionFailureReferenceDetail) DeclaredReference(execution *Execution) *JavaString {
	if execution != detail.execution {
		panic("assertion detail lost caller execution")
	}
	detail.calls++
	if detail.trace != nil {
		*detail.trace = append(*detail.trace, "render")
	}
	if detail.caller != nil {
		detail.same = ThreadCurrentThread(execution) == detail.caller
	}
	if detail.local != nil {
		detail.contextual = detail.local.Get(execution) == detail.value
	}
	detail.held = ThreadHoldsLockExecution(execution, detail)
	if detail.abrupt != nil {
		panic(detail.abrupt)
	}
	return detail.value
}

type assertionFailureReferenceCause struct {
	Exception
	*assertionFailureReferenceDetail
}

func (*assertionFailureReferenceCause) JavaDynamicTypeID() TypeID {
	return "campaign.assertion.statement.Cause"
}
func (*assertionFailureReferenceCause) String() string { panic("legacy native cause renderer") }

func registerAssertionFailureReferenceDetails() {
	RegisterJavaType("campaign.assertion.statement.Detail", ObjectTypeID)
	RegisterJavaSourceType("campaign.assertion.statement.Detail")
	RegisterJavaSourceToString("campaign.assertion.statement.Detail", "DeclaredReference")
	RegisterJavaType("campaign.assertion.statement.Cause", BuiltinThrowableTypeID("Exception"))
	RegisterJavaSourceType("campaign.assertion.statement.Cause")
	RegisterJavaSourceToString("campaign.assertion.statement.Cause", "DeclaredReference")
}

func TestAssertionFailureCanonicalDetailReferencesAndCauseSlots(t *testing.T) {
	registerAssertionFailureReferenceDetails()
	execution := NewExecution()
	units := []uint16{'r', 0, 0xd800, 'x', 0xdc00}
	message := NewJavaStringUTF16(units)
	probe := &assertionFailureReferenceDetail{value: message, execution: execution}
	nullProbe := &assertionFailureReferenceDetail{execution: execution}
	causeProbe := &assertionFailureReferenceDetail{value: message, execution: execution}
	cause := &assertionFailureReferenceCause{NewJavaExceptionMessage(nil), causeProbe}
	for _, test := range []struct {
		name      string
		failure   AssertionError
		message   *JavaString
		cause     any
		available bool
	}{
		{"empty", NewAssertionFailure(nil), nil, nil, true},
		{"string", NewAssertionFailure(execution, message), message, nil, true},
		{"null-detail", NewAssertionFailure(execution, nil), JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'}), nil, true},
		{"typed-null-string", NewAssertionFailure(execution, (*JavaString)(nil)), JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'}), nil, true},
		{"source", NewAssertionFailure(execution, probe), message, nil, true},
		{"source-null-result", NewAssertionFailure(execution, nullProbe), nil, nil, true},
		{"source-cause", NewAssertionFailure(execution, cause), message, cause, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if JavaThrowableMessageDefault(test.failure) != test.message || GetCause(test.failure) != test.cause {
				t.Fatal("assertion message/cause reference changed")
			}
			if test.message == message && !slices.Equal(JavaThrowableMessageDefault(test.failure).UTF16Copy(), units) {
				t.Fatal("assertion message UTF16 units changed")
			}
			replacement := NewJavaRuntimeExceptionMessage(nil)
			if test.available {
				returned := ThrowableInitCauseExecution(execution, test.failure, replacement)
				if !JavaReferenceEqual(returned, test.failure) || GetCause(test.failure) != replacement {
					t.Fatal("message assertion cause slot/returned identity changed")
				}
			} else {
				failure := assertionFailureReferencePanic(t, func() { _ = ThrowableInitCauseExecution(execution, test.failure, replacement) })
				if !CaughtAs(failure, "IllegalStateException") || GetCause(test.failure) != test.cause {
					t.Fatal("Throwable detail cause slot remained available")
				}
			}
		})
	}
	if probe.calls != 1 || nullProbe.calls != 1 || causeProbe.calls != 1 || !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("canonical assertion override count/input immutability changed")
	}
	if NewAssertionError("native").Message() != "native" || NewAssertionErrorExecution(nil, "native").Message() != "native" {
		t.Fatal("legacy host assertion constructor changed")
	}
}

func TestAssertionFailureCanonicalSourceCallerAndAbruptCleanup(t *testing.T) {
	registerAssertionFailureReferenceDetails()
	for _, abrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "return", true: "throw"}[abrupt], func(t *testing.T) {
			execution, sibling := NewExecution(), NewExecution()
			local := NewThreadLocal[*JavaString]()
			message := NewJavaStringUTF16([]uint16{'r', 0, 0xd800})
			local.Set(sibling, message)
			trace := []string{}
			markerValue := NewJavaIllegalArgumentExceptionMessage(message)
			marker := &markerValue
			detail := &assertionFailureReferenceDetail{value: message, execution: execution, caller: ThreadCurrentThread(execution), local: local, trace: &trace}
			if abrupt {
				detail.abrupt = marker
			}
			var built AssertionError
			invoke := func() {
				guard := MonitorEnterExecution(execution, detail)
				defer MonitorExitExecution(guard)
				local.Set(execution, message)
				defer func() { local.Remove(execution); trace = append(trace, "finally") }()
				trace = append(trace, "enter")
				built = NewAssertionFailure(execution, detail)
				trace = append(trace, "built")
			}
			if abrupt {
				if failure := assertionFailureReferencePanic(t, invoke); failure != marker {
					t.Fatal("abrupt detail lost marker identity")
				}
				trace = append(trace, "catch")
			} else {
				invoke()
				if JavaThrowableMessageDefault(built) != message || GetCause(built) != nil {
					t.Fatal("detail result/cause changed")
				}
			}
			if detail.calls != 1 || !detail.same || !detail.held || !detail.contextual || ThreadHoldsLockExecution(execution, detail) || len(execution.threadLocals) != 0 || local.Get(sibling) != message {
				t.Fatal("assertion detail lost caller/count/thread/local/monitor cleanup")
			}
			guard := MonitorEnterExecution(execution, detail)
			MonitorExitExecution(guard)
			trace = append(trace, "reacquire")
			want := []string{"enter", "render", "built", "finally", "reacquire"}
			if abrupt {
				want = []string{"enter", "render", "finally", "catch", "reacquire"}
			}
			if !slices.Equal(trace, want) {
				t.Fatalf("callback order: %v", trace)
			}
		})
	}
}

type assertionFailureReferenceInvalid struct {
	id    TypeID
	calls int
}

func (value *assertionFailureReferenceInvalid) JavaDynamicTypeID() TypeID { return value.id }
func (value *assertionFailureReferenceInvalid) Native(*Execution) string {
	value.calls++
	return "native"
}
func (value *assertionFailureReferenceInvalid) WrongArity() *JavaString { value.calls++; return nil }
func (value *assertionFailureReferenceInvalid) StringJava2goExecution(*Execution) *JavaString {
	value.calls++
	return nil
}

func TestAssertionFailureCanonicalSourceRefusals(t *testing.T) {
	execution := NewExecution()
	if failure := assertionFailureReferencePanic(t, func() { _ = NewAssertionFailure(execution, "native") }); !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatal("assertion bridge borrowed native string ingress")
	}
	for _, selector := range []string{"", "Native", "Missing", "WrongArity"} {
		id := TypeID("campaign.assertion.statement.Invalid" + selector)
		RegisterJavaType(id, ObjectTypeID)
		RegisterJavaSourceType(id)
		if selector != "" {
			RegisterJavaSourceToString(id, selector)
		}
		value := &assertionFailureReferenceInvalid{id: id}
		assertionFailureReferencePanic(t, func() { _ = NewAssertionFailure(execution, value) })
		if value.calls != 0 {
			t.Fatal("assertion borrowed ordinary/native source method")
		}
	}
}
