package transpiler

import "testing"

func TestReflectionCallbackHygieneOrdinaryConvertedJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/ordinary-converted", "hygiene.Probe")
}

func TestReflectionCallbackHygieneOrdinaryExecutionJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/ordinary-execution", "hygiene.Probe")
}

func TestReflectionCallbackHygieneOrdinaryValueJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/ordinary-value", "hygiene.Probe")
}

func TestReflectionCallbackHygieneVolatileConvertedJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/volatile-converted", "hygiene.Probe")
}

func TestReflectionCallbackHygieneVolatileExecutionJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/volatile-execution", "hygiene.Probe")
}

func TestReflectionCallbackHygieneVolatileValueJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "reflection_callback_hygiene/volatile-value", "hygiene.Probe")
}
