package transpiler

import (
	"strings"
	"testing"
)

func TestTypeInfoIntegration_InstanceofExpression(t *testing.T) {
	src := `
package typeinfo.instanceof;
public interface Animal {}
public class Dog implements Animal {}
public class Check {
    public static boolean isAnimal(Object value) {
        return value instanceof Animal;
    }
    public static boolean isDog(Object value) {
        return value instanceof Dog;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "_, ok := any(value).(Animal)") {
		t.Fatalf("expected interface instanceof to lower to an interface type assertion, got:\n%s", out)
	}
	if !strings.Contains(flat, "_, ok := any(value).(*Dog)") {
		t.Fatalf("expected class instanceof to lower to a pointer type assertion, got:\n%s", out)
	}
}

func TestTypeInfoIntegration_LambdaTypesFromVariableDeclaration(t *testing.T) {
	src := `
package typeinfo.lambda.var;
public interface Mapper<T, R> { R apply(T value); }
public class Demo {
    public static void run() {
        Mapper<String, String> m = v -> v;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "func(__java2goExecution *stdjava.Execution, v string)") {
		t.Fatalf("expected lambda parameter type to be inferred from declaration type, got:\n%s", out)
	}
	if !strings.Contains(out, "func(__java2goExecution *stdjava.Execution, v string) string") {
		t.Fatalf("expected lambda return type to be inferred from declaration type, got:\n%s", out)
	}
	if !strings.Contains(out, "NewMapperFuncAdapterJava2goExecution[string, string]") {
		t.Fatalf("expected lambda to be wrapped in functional interface adapter, got:\n%s", out)
	}
}

func TestTypeInfoIntegration_LambdaTypesFromMethodArgument(t *testing.T) {
	src := `
package typeinfo.lambda.arg;
public interface Mapper<T, R> { R apply(T value); }
public class Demo {
    public static void accept(Mapper<String, String> mapper) {}
    public static void run() {
        accept(v -> v);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "func(__java2goExecution *stdjava.Execution, v string) string") {
		t.Fatalf("expected lambda parameter type to be inferred from method argument type, got:\n%s", out)
	}
	if !strings.Contains(out, "func Accept(mapper Mapper[string, string])") {
		t.Fatalf("expected interface-typed parameter without pointer, got:\n%s", out)
	}
	if !strings.Contains(out, "AcceptJava2goExecution(__java2goExecution, NewMapperFuncAdapterJava2goExecution[string, string]") {
		t.Fatalf("expected lambda argument to be wrapped in functional interface adapter, got:\n%s", out)
	}
	if !strings.Contains(out, "type MapperFuncAdapter[T any, R any] struct") {
		t.Fatalf("expected generated adapter type for functional interface, got:\n%s", out)
	}
}
