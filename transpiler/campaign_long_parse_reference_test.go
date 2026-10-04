package transpiler

import (
	"os"
	"strings"
	"testing"
)

func longParseActualJava(t *testing.T, path string) string {
	t.Helper()
	source, err := os.ReadFile("testdata/long_parse_actual/src/main/java/" + path)
	if err != nil { t.Fatal(err) }
	return string(source)
}

// Exact independently JDK-compiled source checks the qualified String,int call
// and the statically imported String default overload in one preserved unit.
func TestCampaignJavaLongParseLongActualSourceLowering(t *testing.T) {
	out := renderIntrinsicProgram(t, longParseActualJava(t, "probe/parse/LongFlow.java"))
	if count := strings.Count(out, "stdjava.JavaLongParseLong("); count != 2 { t.Fatalf("canonical helper count=%d, want2:\n%s", count, out) }
	if strings.Contains(out, "stdjava.ParseLong(") { t.Fatalf("native parse helper remains:\n%s", out) }
	if strings.Contains(out, "StringToNative") { t.Fatalf("native string conversion in canonical LongFlow:\n%s", out) }
}

// Exact original Main imports probe.shadow.Long; external owner registration
// must preserve that source parseLong call, even with java.lang.Long present.
func TestCampaignJavaLongParseLongActualSourceShadow(t *testing.T) {
	out := renderIntrinsicProgram(t, longParseActualJava(t, "probe/app/Main.java"))
	if strings.Contains(out, "stdjava.JavaLongParseLong(") || strings.Contains(out, "stdjava.ParseLong(") { t.Fatalf("source Long captured by JDK lowering:\n%s", out) }
}
