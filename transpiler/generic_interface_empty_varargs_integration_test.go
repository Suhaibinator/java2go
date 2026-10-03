package transpiler

import (
	"strings"
	"testing"
)

func TestEmptyVarargsInvocation_MapsGenericInterfaceOwnerTypeArguments(t *testing.T) {
	out := renderGoFileFromJava(t, `
interface GenericDefaults<T> {
    default int size(T... values) {
        return values == null ? -1 : values.length;
    }
}

interface InheritedDefaults<U> extends GenericDefaults<U> {}

class StringDefaults implements InheritedDefaults<String> {}
class ObjectDefaults implements GenericDefaults<Object> {}

public class GenericInterfaceEmptyVarargsProgram {
    static <X extends GenericDefaults> int throughRawBound(X value) {
        return value.size();
    }

    public static String run() {
        StringDefaults concrete = new StringDefaults();
        InheritedDefaults<String> inherited = concrete;
        GenericDefaults<String> parent = concrete;
        return concrete.size() + ":" + inherited.size() + ":" + parent.size()
                + ":" + throughRawBound(new ObjectDefaults());
    }
}
`)
	if count := strings.Count(out, "stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID)"); count != 3 {
		t.Fatalf("empty generic-interface varargs calls emitted %d concrete arrays, want 3:\n%s", count, out)
	}
	if count := strings.Count(out, "stdjava.ReferenceArrayLiteralOf[any](stdjava.ObjectTypeID)"); count != 1 {
		t.Fatalf("raw generic-interface bound emitted %d erased arrays, want 1:\n%s", count, out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import (
    "slices"
    "testing"
    "unicode/utf16"
)

func TestGenericInterfaceEmptyVarargsBehavior(t *testing.T) {
    const want = "0:0:0:0"
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune(want))) {
        t.Fatalf("Run() = %v, want %q", got, want)
    }
}
`)
}
