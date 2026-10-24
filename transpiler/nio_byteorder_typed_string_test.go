package transpiler

import "testing"

// Returning, forwarding and erasing a JDK String must retain the canonical
// reference ABI; this regression exercises ByteOrder's actual invocation result.
func TestNIOByteOrderTypedJavaStringInvocation(t *testing.T) {
	const source = `import java.nio.ByteBuffer;
import java.nio.ByteOrder;
public class TypedOrderContract {
    public static String typed(ByteOrder value) { return value.toString(); }
    public static String chained() { return ByteBuffer.allocate(0).order(ByteOrder.LITTLE_ENDIAN).order().toString(); }
    public static String forward(String value) { return value; }
    public static String argument() { return forward(ByteOrder.BIG_ENDIAN.toString()); }
    public static Object erased() { return ByteOrder.BIG_ENDIAN.toString(); }
}`
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, `package main
import (
    "slices"
    "testing"
    "unicode/utf16"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestTypedOrderReferenceResult(t *testing.T) {
    var big *j.JavaString = Typed(j.ByteOrderBIG_ENDIAN)
    var little *j.JavaString = Typed(j.ByteOrderLITTLE_ENDIAN)
    for _, item := range []struct { value *j.JavaString; label string }{
        {big, "BIG_ENDIAN"}, {little, "LITTLE_ENDIAN"},
    } {
        units := utf16.Encode([]rune(item.label))
        if item.value == nil || !slices.Equal(item.value.UTF16Copy(), units) {
            t.Fatal("typed ByteOrder.toString result lost canonical UTF16")
        }
        if item.value != j.JavaStringLiteralUTF16(units) {
            t.Fatal("typed ByteOrder.toString result replaced the immutable name reference")
        }
    }
    var chain *j.JavaString = Chained()
    var argument *j.JavaString = Argument()
    if chain != little || argument != big {
        t.Fatal("chained or String-argument invocation changed the name reference")
    }
    erased := Erased()
    if text, ok := erased.(*j.JavaString); !ok || text != big {
        t.Fatal("Object-erased ByteOrder.toString result is not the original Java String")
    }
}
`)
}
