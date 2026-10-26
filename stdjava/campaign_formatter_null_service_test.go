package stdjava

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type campaignFormatterNullProbe struct {
	calls  int
	marker any
}

func (*campaignFormatterNullProbe) JavaDynamicTypeID() TypeID { return "campaign.FormatterNullJDK21" }
func (p *campaignFormatterNullProbe) FormatNullText(*Execution) *JavaString {
	p.calls++
	panic(p.marker)
}

// The message and zero-callback outcome come from the frozen actual JDK21
// adversarial stream. Cause availability is additionally checked against the
// installed JDK21 Throwable contract; the null record itself does not print cause.
func TestCampaignStringFormatNullFormatActualJDK21Service(t *testing.T) {
	var oracle struct {
		ExceptionClass string   `json:"exception_class"`
		Message        []uint16 `json:"message_utf16"`
		Calls          []int    `json:"callback_calls"`
	}
	data, err := os.ReadFile("testdata/formatter_null_actual_jdk21.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if oracle.ExceptionClass != "java.lang.NullPointerException" || len(oracle.Message) != 51 || !reflect.DeepEqual(oracle.Calls, []int{0, 0, 0, 0}) {
		t.Fatal("frozen actual oracle shape drift")
	}
	RegisterJavaType("campaign.FormatterNullJDK21", ObjectTypeID)
	RegisterJavaSourceType("campaign.FormatterNullJDK21")
	RegisterJavaSourceToString("campaign.FormatterNullJDK21", "FormatNullText")
	cases := []struct {
		name      string
		locale    *Locale
		localized bool
		nilArgs   bool
	}{
		{name: "default_callbacks"}, {name: "default_null_array", nilArgs: true}, {name: "root_callbacks", locale: LocaleROOT, localized: true}, {name: "null_locale_null_array", localized: true, nilArgs: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			execution := NewExecution()
			marker := NewIllegalStateException("callback-must-not-run")
			probe := &campaignFormatterNullProbe{marker: marker}
			args := ReferenceArrayLiteral(ObjectTypeID, probe)
			if c.nilArgs {
				args = nil
			}
			var got any
			heldAfterCatch := false
			func() {
				guard := MonitorEnterExecution(execution, probe)
				defer MonitorExitExecution(guard)
				func() {
					defer func() { got = recover() }()
					if c.localized {
						JavaStringFormatLocaleExecution(execution, c.locale, nil, args)
					} else {
						JavaStringFormatExecution(execution, nil, args)
					}
				}()
				heldAfterCatch = ThreadHoldsLockExecution(execution, probe)
			}()
			if _, ok := got.(NullPointerException); !ok || !CaughtAs(got, "RuntimeException") {
				t.Fatalf("exception type %T; want actual JDK21 NPE", got)
			}
			message := JavaThrowableMessageExecution(execution, got)
			if message == nil || !reflect.DeepEqual(message.UTF16Copy(), oracle.Message) {
				t.Fatalf("message units %v; want frozen actual units %v", message, oracle.Message)
			}
			if probe.calls != 0 {
				t.Fatalf("validation rendered callback %d times", probe.calls)
			}
			if !heldAfterCatch || ThreadHoldsLockExecution(execution, probe) {
				t.Fatal("exception changed held monitor or failed to release it")
			}
			guard := MonitorEnterExecution(execution, probe)
			if !ThreadHoldsLockExecution(execution, probe) {
				t.Fatal("released monitor could not be reacquired")
			}
			MonitorExitExecution(guard)
			if GetCause(got) != nil {
				t.Fatalf("fresh JDK NPE cause %v", GetCause(got))
			}
			returned := ThrowableInitCauseExecution(execution, got, marker)
			if !JavaReferenceEqual(returned, got) || !JavaReferenceEqual(GetCause(got), marker) {
				t.Fatal("fresh null-format NPE lost initCause availability or identity")
			}
		})
	}
}
