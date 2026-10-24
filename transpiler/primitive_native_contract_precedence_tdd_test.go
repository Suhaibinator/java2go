package transpiler

import (
	"go/ast"
	"testing"
)

func TestPrimitiveSAMNativeTypeContractPrecedenceTDD(t *testing.T) {
	typ := javaTypeStringToGoTypeExpr("java.util.function.IntBinaryOperator", nil, Ctx{})
	selected, ok := typ.(*ast.SelectorExpr)
	if !ok || selected.Sel.Name != "IntBinaryOperator" {
		t.Fatalf("native SAM type contract preempted by primitive raw function: %T", typ)
	}
}

func TestPrimitiveSAMNativeInvocationContractPrecedenceTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.IntBinaryOperator;public class PrimitiveNativeContractProbe{int apply(IntBinaryOperator value){return value.applyAsInt(3,4);}}`)
	calls := updaterTDDCalls(file)
	if calls["CallIntBinaryOperatorExecution"] != 1 {
		t.Fatalf("native SAM invocation contract preempted by primitive raw function: nativeCalls=%d\n%s", calls["CallIntBinaryOperatorExecution"], generated)
	}
}

func TestPrimitiveSAMNativeSourceShadowDeclinesTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `public class PrimitiveNativeShadowProbe{static class IntBinaryOperator{int applyAsInt(int a,int b){return a-b;}} int apply(IntBinaryOperator value){return value.applyAsInt(3,4);}}`)
	if updaterTDDCalls(file)["CallIntBinaryOperatorExecution"] != 0 {
		t.Fatalf("source shadow was admitted to native SAM contract\n%s", generated)
	}
}

func TestPrimitiveSAMNativeForeignOwnerDeclinesTDD(t *testing.T) {
	if nativeFunctionalFamily("foreign.IntBinaryOperator", Ctx{}) != "" {
		t.Fatal("foreign owner admitted to native SAM contract")
	}
	if typ := primitiveOperatorTypeExpr("foreign.IntBinaryOperator", Ctx{}); typ != nil {
		t.Fatalf("foreign owner admitted to primitive fallback: %T", typ)
	}
}
