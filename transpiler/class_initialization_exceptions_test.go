package transpiler

import "testing"

func TestClassInitializationErrorsUseBuiltinRuntimeTypes(t *testing.T) {
	for _, javaType := range []string{
		"LinkageError",
		"java.lang.ExceptionInInitializerError",
		"NoClassDefFoundError",
	} {
		if !isBuiltinExceptionType(javaType) {
			t.Fatalf("%s was not routed through the stdjava exception runtime", javaType)
		}
	}
}
