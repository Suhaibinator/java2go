package transpiler

import (
	"strings"
	"testing"
)

func TestBufferedReaderNullableReadLoopUsesPresenceBridge(t *testing.T) {
	src := `
import java.io.BufferedReader;
public class ReadLoopProgram {
    static int count(BufferedReader reader) throws Exception {
        String line;
        int count = 0;
        while ((line = reader.readLine()) != null) {
            count += line.length();
        }
        return count;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, normalizeSpaces(`for !stdjava.JavaReferenceEqual((func(dst **stdjava.JavaString, value *stdjava.JavaString) *stdjava.JavaString {
        *dst = value
        return value
    }(&line, reader.ReadLineReference())), nil) {`)) {
		t.Fatalf("expected nullable BufferedReader loop bridge, got:\n%s", out)
	}
	if !strings.Contains(flat, "var line *stdjava.JavaString") ||
		!strings.Contains(flat, "stdjava.RequireJavaString(line).Length()") {
		t.Fatalf("expected nullable String storage and checked loop-body length, got:\n%s", out)
	}
	if strings.Contains(out, "AssignmentExpression") {
		t.Fatalf("readLine loop must not retain assignment placeholder, got:\n%s", out)
	}
}
