package stdjava

import (
	"slices"
	"testing"
)

type exception165Constructor struct {
	name string
	make func(*Execution, ...any) Throwable
}

func exception165Constructors() []exception165Constructor {
	return []exception165Constructor{
		{"Throwable", func(e *Execution, a ...any) Throwable { return NewThrowableExecution(e, a...) }},
		{"Exception", func(e *Execution, a ...any) Throwable { return NewExceptionExecution(e, a...) }},
		{"RuntimeException", func(e *Execution, a ...any) Throwable { return NewRuntimeExceptionExecution(e, a...) }},
		{"IllegalArgumentException", func(e *Execution, a ...any) Throwable { return NewIllegalArgumentExceptionExecution(e, a...) }},
		{"IllegalStateException", func(e *Execution, a ...any) Throwable { return NewIllegalStateExceptionExecution(e, a...) }},
		{"UnsupportedEncodingException native companion", func(e *Execution, a ...any) Throwable { return NewUnsupportedEncodingExceptionExecution(e, a...) }},
	}
}

func exception165NoPanic(t *testing.T, call func()) {
	t.Helper()
	defer func() {
		if got := recover(); got != nil {
			t.Errorf("existing constructor API panicked: %T: %v", got, got)
		}
	}()
	call()
}

func TestException165ExistingCanonicalMessageAPIRED(t *testing.T) {
	e := NewExecution()
	message := NewJavaStringUTF16([]uint16{'m', 0xd800, 0, 0xdc00})
	cause := NewJavaExceptionMessage(JavaStringFromHostUTF8("cause"))
	for _, c := range exception165Constructors() {
		for _, pair := range []bool{false, true} {
			name := c.name + " message"
			if pair {
				name += " cause"
			}
			t.Run(name, func(t *testing.T) {
				exception165NoPanic(t, func() {
					args := []any{message}
					if pair {
						args = append(args, cause)
					}
					failure := c.make(e, args...)
					if got := JavaThrowableMessageDefault(failure); got != message || !slices.Equal(got.UTF16Copy(), message.UTF16Copy()) {
						t.Error("canonical message reference or exact isolated UTF16 units changed")
					}
					if pair {
						if GetCause(failure) != cause {
							t.Error("explicit cause identity changed")
						}
						if !failure.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
							t.Error("explicit cause slot left open")
						}
					} else {
						if GetCause(failure) != nil {
							t.Error("message was treated as a cause")
						}
						if failure.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
							t.Error("message-only cause slot closed")
						}
					}
				})
			})
		}
	}
}

func TestException165TypedNullStringVsCauseAPIRED(t *testing.T) {
	e := NewExecution()
	for _, c := range exception165Constructors() {
		t.Run(c.name, func(t *testing.T) {
			exception165NoPanic(t, func() {
				var text *JavaString
				message := c.make(e, text)
				if JavaThrowableMessageDefault(message) != nil || GetCause(message) != nil {
					t.Error("typed null String changed")
				}
				if message.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
					t.Error("typed null String incorrectly selects initialized cause overload")
				}
				cause := NewException("later")
				if got := ThrowableInitCauseExecution(e, message, cause); got != message || GetCause(message) != cause {
					t.Error("typed null message must allow later initCause")
				}
			})
		})
	}
}

type exception165CanonicalCause struct {
	RuntimeException
	execution *Execution
	text      *JavaString
	calls     int
	held      bool
	abrupt    any
}

func (p *exception165CanonicalCause) StringJava2goExecution(e *Execution) *JavaString {
	if e != p.execution {
		panic("cause callback lost execution")
	}
	p.calls++
	p.held = ThreadHoldsLockExecution(e, p)
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}
func TestException165CanonicalCauseCallbackAPIRED(t *testing.T) {
	for _, c := range exception165Constructors() {
		t.Run(c.name, func(t *testing.T) {
			exception165NoPanic(t, func() {
				e := NewExecution()
				text := NewJavaStringUTF16([]uint16{0xd800, 0, 0xdc00})
				probe := &exception165CanonicalCause{RuntimeException: NewJavaRuntimeExceptionMessage(JavaStringFromHostUTF8("base")), execution: e, text: text}
				guard := MonitorEnterExecution(e, probe)
				defer MonitorExitExecution(guard)
				failure := c.make(e, probe)
				if JavaThrowableMessageDefault(failure) != text || GetCause(failure) != probe || probe.calls != 1 || !probe.held {
					t.Errorf("reference callback changed: calls=%d held=%t", probe.calls, probe.held)
				}
				probe.calls = 0
				explicit := c.make(e, text, probe)
				if JavaThrowableMessageDefault(explicit) != text || GetCause(explicit) != probe || probe.calls != 0 {
					t.Error("explicit pair invoked cause callback or changed references")
				}
				probe.text = nil
				probe.calls = 0
				if JavaThrowableMessageDefault(c.make(e, probe)) != nil || probe.calls != 1 {
					t.Error("null callback result changed or callback not once")
				}
				marker := &struct{ n int }{7}
				probe.abrupt = marker
				probe.calls = 0
				func() {
					defer func() {
						if got := recover(); got != marker {
							t.Errorf("callback abrupt identity=%T want original marker", got)
						}
					}()
					_ = c.make(e, probe)
					t.Error("cause callback abrupt completion swallowed")
				}()
				if probe.calls != 1 || !probe.held {
					t.Error("abrupt callback count/execution changed")
				}
			})
		})
	}
}

func TestException165LegacyAndNullCauseShields(t *testing.T) {
	e := NewExecution()
	for _, c := range exception165Constructors() {
		t.Run(c.name, func(t *testing.T) {
			for _, text := range []string{"legacy", "", NullString()} {
				failure := c.make(e, text)
				if failure.Message() != text || GetCause(failure) != nil || failure.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
					t.Error("legacy Go String boundary changed")
				}
			}
			for _, args := range [][]any{{nil}, {(*Exception)(nil)}, {NullString(), nil}, {nil, nil}} {
				failure := c.make(e, args...)
				if JavaThrowableMessageDefault(failure) != nil || GetCause(failure) != nil || !failure.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
					t.Error("null cause/pair contract changed")
				}
			}
			noarg := c.make(e)
			if JavaThrowableMessageDefault(noarg) != nil || GetCause(noarg) != nil || noarg.(interface{ throwableCauseState() *throwableState }).throwableCauseState().causeInitialized {
				t.Error("noarg state changed")
			}
		})
	}
}
