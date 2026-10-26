package transpiler

import (
	"strings"
	"testing"
)

// This gate pins the coordinated String ABI separately from JVM parity. It
// changes no preexisting AST expectation or semantic control.
func TestCampaignBigMathCanonicalStringLowering(t *testing.T) {
	generated := renderGoFileFromJava(t, `import java.math.BigInteger;import java.math.BigDecimal;
public class BigMathCanonicalLowering {static String integer(String text){return new BigInteger(text).toString();}static String decimal(String text){return new BigDecimal(text).toString();}}`)
	for _, required := range []string{"stdjava.NewBigIntegerJavaString(", "stdjava.NewBigDecimalJavaString(", ".StringJava2goExecution(", "*stdjava.JavaString"} {
		if !strings.Contains(generated, required) {
			t.Fatalf("missing canonical BigMath ABI %q in:\n%s", required, generated)
		}
	}
	for _, legacy := range []string{"stdjava.NewBigInteger(", "stdjava.NewBigDecimal(", ".String()"} {
		if strings.Contains(generated, legacy) {
			t.Fatalf("legacy BigMath ABI %q remains in:\n%s", legacy, generated)
		}
	}
}
