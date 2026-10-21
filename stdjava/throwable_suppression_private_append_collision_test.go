package stdjava_test

import (
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

// This has the same embedding and exported business-method shape as a legal
// Java source subclass declaring public addSuppressedValue(Object).
type suppressionBusinessCollision struct {
	j.Exception
	auditCalls int
	auditValue any
}

func (p *suppressionBusinessCollision) AddSuppressedValue(value any) {
	p.auditCalls++
	p.auditValue = value
}

func captureSuppressionPanic(f func()) (caught any) {
	defer func() { caught = recover() }()
	f()
	return nil
}

func TestJavaThrowableSuppressionAppendIgnoresSourceBusinessMethod(t *testing.T) {
	for _, route := range []string{"explicit", "tryWithResources"} {
		t.Run(route, func(t *testing.T) {
			primary := &suppressionBusinessCollision{Exception: j.NewJavaExceptionMessageCause(nil, nil)}
			primary.AddSuppressedValue("audit")
			firstValue, secondValue := j.NewJavaExceptionMessage(nil), j.NewJavaExceptionMessage(nil)
			first, second := &firstValue, &secondValue
			appendOne := func(value any) {
				if route == "explicit" {
					j.JavaThrowableAddSuppressedExecution(j.NewExecution(), primary, value)
					return
				}
				caught := captureSuppressionPanic(func() {
					defer j.CloseResource(func() { panic(value) })
					panic(primary)
				})
				if caught != primary {
					t.Fatalf("cleanup changed primary reference: %T", caught)
				}
			}
			appendOne(first)
			appendOne(second)
			if primary.auditCalls != 1 || primary.auditValue != "audit" {
				t.Errorf("native append invoked source business method: calls=%d value=%v", primary.auditCalls, primary.auditValue)
			}
			assertStored := func() {
				t.Helper()
				values := j.GetSuppressed(primary)
				if len(values) != 2 || values[0] != first || values[1] != second {
					t.Errorf("suppression lost exact references or order: %#v", values)
				}
			}
			assertStored()
			// A value copy retains one Throwable allocation state; snapshots are
			// independent, and failed self/null calls cannot mutate that state.
			copyOfPrimary := *primary
			if len(j.GetSuppressed(&copyOfPrimary)) != 2 {
				t.Error("value copy lost inherited suppression state")
			}
			snapshot := j.GetSuppressed(primary)
			if len(snapshot) > 0 {
				snapshot[0] = nil
			}
			assertStored()
			self := captureSuppressionPanic(func() {
				if route == "explicit" {
					j.JavaThrowableAddSuppressedExecution(j.NewExecution(), primary, &copyOfPrimary)
				} else {
					j.AddSuppressed(primary, &copyOfPrimary)
				}
			})
			if !j.CaughtAs(self, "IllegalArgumentException") || j.GetCause(self) != primary {
				t.Errorf("self rejection lost original cause reference: %T", self)
			}
			var absent *j.Exception
			null := captureSuppressionPanic(func() {
				j.JavaThrowableAddSuppressedExecution(j.NewExecution(), primary, absent)
			})
			if !j.CaughtAs(null, "NullPointerException") {
				t.Errorf("typed-null argument rejection: %T", null)
			}
			var missing *suppressionBusinessCollision
			nullReceiver := captureSuppressionPanic(func() {
				j.JavaThrowableAddSuppressedExecution(j.NewExecution(), missing, first)
			})
			if !j.CaughtAs(nullReceiver, "NullPointerException") {
				t.Errorf("typed-null receiver rejection: %T", nullReceiver)
			}
			assertStored()
			if primary.auditCalls != 1 || j.GetCause(primary) != nil {
				t.Error("failed calls changed source business method or cause state")
			}
		})
	}
}
