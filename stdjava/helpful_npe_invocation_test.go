package stdjava

import "testing"

func TestInvocationReceiverTypedIdentityAndNullDiagnostic(t *testing.T) {
	type receiver struct{ value int }
	valid := &receiver{7}
	if RequireInvocationReceiver(valid, "unused") != valid {
		t.Fatal("nonnull exact receiver changed")
	}
	for _, absent := range []any{nil, (*receiver)(nil)} {
		func() {
			defer func() {
				caught := recover()
				e, ok := caught.(NullPointerException)
				if !ok || e.Message() != "owned diagnostic" || !CaughtAs(e, "RuntimeException") || NormalizePanic(e) != caught {
					t.Fatalf("null receiver lost canonical Throwable/identity: %#v", caught)
				}
			}()
			RequireInvocationReceiver(absent, "owned diagnostic")
			t.Fatal("null receiver returned")
		}()
	}
}
func TestInvocationReceiverAbruptArgumentIdentity(t *testing.T) {
	marker := NewIllegalArgumentException("argument")
	trace := 0
	var receiver *int
	argument := func() int { trace++; panic(marker) }
	defer func() {
		caught := recover()
		if caught != marker || NormalizePanic(caught) != marker || trace != 1 {
			t.Fatal("argument failure replaced by receiver null exception")
		}
	}()
	func(r *int, a int) { RequireInvocationReceiver(r, "must not run") }(receiver, argument())
}
