package stdjava_test

import (
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

var _ j.Throwable = j.ThrowableObject{}

func TestThrowableObjectIdentityAndCauseInitialization(t *testing.T) {
	e := j.NewExecution()
	a, b := j.NewThrowableExecution(e), j.NewThrowableExecution(e)
	if j.JavaReferenceEqual(a, b) || !j.JavaReferenceEqual(a, a) || a.JavaDynamicTypeID() != j.ThrowableTypeID {
		t.Fatal("Throwable allocation or nominal identity changed")
	}
	if !j.StringIsNull(a.Message()) || j.GetCause(a) != nil {
		t.Fatal("no-arg Throwable state")
	}
	if !j.JavaReferenceEqual(j.ThrowableInitCauseExecution(e, a, b), a) || !j.JavaReferenceEqual(j.GetCause(a), b) {
		t.Fatal("initCause did not preserve references")
	}
	textNull := j.NewThrowableExecution(e, j.NullString())
	if !j.JavaReferenceEqual(j.ThrowableInitCauseExecution(e, textNull, b), textNull) || !j.JavaReferenceEqual(j.GetCause(textNull), b) {
		t.Fatal("null-message initCause did not preserve references")
	}
	causeNull := j.NewThrowableExecution(e, nil)
	if !j.StringIsNull(causeNull.Message()) || j.GetCause(causeNull) != nil {
		t.Fatal("null cause state")
	}
	defer func() {
		if !j.CaughtAs(recover(), "IllegalStateException") {
			t.Error("explicit-null cause allowed repeated initCause")
		}
	}()
	_ = j.ThrowableInitCauseExecution(e, causeNull, b)
}

type throwableObjectCause struct {
	j.ThrowableObject
	execution *j.Execution
	calls     int
	held      bool
	text      string
	abrupt    any
}

func (c *throwableObjectCause) StringJava2goExecution(e *j.Execution) string {
	if e != c.execution {
		panic("wrong calling Execution")
	}
	c.calls++
	c.held = j.ThreadHoldsLockExecution(e, c)
	if c.abrupt != nil {
		panic(c.abrupt)
	}
	return c.text
}
func TestThrowableObjectCauseCallback(t *testing.T) {
	e := j.NewExecution()
	cause := &throwableObjectCause{ThrowableObject: j.NewThrowable("detail"), execution: e, text: "virtual"}
	guard := j.MonitorEnterExecution(e, cause)
	defer j.MonitorExitExecution(guard)
	got := j.NewThrowableExecution(e, cause)
	if got.Message() != "virtual" || j.GetCause(got) != cause || cause.calls != 1 || !cause.held {
		t.Fatal("cause callback lost execution, identity or effects")
	}
	explicit := j.NewThrowableExecution(e, "fixed", cause)
	if explicit.Message() != "fixed" || j.GetCause(explicit) != cause || cause.calls != 1 {
		t.Fatal("message+cause constructor invoked callback")
	}
	cause.text = j.NullString()
	missing := j.NewThrowableExecution(e, cause)
	if !j.StringIsNull(missing.Message()) || cause.calls != 2 {
		t.Fatal("null override result changed")
	}
	marker := j.NewIllegalStateException("marker")
	cause.abrupt = marker
	defer func() {
		if !j.JavaReferenceEqual(recover(), marker) || cause.calls != 3 {
			t.Error("abrupt callback changed identity or count")
		}
	}()
	_ = j.NewThrowableExecution(e, cause)
}
