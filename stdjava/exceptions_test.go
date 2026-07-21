package stdjava

import (
	"fmt"
	"testing"
)

// recoverNormalized runs fn, recovers any panic it raises, and returns the
// value after NormalizePanic, mimicking the generated recover boundary.
func recoverNormalized(fn func()) (out interface{}) {
	defer func() {
		out = NormalizePanic(recover())
	}()
	fn()
	return nil
}

func TestNormalizePanic_DivideByZero(t *testing.T) {
	got := recoverNormalized(func() {
		a, b := 1, 0
		_ = a / b
	})
	if !CaughtAs(got, "ArithmeticException") {
		t.Fatalf("divide by zero should normalize to ArithmeticException, got %T (%v)", got, got)
	}
}

func TestNormalizePanic_NilPointerDereference(t *testing.T) {
	got := recoverNormalized(func() {
		var p *int
		_ = *p
	})
	if !CaughtAs(got, "NullPointerException") {
		t.Fatalf("nil dereference should normalize to NullPointerException, got %T (%v)", got, got)
	}
}

func TestNormalizePanic_IndexOutOfRange(t *testing.T) {
	got := recoverNormalized(func() {
		s := []int{1, 2, 3}
		idx := 5
		_ = s[idx]
	})
	if !CaughtAs(got, "ArrayIndexOutOfBoundsException") {
		t.Fatalf("index out of range should normalize to ArrayIndexOutOfBoundsException, got %T (%v)", got, got)
	}
	// It must also be catchable by the supertype.
	if !CaughtAs(got, "IndexOutOfBoundsException") || !CaughtAs(got, "RuntimeException") {
		t.Fatal("normalized index error should be catchable by supertypes")
	}
}

func TestNormalizePanic_NegativeSliceLength(t *testing.T) {
	got := recoverNormalized(func() {
		n := int32(-1)
		_ = make([]int, n)
	})
	if !CaughtAs(got, "NegativeArraySizeException") {
		t.Fatalf("negative make length should normalize to NegativeArraySizeException, got %T (%v)", got, got)
	}
	if !CaughtAs(got, "RuntimeException") {
		t.Fatal("normalized negative array size should be catchable by RuntimeException")
	}
}

func TestNormalizePanic_FailedTypeAssertion(t *testing.T) {
	got := recoverNormalized(func() {
		var v interface{} = "a string"
		_ = v.(int)
	})
	if !CaughtAs(got, "ClassCastException") {
		t.Fatalf("failed type assertion should normalize to ClassCastException, got %T (%v)", got, got)
	}
}

func TestNormalizePanic_ThrowablePassesThrough(t *testing.T) {
	orig := NewIllegalArgumentException("explicit")
	got := NormalizePanic(orig)
	if !CaughtAs(got, "IllegalArgumentException") {
		t.Fatal("an explicit stdjava exception must pass through normalization unchanged")
	}
	if GetMessage(got) != "explicit" {
		t.Fatalf("message lost during normalization: %q", GetMessage(got))
	}
}

func TestNormalizePanic_NilIsNil(t *testing.T) {
	if NormalizePanic(nil) != nil {
		t.Fatal("normalizing a nil (no panic) must yield nil")
	}
}

func TestNormalizePanic_PlainStringWrappedAsRuntimeException(t *testing.T) {
	got := NormalizePanic("custom panic")
	if !CaughtAs(got, "RuntimeException") {
		t.Fatalf("a non-runtime panic value should be wrapped as RuntimeException, got %T (%v)", got, got)
	}
	if GetMessage(got) != "custom panic" {
		t.Fatalf("wrapped message lost: %q", GetMessage(got))
	}
}

func TestNormalizePanic_PlainErrorWrappedAsRuntimeException(t *testing.T) {
	got := NormalizePanic(fmt.Errorf("boom"))
	if !CaughtAs(got, "RuntimeException") {
		t.Fatalf("a panicked error should be wrapped as RuntimeException, got %T (%v)", got, got)
	}
}

func TestExceptionFidelity_ErrorNotCaughtByException(t *testing.T) {
	// In Java, catch (Exception e) must NOT catch an Error/Throwable-level throw.
	err := NewAssertionError("assertion failed")
	if CaughtAs(err, "Exception") {
		t.Fatal("AssertionError must NOT be caught by catch (Exception e)")
	}
	if CaughtAs(err, "RuntimeException") {
		t.Fatal("AssertionError must NOT be caught by catch (RuntimeException e)")
	}
	// It IS catchable by Error and Throwable.
	if !CaughtAs(err, "Error") {
		t.Fatal("AssertionError should be caught by catch (Error e)")
	}
	if !CaughtAs(err, "Throwable") {
		t.Fatal("AssertionError should be caught by catch (Throwable e)")
	}
}

func TestRegisterException_ConflictingParentWarns(t *testing.T) {
	// Registering the same simple name with a different parent should not panic
	// and should leave the most recent registration in effect (a warning is
	// printed to stderr, which we don't assert on here).
	RegisterException("CollidingName", "RuntimeException")
	RegisterException("CollidingName", "IOException")
	v := ThrowableBase{typeName: "CollidingName", message: "x"}
	if !CaughtAs(v, "IOException") {
		t.Fatal("most recent registration should win for a colliding simple name")
	}
}

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

func TestCaughtAs_NegativeArraySizeException(t *testing.T) {
	e := NewNegativeArraySizeException("negative dimension")
	for _, super := range []string{"NegativeArraySizeException", "RuntimeException", "Exception", "Throwable"} {
		if !CaughtAs(e, super) {
			t.Fatalf("NegativeArraySizeException should be caught by %s", super)
		}
	}
	if CaughtAs(e, "IllegalArgumentException") {
		t.Fatal("NegativeArraySizeException must not match an unrelated sibling type")
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
