package stdjava

import (
	"slices"
	"testing"
)

func TestJavaThrowableConstructorReferenceMessages(t *testing.T) {
	execution := NewExecution()
	message := NewJavaStringUTF16([]uint16{'Q', 0xd800, 0, 0xdc00})
	cause := NewJavaExceptionMessage(message)
	for name, failure := range map[string]any{
		"Assertion Object":       NewJavaAssertionErrorExecution(execution, message),
		"Assertion pair":         NewJavaAssertionErrorExecution(execution, message, cause),
		"IllegalArgument String": NewJavaIllegalArgumentExceptionMessage(message),
		"IllegalArgument pair":   NewJavaIllegalArgumentExceptionMessageCause(message, cause),
	} {
		t.Run(name, func(t *testing.T) {
			if JavaThrowableMessageDefault(failure) != message {
				t.Fatal("constructor copied or rendered the original UTF16 message reference")
			}
		})
	}
	for name, failure := range map[string]any{
		"Assertion pair":       NewJavaAssertionErrorExecution(execution, message, cause),
		"IllegalArgument pair": NewJavaIllegalArgumentExceptionMessageCause(message, cause),
	} {
		t.Run(name+" cause", func(t *testing.T) {
			if GetCause(failure) != cause {
				t.Fatal("pair constructor changed the cause identity")
			}
			expectBoxedException(t, "IllegalStateException", func() {
				got := ThrowableInitCauseExecution(execution, failure, nil)
				t.Fatalf("initCause unexpectedly returned %T instead of throwing", got)
			})
		})
	}
	if got := JavaThrowableMessageDefault(NewJavaAssertionErrorCharExecution(execution, 0xd800)); got == nil || !slices.Equal(got.UTF16Copy(), []uint16{0xd800}) {
		t.Fatal("char constructor changed an isolated UTF16 unit")
	}
	if got := JavaThrowableMessageDefault(NewJavaAssertionErrorExecution(execution)); got != nil {
		t.Fatal("no-argument assertion message must be null")
	}
	if got := JavaThrowableMessageDefault(NewJavaAssertionErrorExecution(execution, (*JavaString)(nil))); got == nil || !slices.Equal(got.UTF16Copy(), []uint16{'n', 'u', 'l', 'l'}) {
		t.Fatal("Object-null detail must become text null")
	}
}

func TestJavaThrowableConstructorNullOverloadState(t *testing.T) {
	execution := NewExecution()
	for name, construct := range map[string]func() any{
		"Assertion noarg":             func() any { return NewJavaAssertionErrorExecution(execution) },
		"Assertion Object-null":       func() any { return NewJavaAssertionErrorExecution(execution, (*JavaString)(nil)) },
		"IllegalArgument String-null": func() any { return NewJavaIllegalArgumentExceptionMessage(nil) },
	} {
		t.Run(name, func(t *testing.T) {
			failure := construct()
			cause := NewJavaExceptionMessage(nil)
			if got := ThrowableInitCauseExecution(execution, failure, cause); got != failure {
				t.Fatal("initCause changed returned receiver identity")
			}
			if GetCause(failure) != cause {
				t.Fatal("available cause slot changed assigned reference")
			}
		})
	}
	for name, failure := range map[string]any{
		"Assertion null pair":            NewJavaAssertionErrorExecution(execution, nil, nil),
		"IllegalArgument Throwable-null": NewJavaIllegalArgumentExceptionCauseExecution(execution, nil),
		"IllegalArgument null pair":      NewJavaIllegalArgumentExceptionMessageCause(nil, nil),
	} {
		t.Run(name, func(t *testing.T) {
			if JavaThrowableMessageDefault(failure) != nil || GetCause(failure) != nil {
				t.Fatal("explicit null message/cause changed")
			}
			expectBoxedException(t, "IllegalStateException", func() {
				got := ThrowableInitCauseExecution(execution, failure, nil)
				t.Fatalf("initCause unexpectedly returned %T instead of throwing", got)
			})
		})
	}
}

type throwableConstructorReferenceProbe struct {
	RuntimeException
	execution *Execution
	text      *JavaString
	calls     int
	held      bool
	abrupt    any
}

func (*throwableConstructorReferenceProbe) JavaDynamicTypeID() TypeID {
	return "constructor.reference.Probe"
}
func (p *throwableConstructorReferenceProbe) DeclaredDetail(execution *Execution) *JavaString {
	if execution != p.execution {
		panic("constructor lost execution")
	}
	p.calls++
	p.held = ThreadHoldsLockExecution(execution, p)
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}
func (*throwableConstructorReferenceProbe) String() string { panic("native text conversion") }

func TestJavaThrowableConstructorCallbackExecutionAndAbruptIdentity(t *testing.T) {
	id := TypeID("constructor.reference.Probe")
	RegisterJavaType(id, BuiltinThrowableTypeID("RuntimeException"))
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredDetail")
	execution := NewExecution()
	text := NewJavaStringUTF16([]uint16{0xd800, 0, 0xdc00})
	for name, construct := range map[string]func(any) any{
		"Assertion Object":          func(value any) any { return NewJavaAssertionErrorExecution(execution, value) },
		"IllegalArgument Throwable": func(value any) any { return NewJavaIllegalArgumentExceptionCauseExecution(execution, value) },
	} {
		t.Run(name, func(t *testing.T) {
			probe := &throwableConstructorReferenceProbe{RuntimeException: NewJavaRuntimeExceptionMessage(nil), execution: execution, text: text}
			guard := MonitorEnterExecution(execution, probe)
			defer MonitorExitExecution(guard)
			failure := construct(probe)
			if JavaThrowableMessageDefault(failure) != text || GetCause(failure) != probe || probe.calls != 1 || !probe.held {
				t.Fatal("constructor changed callback reference, cause, dispatch count, or execution monitor")
			}
			probe.text = nil
			nullText := construct(probe)
			if JavaThrowableMessageDefault(nullText) != nil || probe.calls != 2 {
				t.Fatal("returned null was replaced or callback repeated")
			}
			marker := &struct{ value int }{7}
			probe.abrupt = marker
			func() {
				defer func() {
					if got := recover(); got != marker {
						t.Fatalf("callback abrupt identity=%v want %v", got, marker)
					}
				}()
				construct(probe)
				t.Fatal("constructor swallowed callback failure")
			}()
			if probe.calls != 3 || !probe.held {
				t.Fatal("abrupt callback lost dispatch count or caller monitor")
			}
		})
	}
}

func TestJavaAssertionConstructorKeepsNativeBoundaryGuard(t *testing.T) {
	expectBoxedException(t, "UnsupportedOperationException", func() {
		got := NewJavaAssertionErrorExecution(NewExecution(), "native Go text")
		t.Fatalf("native-string constructor unexpectedly returned %T instead of throwing", got)
	})
	if got := NewAssertionError("native Go text").Message(); got != "native Go text" {
		t.Fatal("legacy native constructor behavior changed")
	}
}
