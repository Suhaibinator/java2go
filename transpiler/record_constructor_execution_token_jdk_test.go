package transpiler

import "testing"

func TestRecordConstructorTokenExecutionJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "record_constructor_execution_token", "records.ExecutionTokenProbe")
}
