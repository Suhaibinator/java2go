package stdjava

import "testing"

func TestEnumCloneCanonicalFailure(t *testing.T) {
	var previous *throwableState
	for i := 0; i < 2; i++ {
		func() {
			defer func() {
				value := recover()
				failure, ok := value.(CloneNotSupportedException)
				if !ok {
					t.Fatalf("Enum.clone failure type %T", value)
				}
				if failure.JavaDynamicTypeID() != "java.lang.CloneNotSupportedException" || !CaughtAs(failure, "Exception") {
					t.Fatalf("wrong descriptor/hierarchy: %v", failure)
				}
				if failure.state == previous {
					t.Fatal("Enum.clone reused throwable state")
				}
				previous = failure.state
				if failure.state == nil || !failure.state.canonicalMessage || JavaThrowableMessageDefault(failure) != nil || !StringIsNull(failure.Message()) {
					t.Fatal("Enum.clone must retain canonical null message")
				}
			}()
			EnumCloneExecution(NewExecution())
			t.Fatal("Enum.clone returned")
		}()
	}
}
func TestEnumCloneRequiresExecution(t *testing.T) {
	defer func() {
		if _, ok := recover().(IllegalArgumentException); !ok {
			t.Fatal("nil execution must fail before enum clone body")
		}
	}()
	EnumCloneExecution(nil)
	t.Fatal("nil execution accepted")
}
