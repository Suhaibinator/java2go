package transpiler

import "testing"

func TestRecordRetainedDelegatingConstructorJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_retained", "records.RecordProbe")
}
