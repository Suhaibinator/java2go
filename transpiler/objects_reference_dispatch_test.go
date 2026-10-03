package transpiler

import (
	"strings"
	"testing"
)

func TestJavaObjectsReferenceCanonicalDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
import java.util.Objects;
import java.util.function.Supplier;
public class CanonicalObjectsDispatch {
    static String one(String value) { return Objects.requireNonNull(value); }
    static String message(String value, String detail) { return Objects.requireNonNull(value, detail); }
    static String supplier(String value, Supplier<String> detail) { return Objects.requireNonNull(value, detail); }
    static <T> T generic(T value) { return Objects.<T>requireNonNull(value); }
    static <S extends String> String bound(Object value, S detail) { Objects.requireNonNull(value, detail); return detail; }
    static String nullMessage(String value) { return Objects.requireNonNull(value, (String)null); }
    static String nullSupplier(String value) { return Objects.requireNonNull(value, (Supplier<String>)null); }
}`)
	for _, call := range []string{
		"ObjectsRequireNonNullReference[*stdjava.JavaString]",
		"ObjectsRequireNonNullMessageReference[*stdjava.JavaString]",
		"ObjectsRequireNonNullSupplierReference[*stdjava.JavaString]",
		"ObjectsRequireNonNullReference[T]",
		"ObjectsRequireNonNullMessageReference[any]",
	} {
		if !strings.Contains(out, call) {
			t.Errorf("missing canonical Objects dispatch %s:\n%s", call, out)
		}
	}
	for _, legacy := range []string{"ObjectsRequireNonNull[", "ObjectsRequireNonNullMessage[", "ObjectsRequireNonNullSupplier["} {
		if strings.Contains(out, legacy) {
			t.Errorf("Java intrinsic selected native Go text helper %s:\n%s", legacy, out)
		}
	}
}

func TestJavaObjectsReferenceSourceShadowDispatch(t *testing.T) {
	out := renderGoFileFromJava(t, `
class Objects {
    static String requireNonNull(String value, String message) { return message; }
}
public class ObjectsShadowDispatch {
    static String source(String value, String message) { return Objects.requireNonNull(value, message); }
    static String platform(String value, String message) { return java.util.Objects.requireNonNull(value, message); }
}`)
	if count := strings.Count(out, "ObjectsRequireNonNullMessageReference["); count != 1 {
		t.Fatalf("source Objects shadow was hijacked or qualified platform dispatch was lost: canonical calls=%d\n%s", count, out)
	}
	if !strings.Contains(out, "ObjectsRequireNonNull") {
		t.Fatal("source requireNonNull method disappeared")
	}
}
