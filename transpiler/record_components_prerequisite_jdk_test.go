package transpiler

import "testing"

func TestRecordComponentsDynamicAPIsJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_components_prerequisite/dynamic-api", "records.DynamicProbe")
}

func TestRecordComponentsActualGsonHelperJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_components_prerequisite/gson-five-unit", "records.GsonRecordProbe")
}
