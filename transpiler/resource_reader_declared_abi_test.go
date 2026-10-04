package transpiler

import (
	"strings"
	"testing"
)

func TestCanonicalResourceReaderDeclaredPointerABI(t *testing.T) {
	out := renderGoFileFromJava(t, `public class DeclaredReader { static java.io.BufferedReader identity(java.io.BufferedReader reader) { return reader; } static String line(java.io.BufferedReader reader) throws Exception { return reader.readLine(); } }`)
	if !strings.Contains(out, "reader *stdjava.BufferedReader") || !strings.Contains(out, ") *stdjava.BufferedReader") || !strings.Contains(out, ".ReadLineReference()") {
		t.Fatalf("qualified canonical owner must govern parameters, result, and selector:\n%s", out)
	}
}

func TestCanonicalResourceReaderForeignOwnerExcluded(t *testing.T) {
	if _, mapped := digestIORuntimeTypeExpr("other.io.BufferedReader", Ctx{}); mapped {
		t.Fatal("unregistered qualified owner must not acquire java.io.BufferedReader storage")
	}
}
