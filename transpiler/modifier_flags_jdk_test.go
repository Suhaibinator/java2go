package transpiler

import "testing"

func TestModifierFlagsMatrixJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "modifier_flags/matrix", "flags.ModifierFlagsProbe")
}
func TestModifierFlagsSourceAndValueShadowJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "modifier_flags/shadow", "shadow.ShadowProbe")
}

func TestModifierFlagsReferenceAdaptationJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "modifier_flags/functional", "flags.FunctionalProbe")
}
