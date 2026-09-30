package transpiler

import (
	"strings"
	"testing"
)

func TestJavaPathReferenceCanonicalStringAndArrayDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class CanonicalPathDispatch {
    static java.nio.file.Path single(String first) { return java.nio.file.Paths.get(first); }
    static java.nio.file.Path expanded(String first, String one, String two) { return java.nio.file.Paths.get(first, one, two); }
    static java.nio.file.Path array(String first, String[] parts) { return java.nio.file.Paths.get(first, parts); }
    static java.nio.file.Path nullArray(String first) { return java.nio.file.Paths.get(first, (String[])null); }
    static String text(java.nio.file.Path value) { return value.toString(); }
}`)
	for _, name := range []string{"PathsGetReference", "PathsGetArrayReference", "PathToStringReference"} {
		if !strings.Contains(out, "stdjava."+name+"(") {
			t.Errorf("missing canonical path dispatch %s:\n%s", name, out)
		}
	}
	if strings.Contains(out, "stdjava.PathsGet(") {
		t.Errorf("Java Paths.get selected native Go String overload:\n%s", out)
	}
}

func TestJavaIOExceptionReferenceConstructorDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class CanonicalIOExceptionDispatch {
    static java.io.IOException empty() { return new java.io.IOException(); }
    static java.io.IOException message(String text) { return new java.io.IOException(text); }
    static java.io.IOException nullMessage() { return new java.io.IOException((String)null); }
    static java.io.IOException cause(Throwable cause) { return new java.io.IOException(cause); }
    static java.io.IOException nullCause() { return new java.io.IOException((Throwable)null); }
    static java.io.IOException both(String text, Throwable cause) { return new java.io.IOException(text, cause); }
}`)
	for _, name := range []string{"NewJavaIOExceptionMessage", "NewJavaIOExceptionCauseExecution", "NewJavaIOExceptionMessageCause"} {
		if !strings.Contains(out, "stdjava."+name+"(") {
			t.Errorf("missing canonical IOException constructor %s:\n%s", name, out)
		}
	}
	if strings.Contains(out, "stdjava.NewIOException(") {
		t.Errorf("Java IOException selected native Go text constructor:\n%s", out)
	}
}
