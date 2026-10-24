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
	ci199AssertGenericReceiverContract(t, strings.TrimSpace(out))
}
