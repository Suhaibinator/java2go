package transpiler

import (
	"strings"
	"testing"
)

func TestGenericReceiverReturnTypeDrivesChainedStringIntrinsic(t *testing.T) {
	src := `
class Box<T> {
    private T value;
    Box(T value) { this.value = value; }
    public T value() { return this.value; }
}

public class App {
    public static boolean blocked(Box<String> box) {
        return box.value().startsWith("BLOCK:");
    }
    public static String run() {
        return blocked(new Box<String>("BLOCK:value")) + ":" + blocked(new Box<String>("allow"));
    }
}
`

	want := campaignRuntimeJavaOracle(t, "App", src)
	assertGeneratedLocalConstructorResult(t, src, want)
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, `strings.HasPrefix(stdjava.StringRequireNonNull(stdjava.ObjectView[string](box.ValueJava2goExecution(__java2goExecution), stdjava.StringTypeID)), "BLOCK:")`) {
		t.Fatalf("expected a class type-parameter return to resolve to String for the chained intrinsic, got:\n%s", out)
	}
}
