package transpiler

import "testing"

// The published primitive result contract drives argument applicability and
// boxing even when its invocation representation becomes a native SAM object.
func TestPrimitiveSAMNativeResultMetadataPreservedTDD(t *testing.T) {
	got := instanceIntrinsicResultTypes[intrinsicKey{"IntBinaryOperator", "applyAsInt"}]
	if got != "int" {
		t.Fatalf("published primitive SAM result metadata lost after native precedence: got %q, want int", got)
	}
	signature := builtinFunctionalInterfaces["IntBinaryOperator"]
	if signature.resultType != "int" || len(signature.parameterTypes) != 2 || signature.parameterTypes[0] != "int" || signature.parameterTypes[1] != "int" {
		t.Fatalf("published primitive SAM width signature changed: %+v", signature)
	}
}
