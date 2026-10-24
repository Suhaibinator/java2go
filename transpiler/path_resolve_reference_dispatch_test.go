package transpiler

import (
	"strings"
	"testing"
)

func TestJavaPathResolveReferenceStringAndPathDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class CanonicalPathResolveDispatch {
    static java.nio.file.Path text(java.nio.file.Path base, String other) { return base.resolve(other); }
    static java.nio.file.Path path(java.nio.file.Path base, java.nio.file.Path other) { return base.resolve(other); }
    static java.nio.file.Path nullText(java.nio.file.Path base) { return base.resolve((String)null); }
    static java.nio.file.Path nullPath(java.nio.file.Path base) { return base.resolve((java.nio.file.Path)null); }
    static boolean absolute(java.nio.file.Path value) { return value.isAbsolute(); }
}`)
	for _, alias := range []string{"PathResolveStringReference", "PathResolvePathReference", "PathIsAbsoluteReference"} {
		if !strings.Contains(out, "stdjava."+alias+"(") {
			t.Errorf("missing audited canonical path selector %s:\n%s", alias, out)
		}
	}
	if strings.Contains(out, ".Resolve(") {
		t.Errorf("canonical Path.resolve still selected the native any overload:\n%s", out)
	}
}
