package stdjava

import (
	"slices"
	"testing"
)

func TestJavaCancellationExceptionReferenceMessageNullIdentity(t *testing.T) {
	execution := NewExecution()
	units := []uint16{'r', 0, 0xd800, 'x', 0xdc00}
	message := NewJavaStringUTF16(units)
	for _, input := range []*JavaString{nil, NewJavaStringUTF16(nil), message} {
		value := NewJavaCancellationExceptionMessage(input)
		other := NewJavaCancellationExceptionMessage(input)
		if JavaThrowableMessageDefault(value) != input || GetCause(value) != nil || JavaReferenceEqual(value, other) {
			t.Fatal("message/null/new identity changed")
		}
		copy := value
		if !JavaReferenceEqual(copy, value) {
			t.Fatal("copy lost Java identity")
		}
		cause := NewJavaExceptionMessage(nil)
		if !JavaReferenceEqual(ThrowableInitCauseExecution(execution, copy, cause), value) || GetCause(value) != cause {
			t.Fatal("message constructor cause slot/reference changed")
		}
		failure := nsaConstructorReferencePanic(t, func() { _ = ThrowableInitCauseExecution(execution, value, cause) })
		if !CaughtAs(failure, "IllegalStateException") {
			t.Fatal("second cause initialization accepted")
		}
	}
	if !slices.Equal(message.UTF16Copy(), units) {
		t.Fatal("message UTF16 changed")
	}
	if NewCancellationException("native").Message() != "native" {
		t.Fatal("legacy native Cancellation constructor changed")
	}
}
func TestJavaSecurityExceptionReferenceNominalHierarchy(t *testing.T) {
	nsa := NewJavaNoSuchAlgorithmExceptionMessage(nil)
	gse := NewJavaGeneralSecurityExceptionMessage(nil)
	for _, parent := range []string{"NoSuchAlgorithmException", "GeneralSecurityException", "Exception", "Throwable"} {
		if !CaughtAs(nsa, parent) {
			t.Fatalf("NSA ancestry missing %s", parent)
		}
	}
	if CaughtAs(nsa, "RuntimeException") || CaughtAs(gse, "NoSuchAlgorithmException") {
		t.Fatal("security hierarchy widened")
	}
	if BuiltinThrowableTypeID("GeneralSecurityException") != TypeID("java.security.GeneralSecurityException") {
		t.Fatal("security descriptor lost canonical owner")
	}
}
