package transpiler

import "testing"

func TestRecordConstructorBroaderInitializationJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_broader/initialization", "records.InitTokenProbe")
}

func TestRecordConstructorBroaderLifecycleJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_broader/lifecycle", "observe.Main")
}

func TestRecordConstructorBroaderReferencesJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_broader/references", "records.ReferenceProbe")
}
