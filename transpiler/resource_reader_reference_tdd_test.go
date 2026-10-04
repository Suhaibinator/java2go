package transpiler

import (
	"strings"
	"testing"
)

func TestCanonicalResourceReaderResourceNameABI(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.io.InputStream; public class ResourceABI { static InputStream read(Class<?> owner, String name) { return owner.getResourceAsStream(name); } }`)
	if !strings.Contains(out, ".GetResourceAsStreamReference(name)") {
		t.Fatalf("canonical resource name must stay a nullable Java String reference at the modeled Class boundary:\n%s", out)
	}
}

func TestCanonicalResourceReaderNullableLineABI(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.io.BufferedReader; public class ReaderABI { static String line(BufferedReader reader) throws Exception { return reader.readLine(); } }`)
	if !strings.Contains(out, ".ReadLineReference()") {
		t.Fatalf("readLine must expose a nullable Java String; empty content must remain distinct from EOF:\n%s", out)
	}
}

func TestCanonicalResourceReaderConstructorABI(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.io.*; public class ReaderCtorABI { static BufferedReader wrap(InputStream input) { return new BufferedReader(new InputStreamReader(input), 2); } }`)
	if !strings.Contains(out, "stdjava.NewBufferedReaderReference(") {
		t.Fatalf("canonical BufferedReader construction must retain sized buffering and closed-state ownership:\n%s", out)
	}
}

func TestCanonicalResourceReaderSourceReaderShadow(t *testing.T) {
	out := renderGoFileFromJava(t, `class BufferedReader { int readLine() { return 7; } } public class ReaderShadow { static int read(BufferedReader reader) { return reader.readLine(); } }`)
	if strings.Contains(out, "ReadLineReference") || strings.Contains(out, "NewBufferedReaderReference") {
		t.Fatalf("source declaration must retain its own readLine dispatch:\n%s", out)
	}
}

func TestCanonicalResourceReaderSourceClassShadow(t *testing.T) {
	out := renderGoFileFromJava(t, `class Class { int getResourceAsStream(String name) { return 7; } } public class ResourceShadow { static int read(Class owner, String name) { return owner.getResourceAsStream(name); } }`)
	if strings.Contains(out, "GetResourceAsStreamReference") {
		t.Fatalf("source Class declaration must retain its own resource dispatch:\n%s", out)
	}
}
