package transpiler

import (
	"strings"
	"testing"
)

func TestLongToStringCanonicalBoundary(t *testing.T) {
	generated := renderGoFileFromJava(t, `public class LongTextBoundary {
 static String render(long value) { return Long.toString(value); }
 static String[] store(long value) { String text=Long.toString(value); return new String[]{text,text}; }
 static String qualified(long value) { return java.lang.Long.toString(value); }
}`)
	if count := strings.Count(generated, "stdjava.JavaStringValueOfLong("); count != 3 {
		t.Fatalf("canonical Long text conversions %d want3: %s", count, generated)
	}
	if strings.Contains(generated, "fmt.Sprint(") {
		t.Fatalf("canonical String result escaped to native text: %s", generated)
	}
}

func TestLongToStringSourceShadowBoundary(t *testing.T) {
	generated := renderGoFileFromJava(t, `class Long {
 static String toString(long value) { return "source"; }
}
public class LongTextShadow {
 static String local(long value) { return Long.toString(value); }
 static String qualified(long value) { return java.lang.Long.toString(value); }
}`)
	if count := strings.Count(generated, "stdjava.JavaStringValueOfLong("); count != 1 {
		t.Fatalf("only canonical qualified Long may lower: count%d %s", count, generated)
	}
	if !strings.Contains(generated, "return toStringJava2goExecution(") {
		t.Fatalf("source Long lost its method call: %s", generated)
	}
}
