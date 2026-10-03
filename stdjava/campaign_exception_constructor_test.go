package stdjava

import "testing"

func TestCampaignExceptionConstructors(t *testing.T) {
	cause := NewIllegalArgumentException("failure")
	for _, makeException := range []func() Exception{
		func() Exception { return NewException("decode", cause) },
		func() Exception { return NewException(cause) },
	} {
		exception := makeException()
		if !JavaReferenceEqual(GetCause(exception), cause) {
			t.Fatal("constructor lost cause identity")
		}
	}
	if got := NewException(cause).Message(); got != "java.lang.IllegalArgumentException: failure" {
		t.Fatalf("cause message: %q", got)
	}
	if got := NewException(NewIllegalArgumentException("")).Message(); got != "java.lang.IllegalArgumentException: " {
		t.Fatalf("empty cause message: %q", got)
	}
	for _, exception := range []Exception{NewException(), NewException(nil), NewException(NullString(), nil)} {
		if !StringIsNull(exception.Message()) || GetCause(exception) != nil {
			t.Fatal("null constructor state changed")
		}
		if exception.Error() != "Exception" {
			t.Fatal("null sentinel leaked into error formatting")
		}
	}
	if got := NewException("").Message(); got != "" {
		t.Fatalf("empty message changed: %q", got)
	}
}
