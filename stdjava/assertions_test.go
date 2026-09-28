package stdjava

import "testing"

func TestAssertionFailureMessagesAndCause(t *testing.T) {
	empty := NewAssertionFailure(nil)
	if !StringIsNull(empty.Message()) {
		t.Fatalf("no-detail message %q", empty.Message())
	}
	null := NewAssertionFailure(nil, nil)
	if null.Message() != "null" {
		t.Fatalf("null detail message %q", null.Message())
	}
	cause := NewIllegalArgumentException("detail")
	failure := NewAssertionFailure(nil, cause)
	if !JavaReferenceEqual(failure.CauseValue(), cause) {
		t.Fatal("Throwable detail cause lost")
	}
}
func TestAssertionStartupModeValidation(t *testing.T) {
	if parseJavaAssertionMode("") || parseJavaAssertionMode("false") || !parseJavaAssertionMode("true") {
		t.Fatal("assertion startup policy")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("invalid assertion mode accepted")
		}
	}()
	parseJavaAssertionMode("enabled")
}

func TestAssertionsReadPolicyOnlyAtStartup(t *testing.T) {
	before := JavaAssertionsEnabled()
	if before {
		t.Setenv("JAVA2GO_ASSERTIONS", "false")
	} else {
		t.Setenv("JAVA2GO_ASSERTIONS", "true")
	}
	if JavaAssertionsEnabled() != before {
		t.Fatal("assertion policy changed after startup")
	}
}
