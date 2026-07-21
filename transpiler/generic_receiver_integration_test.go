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
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, `strings.HasPrefix(box.Value(), "BLOCK:")`) {
		t.Fatalf("expected a class type-parameter return to resolve to String for the chained intrinsic, got:\n%s", out)
	}
}
