package transpiler

import "testing"

func TestReflectionProtectedConstructorCallerJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_core_access_differential/protected-constructor", "q.Sub")
}
func TestReflectionTypedLongCallerJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_core_access_differential/long-caller", "fields.LongCaller")
}
