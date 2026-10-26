package stdjava

import (
	"reflect"
	"testing"
)

type exception165RegisteredCause struct {
	RuntimeException
	execution *Execution
	text      *JavaString
	calls     int
	held      bool
	reentrant bool
	abrupt    any
}

func (*exception165RegisteredCause) JavaDynamicTypeID() TypeID { return "exception165.RegisteredCause" }
func (p *exception165RegisteredCause) DeclaredText(e *Execution) *JavaString {
	if e != p.execution {
		panic("registered cause lost execution")
	}
	p.calls++
	p.held = ThreadHoldsLockExecution(e, p)
	if p.reentrant && JavaThrowableMessageDefault(NewExceptionExecution(e, p.text)) != p.text {
		panic("reentrant message boundary changed")
	}
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}
func TestException165RegisteredCauseReentrancyAndProtocolPriority(t *testing.T) {
	id := TypeID("exception165.RegisteredCause")
	RegisterJavaType(id, BuiltinThrowableTypeID("RuntimeException"))
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredText")
	kind := reflect.TypeOf((*exception165RegisteredCause)(nil))
	defer javaThrowableToStringOverrides.Delete(kind)
	defer throwableToStringOverrides.Delete(kind)
	for _, c := range exception165Constructors() {
		t.Run(c.name, func(t *testing.T) {
			javaThrowableToStringOverrides.Delete(kind)
			throwableToStringOverrides.Delete(kind)
			e := NewExecution()
			text := NewJavaStringUTF16([]uint16{0xd800, 0, 0xdc00})
			p := &exception165RegisteredCause{RuntimeException: NewJavaRuntimeExceptionMessage(text), execution: e, text: text, reentrant: true}
			guard := MonitorEnterExecution(e, p)
			defer MonitorExitExecution(guard)
			got := c.make(e, p)
			if JavaThrowableMessageDefault(got) != text || GetCause(got) != p || p.calls != 1 || !p.held {
				t.Fatal("registered source callback lost reference, identity, reentrancy or monitor")
			}
			legacyCalls := 0
			RegisterThrowableToStringOverride(p, func(seen *Execution, v any) string {
				if seen != e || v != p || !ThreadHoldsLockExecution(seen, p) {
					panic("legacy override lost receiver/execution")
				}
				legacyCalls++
				return "legacy override"
			})
			p.calls = 0
			got = c.make(e, p)
			if got.Message() != "legacy override" || GetCause(got) != p || legacyCalls != 1 || p.calls != 0 {
				t.Fatal("explicit legacy registration failed to keep its ABI precedence over canonical detail/source method")
			}
			canonicalCalls := 0
			RegisterJavaThrowableToStringOverride(p, func(seen *Execution, v any) *JavaString {
				if seen != e || v != p || !ThreadHoldsLockExecution(seen, p) {
					panic("canonical override lost receiver/execution")
				}
				canonicalCalls++
				return text
			})
			got = c.make(e, p)
			if JavaThrowableMessageDefault(got) != text || GetCause(got) != p || canonicalCalls != 1 || legacyCalls != 1 {
				t.Fatal("explicit canonical registration failed to take its declared protocol")
			}
			javaThrowableToStringOverrides.Delete(kind)
			throwableToStringOverrides.Delete(kind)
			p.calls = 0
			p.abrupt = &struct{ n int }{9}
			func() {
				defer func() {
					if v := recover(); v != p.abrupt {
						t.Errorf("registered callback abrupt identity=%T", v)
					}
				}()
				_ = c.make(e, p)
				t.Error("registered abrupt completion swallowed")
			}()
			if p.calls != 1 || !p.held {
				t.Fatal("abrupt source callback repeated or lost monitor")
			}
		})
	}
}

type exception165PlainCause struct{ RuntimeException }

func TestException165LegacyGetterPrecedenceOverCanonicalField(t *testing.T) {
	kind := reflect.TypeOf((*exception165PlainCause)(nil))
	defer throwableLocalizedMessageOverrides.Delete(kind)
	defer throwableMessageOverrides.Delete(kind)
	for _, localized := range []bool{false, true} {
		throwableLocalizedMessageOverrides.Delete(kind)
		throwableMessageOverrides.Delete(kind)
		e := NewExecution()
		p := &exception165PlainCause{NewJavaRuntimeExceptionMessage(NewJavaStringUTF16([]uint16{0xd800}))}
		calls := 0
		invoke := func(seen *Execution, v any) string {
			if seen != e || v != p {
				panic("legacy getter lost execution")
			}
			calls++
			return "legacy getter"
		}
		if localized {
			RegisterThrowableLocalizedMessageOverride(p, invoke)
		} else {
			RegisterThrowableMessage(p, invoke)
		}
		got := NewExceptionExecution(e, p)
		if got.Message() != "java.lang.RuntimeException: legacy getter" || calls != 1 || GetCause(got) != p {
			t.Fatal("legacy getter protocol was displaced by canonical field")
		}
	}
}

func TestException165TypedNullPairCauseInitialization(t *testing.T) {
	for _, c := range exception165Constructors() {
		t.Run(c.name, func(t *testing.T) {
			e := NewExecution()
			var message *JavaString
			var cause *Exception
			got := c.make(e, message, cause)
			if JavaThrowableMessageDefault(got) != nil || GetCause(got) != nil {
				t.Fatal("typed null pair acquired a reference")
			}
			expectBoxedException(t, "IllegalStateException", func() { _ = ThrowableInitCauseExecution(e, got, nil) })
		})
	}
}
