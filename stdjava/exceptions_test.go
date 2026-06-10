package stdjava

import "testing"

func TestCaughtAs_ExactMatch(t *testing.T) {
	e := NewIllegalArgumentException("bad arg")
	if !CaughtAs(e, "IllegalArgumentException") {
		t.Fatal("expected exact-type catch to match")
	}
}

func TestCaughtAs_BySupertype(t *testing.T) {
	e := NewIllegalArgumentException("bad arg")
	if !CaughtAs(e, "RuntimeException") {
		t.Fatal("IllegalArgumentException should be caught by RuntimeException")
	}
	if !CaughtAs(e, "Exception") {
		t.Fatal("IllegalArgumentException should be caught by Exception")
	}
	if !CaughtAs(e, "Throwable") {
		t.Fatal("IllegalArgumentException should be caught by Throwable")
	}
}

func TestCaughtAs_BySubtypeDoesNotMatch(t *testing.T) {
	e := NewRuntimeException("boom")
	if CaughtAs(e, "IllegalArgumentException") {
		t.Fatal("RuntimeException must not be caught by IllegalArgumentException")
	}
}

func TestCaughtAs_DeepHierarchy(t *testing.T) {
	e := NewArrayIndexOutOfBoundsException("idx")
	for _, super := range []string{"IndexOutOfBoundsException", "RuntimeException", "Exception", "Throwable"} {
		if !CaughtAs(e, super) {
			t.Fatalf("ArrayIndexOutOfBoundsException should be caught by %s", super)
		}
	}
}

func TestCaughtAs_NilNeverMatches(t *testing.T) {
	if CaughtAs(nil, "Throwable") {
		t.Fatal("nil recovered value must not match any catch")
	}
}

func TestGetMessage(t *testing.T) {
	if got := GetMessage(NewIllegalStateException("nope")); got != "nope" {
		t.Fatalf("GetMessage = %q, want %q", got, "nope")
	}
}

func TestErrorRendering(t *testing.T) {
	if got := NewNumberFormatException("not a number").Error(); got != "NumberFormatException: not a number" {
		t.Fatalf("Error() = %q", got)
	}
	if got := NewException("").Error(); got != "Exception" {
		t.Fatalf("Error() for empty message = %q", got)
	}
}

func TestRegisterException_UserDefined(t *testing.T) {
	RegisterException("MyAppException", "RuntimeException")
	user := ThrowableBase{typeName: "MyAppException", message: "custom"}
	if !CaughtAs(user, "RuntimeException") {
		t.Fatal("registered user exception should be caught by its supertype")
	}
	if !CaughtAs(user, "Exception") {
		t.Fatal("registered user exception should be caught transitively")
	}
	if CaughtAs(user, "IllegalArgumentException") {
		t.Fatal("user exception must not match an unrelated sibling type")
	}
}
