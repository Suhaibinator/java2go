package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestNumericStreams_CanonicalStringConversionsBehavior(t *testing.T) {
	const source = `import java.util.ArrayList;
import java.util.List;
import java.util.stream.IntStream;
public class CanonicalStringConversions {
    public static String run() {
        List<String> words = new ArrayList<String>();
        words.add("A😀");
        words.add("\uD800");
        int lengths = words.stream().mapToInt(w -> w.length()).sum();
        long wide = words.stream().mapToLong(w -> w.length()).sum();
        long boxedCount = IntStream.rangeClosed(1, 3).boxed().count();
        long asLong = IntStream.rangeClosed(1, 3).asLongStream().sum();
        double asDouble = IntStream.rangeClosed(1, 3).asDoubleStream().sum();
        return lengths + ":" + wide + ":" + boxedCount + ":" + asLong + ":" + asDouble;
    }
}`
	verifyCanonicalStringStreamOracle(t, "CanonicalStringConversions", source)
}

func TestNumericStreams_CanonicalStringCharsBehavior(t *testing.T) {
	const source = `public class CanonicalStringChars {
    public static String run() {
        String text = "\0A😀\uD800\uDFFFZ";
        long count = text.chars().count();
        int total = text.chars().sum();
        int smallest = text.chars().min().getAsInt();
        int largest = text.chars().max().getAsInt();
        int first = text.chars().findFirst().getAsInt();
        int middle = text.chars().skip(2).limit(3).sum();
        long wide = text.chars().asLongStream().sum();
        return count + ":" + total + ":" + smallest + ":" + largest + ":" + first + ":" + middle + ":" + wide;
    }
}`
	verifyCanonicalStringStreamOracle(t, "CanonicalStringChars", source)
}

func TestNumericStreams_CanonicalStringCharsNullBehavior(t *testing.T) {
	const source = `public class CanonicalStringCharsNull {
    public static String run() {
        String text = null;
        try {
            text.chars();
            return "unexpected success";
        } catch (NullPointerException expected) {
            return "NullPointerException";
        }
    }
}`
	verifyCanonicalStringStreamOracle(t, "CanonicalStringCharsNull", source)
}

func verifyCanonicalStringStreamOracle(t *testing.T, name, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, name, source)
	t.Logf("JDK canonical String stream oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestCanonicalStreamOracle(t *testing.T) {
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) {
        t.Fatalf("JVM %%q != generated Go UTF16 %%#v", %q, got)
    }
}
`, utf16.Encode([]rune(want)), want))
}
