package transpiler

import (
	"strings"
	"testing"
)

func TestOperatorAndThenDeclaredParentInferenceTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `import java.util.function.*;public class AfterControl {static class Named implements Function<String,String>{public String apply(String s){return s+"#";}}Function<String,String> unary(Function<String,String> f,UnaryOperator<String> u){return f.andThen(u);}Function<String,String> source(Function<String,String> f,Named n){return f.andThen(n);}}`)
	if strings.Contains(generated, "stdjava.Function[*stdjava.JavaString, any]") {
		t.Fatalf("declared Function parent result incorrectly inferred as Object\n%s", generated)
	}
	if strings.Count(generated, "FunctionAndThenExecution[") != 2 {
		t.Fatalf("resolved parent default dispatch missing\n%s", generated)
	}
}
func TestOperatorComposeDeclaredParentInferenceTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `import java.util.function.*;public class BeforeControl {static class Named implements Function<String,String>{public String apply(String s){return s+"#";}} String unary(Function<String,String> f,UnaryOperator<String> u){return f.compose(u).apply("a");}String source(Function<String,String> f,Named n){return f.compose(n).apply("b");}}`)
	if strings.Contains(generated, "stdjava.Function[any, *stdjava.JavaString]") {
		t.Fatalf("declared Function parent input incorrectly inferred as Object\n%s", generated)
	}
	if strings.Count(generated, "FunctionComposeExecution[") != 2 {
		t.Fatalf("resolved parent composition dispatch missing\n%s", generated)
	}
}
