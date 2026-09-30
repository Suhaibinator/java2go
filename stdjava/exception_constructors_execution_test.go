package stdjava_test

import (
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

// An external package models a generated source class: Error is promoted from
// Throwable, while the Java override has an execution-aware String companion.
type causeStringProbe struct {
	stdjava.RuntimeException
	seen   *stdjava.Execution
	calls  int
	lock   any
	held   bool
	text   string
	abrupt any
}

func (p *causeStringProbe) String() string { return "wrong non-execution dispatch" }
func (p *causeStringProbe) StringJava2goExecution(execution *stdjava.Execution) string {
	p.calls++
	p.seen = execution
	if p.lock != nil {
		p.held = stdjava.ThreadHoldsLockExecution(execution, p.lock)
	}
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}

func TestExceptionConstructorsForwardCauseExecution(t *testing.T) {
	constructors := map[string]func(*stdjava.Execution, ...any) stdjava.Throwable{
		"Exception": func(e *stdjava.Execution, args ...any) stdjava.Throwable {
			return stdjava.NewExceptionExecution(e, args...)
		},
		"RuntimeException": func(e *stdjava.Execution, args ...any) stdjava.Throwable {
			return stdjava.NewRuntimeExceptionExecution(e, args...)
		},
		"IllegalArgumentException": func(e *stdjava.Execution, args ...any) stdjava.Throwable {
			return stdjava.NewIllegalArgumentExceptionExecution(e, args...)
		},
		"IllegalStateException": func(e *stdjava.Execution, args ...any) stdjava.Throwable {
			return stdjava.NewIllegalStateExceptionExecution(e, args...)
		},
		"UnsupportedEncodingException Go companion": func(e *stdjava.Execution, args ...any) stdjava.Throwable {
			return stdjava.NewUnsupportedEncodingExceptionExecution(e, args...)
		},
	}
	for name, construct := range constructors {
		t.Run(name, func(t *testing.T) {
			execution := stdjava.NewExecution()
			probe := &causeStringProbe{RuntimeException: stdjava.NewRuntimeException("source"), text: "source-cause:source"}
			probe.lock = probe
			guard := stdjava.MonitorEnterExecution(execution, probe)
			defer stdjava.MonitorExitExecution(guard)
			got := construct(execution, probe)
			if got.Message() != probe.text || probe.calls != 1 || probe.seen != execution || !probe.held {
				t.Fatalf("message=%q calls=%d same execution=%t held=%t", got.Message(), probe.calls, probe.seen == execution, probe.held)
			}
			if stdjava.GetCause(got) != probe {
				t.Fatal("cause reference identity changed")
			}
			probe.calls = 0
			explicit := construct(execution, "explicit", probe)
			if explicit.Message() != "explicit" || stdjava.GetCause(explicit) != probe || probe.calls != 0 {
				t.Fatal("message+cause constructor invoked toString or changed state")
			}
			probe.text = stdjava.NullString()
			if !stdjava.StringIsNull(construct(execution, probe).Message()) || probe.calls != 1 {
				t.Fatal("null toString result was converted to text")
			}
		})
	}
}

func TestExceptionCauseCallbackAbruptAndNull(t *testing.T) {
	execution := stdjava.NewExecution()
	marker := &struct{ value int }{7}
	probe := &causeStringProbe{RuntimeException: stdjava.NewRuntimeException("source"), abrupt: marker}
	func() {
		defer func() {
			if got := recover(); got != marker {
				t.Fatalf("callback panic identity=%v want %v", got, marker)
			}
		}()
		_ = stdjava.NewExceptionExecution(execution, probe)
		t.Fatal("constructor swallowed callback failure")
	}()
	if probe.calls != 1 || probe.seen != execution {
		t.Fatal("abrupt callback lost execution or ran more than once")
	}
	var missing *causeStringProbe
	got := stdjava.NewExceptionExecution(execution, missing)
	if !stdjava.StringIsNull(got.Message()) || stdjava.GetCause(got) != nil {
		t.Fatal("typed null cause state changed")
	}
}

func TestExceptionLegacyGoWrapperCreatesCauseExecution(t *testing.T) {
	probe := &causeStringProbe{RuntimeException: stdjava.NewRuntimeException("source"), text: "legacy"}
	got := stdjava.NewException(probe)
	if got.Message() != "legacy" || probe.calls != 1 || probe.seen == nil || stdjava.GetCause(got) != probe {
		t.Fatal("legacy wrapper did not preserve source dispatch/cause")
	}
	builtin := stdjava.NewExceptionExecution(stdjava.NewExecution(), stdjava.NewIOException("io"))
	if builtin.Message() != "java.io.IOException: io" {
		t.Fatalf("builtin cause formatting=%q", builtin.Message())
	}
}
