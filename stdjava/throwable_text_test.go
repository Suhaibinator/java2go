package stdjava_test

import (
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

func TestThrowableJavaTextBuiltinNamesAndEmptyMessage(t *testing.T) {
	e := j.NewExecution()
	cases := []struct {
		value j.Throwable
		want  string
	}{
		{j.NewThrowable(), "java.lang.Throwable"},
		{j.NewThrowable(""), "java.lang.Throwable: "},
		{j.NewException("detail"), "java.lang.Exception: detail"},
		{j.NewParseException("parse", 0), "java.text.ParseException: parse"},
	}
	for _, test := range cases {
		if got := j.ThrowableToStringExecution(e, test.value); got != test.want {
			t.Fatalf("%T: %q want %q", test.value, got, test.want)
		}
		if got := j.StringValueOfExecution(e, test.value); got != test.want {
			t.Fatalf("execution text conversion: %q want %q", got, test.want)
		}
		if got := j.StringValueOf(test.value); got != test.want {
			t.Fatalf("entry text conversion: %q want %q", got, test.want)
		}
	}
	if j.NewThrowable("").Error() != "Throwable" {
		t.Fatal("changed native Error formatting")
	}
	var absent *textMessageProbe
	defer func() {
		if !j.CaughtAs(recover(), "NullPointerException") {
			t.Error("typed-null toString receiver did not throw")
		}
	}()
	j.ThrowableToStringExecution(e, absent)
}

type textMessageProbe struct {
	j.ThrowableObject
	execution *j.Execution
	calls     int
	held      bool
	message   string
}

func (*textMessageProbe) JavaDynamicTypeID() j.TypeID { return "text.test.Message" }
func textMessageCallback(e *j.Execution, value any) string {
	p := value.(*textMessageProbe)
	if e != p.execution {
		panic("lost execution")
	}
	p.calls++
	p.held = j.ThreadHoldsLockExecution(e, p)
	return p.message
}

type textLocalizedProbe struct {
	j.ThrowableObject
	execution *j.Execution
	calls     int
	held      bool
	localized string
}

func (*textLocalizedProbe) JavaDynamicTypeID() j.TypeID { return "text.test.Localized" }
func textLocalizedCallback(e *j.Execution, value any) string {
	p := value.(*textLocalizedProbe)
	if e != p.execution {
		panic("lost execution")
	}
	p.calls++
	p.held = j.ThreadHoldsLockExecution(e, p)
	return p.localized
}

func TestThrowableJavaTextVirtualMessagesAndCause(t *testing.T) {
	j.RegisterJavaType("text.test.Message", j.ThrowableTypeID)
	j.RegisterJavaType("text.test.Localized", j.ThrowableTypeID)
	j.RegisterThrowableMessage((*textMessageProbe)(nil), textMessageCallback)
	j.RegisterThrowableLocalizedMessageOverride((*textLocalizedProbe)(nil), textLocalizedCallback)
	e := j.NewExecution()
	p := &textMessageProbe{ThrowableObject: j.NewThrowable("stored"), execution: e, message: "virtual"}
	guard := j.MonitorEnterExecution(e, p)
	defer j.MonitorExitExecution(guard)
	if got := j.ThrowableToStringExecution(e, p); got != "text.test.Message: virtual" || p.calls != 1 || !p.held {
		t.Fatalf("default toString message dispatch: %q calls%d held%t", got, p.calls, p.held)
	}
	wrapped := j.NewExceptionExecution(e, p)
	if wrapped.Message() != "text.test.Message: virtual" || j.GetCause(wrapped) != p || p.calls != 2 {
		t.Fatal("cause constructor bypassed default Java tostring")
	}
	p.message = j.NullString()
	if got := j.ThrowableToStringDefaultExecution(e, p); got != "text.test.Message" || p.calls != 3 {
		t.Fatal("null virtual message was not omitted")
	}
	localized := &textLocalizedProbe{ThrowableObject: j.NewThrowable("stored"), execution: e, localized: "localized"}
	second := j.MonitorEnterExecution(e, localized)
	defer j.MonitorExitExecution(second)
	if got := j.ThrowableToStringExecution(e, localized); got != "text.test.Localized: localized" || localized.calls != 1 || !localized.held {
		t.Fatal("localized override lost execution or dispatch")
	}
	if got := j.ThrowableLocalizedMessageDefaultExecution(e, localized); got != "stored" || localized.calls != 1 {
		t.Fatal("explicit super localized redispatched override")
	}
	localized.localized = ""
	if got := j.ThrowableToStringExecution(e, localized); got != "text.test.Localized: " {
		t.Fatal("empty localized message was omitted")
	}
}

type textOverrideProbe struct {
	j.ThrowableObject
	execution *j.Execution
	calls     int
	held      bool
	text      string
	abrupt    any
}

func (*textOverrideProbe) JavaDynamicTypeID() j.TypeID { return "text.test.Override" }
func textOverrideCallback(e *j.Execution, value any) string {
	p := value.(*textOverrideProbe)
	if e != p.execution {
		panic("lost execution")
	}
	p.calls++
	p.held = j.ThreadHoldsLockExecution(e, p)
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	return p.text
}
func TestThrowableJavaTextOverrideNullAbruptAndSuper(t *testing.T) {
	j.RegisterJavaType("text.test.Override", j.ThrowableTypeID)
	j.RegisterThrowableToStringOverride((*textOverrideProbe)(nil), textOverrideCallback)
	e := j.NewExecution()
	p := &textOverrideProbe{ThrowableObject: j.NewThrowable(), execution: e, text: "override"}
	guard := j.MonitorEnterExecution(e, p)
	defer j.MonitorExitExecution(guard)
	if got := j.ThrowableToStringExecution(e, p); got != "override" || p.calls != 1 || !p.held {
		t.Fatal("virtual tostring dispatch failed")
	}
	if got := j.ThrowableToStringDefaultExecution(e, p); got != "text.test.Override" || p.calls != 1 {
		t.Fatal("explicit super tostring redispatched override")
	}
	p.text = j.NullString()
	if !j.StringIsNull(j.ThrowableToStringExecution(e, p)) || p.calls != 2 {
		t.Fatal("null result normalized")
	}
	wrapped := j.NewThrowableExecution(e, p)
	if !j.StringIsNull(wrapped.Message()) || j.GetCause(wrapped) != p || p.calls != 3 {
		t.Fatal("cause constructor lost registered override")
	}
	marker := j.NewIllegalStateException("abrupt")
	p.abrupt = marker
	defer func() {
		if !j.JavaReferenceEqual(recover(), marker) || p.calls != 4 {
			t.Error("abrupt callback changed")
		}
	}()
	j.ThrowableToStringExecution(e, p)
}
