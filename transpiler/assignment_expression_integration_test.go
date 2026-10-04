package transpiler

import (
	"strings"
	"testing"
)

func TestAssignmentExpressionYieldsAssignedValueAndEvaluatesTargetOnce(t *testing.T) {
	src := `
public class AssignmentValueProgram {
    public static String run() {
        String line;
        String seen = (line = "alpha");

        int total = 3;
        int updated = (total += 4);

        int[] values = new int[] { 10, 20 };
        int index = 0;
        int element = (values[index++] += 5);

        int staged = 1;
        int stagedResult = (staged += (staged = 5));

		byte narrow = 120;
		byte narrowResult = (narrow += 10);
		boolean flag = true;
		boolean boolResult = (flag &= false);
		String text = "value=";
		String textResult = (text += 2.0);
		int unsigned = -8;
		unsigned >>>= 1;
		long wideUnsigned = -1L;
		wideUnsigned >>>= 1;
		char wrappedChar = (char) 65535;
		char wrappedCharResult = (wrappedChar += 1);

        return seen + ":" + line + ":" + total + ":" + updated
            + ":" + element + ":" + index + ":" + values[0]
			+ ":" + staged + ":" + stagedResult
			+ ":" + narrow + ":" + narrowResult
			+ ":" + flag + ":" + boolResult
			+ ":" + text + ":" + textResult + ":" + unsigned + ":" + wideUnsigned
			+ ":" + (int) wrappedChar + ":" + (int) wrappedCharResult;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "AssignmentExpression") {
		t.Fatalf("assignment expressions must be lowered to executable Go, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "slices"
    "testing"
    "unicode/utf16"
)

func TestAssignmentValueRuntime(t *testing.T) {
	got := Run()
	if got == nil {
	    t.Fatal("Run() returned null")
	}
	const want = "alpha:alpha:7:7:15:1:15:6:6:-126:-126:false:false:value=2.0:value=2.0:2147483644:9223372036854775807:0:0"
	wantUnits := utf16.Encode([]rune(want))
	gotUnits := got.UTF16Copy()
	if !slices.Equal(gotUnits, wantUnits) {
        t.Fatalf("Run() UTF16 = %v, want %v", gotUnits, wantUnits)
    }
}
`)
}
