package transpiler

import "testing"

func TestRecordConstructorContractCompactJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_contracts/compact", "records.CompactProbe")
}

func TestRecordConstructorContractExplicitCanonicalJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_contracts/explicit-canonical", "records.ExplicitCanonicalProbe")
}

func TestRecordConstructorContractSameArityJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_contracts/same-arity", "records.SameArityProbe")
}

func TestRecordConstructorContractParameterCountJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_contracts/parameter-count", "records.ParameterCountProbe")
}

func TestRecordConstructorContractReflectiveCanonicalJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_contracts/reflective-canonical", "records.ReflectiveCanonicalProbe")
}
