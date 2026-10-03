package transpiler

import (
	"strings"
	"testing"
)

func TestAssertionStatementCanonicalPrimitiveDetailLowering(t *testing.T) {
	out := renderGoFileFromJava(t, `class AssertionPrimitiveDetail {
 static void run(boolean condition,char character,byte small,short medium,int integer,long wide,float single,double decimal,Object detail){
  assert condition:character;
  assert condition:small;
  assert condition:medium;
  assert condition:integer;
  assert condition:wide;
  assert condition:single;
  assert condition:decimal;
  assert condition:detail;
  assert condition;
 }
}`)
	for _, required := range []string{
		"stdjava.NewAssertionFailure(__java2goExecution, stdjava.JavaStringValueOfChar(character))",
		"stdjava.NewAssertionFailure(__java2goExecution, int32(small))",
		"stdjava.NewAssertionFailure(__java2goExecution, int32(medium))",
		"stdjava.NewAssertionFailure(__java2goExecution, integer)",
		"stdjava.NewAssertionFailure(__java2goExecution, detail)",
	} {
		if !strings.Contains(out, required) {
			t.Fatalf("missing canonical detail declaration boundary %s:\n%s", required, out)
		}
	}
	if strings.Contains(out, "string(character)") || strings.Count(out, "stdjava.NewAssertionFailure(") != 9 || strings.Count(out, "stdjava.JavaAssertionsEnabled()") != 9 {
		t.Fatalf("assertion detail changed native char/guard/evaluation structure:\n%s", out)
	}
	for _, source := range []string{
		`class Character{}class Probe{static void run(Character value){assert false:value;}}`,
		`class Probe<Character>{void run(Character value){assert false:value;}}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "JavaStringValueOfChar(") {
			t.Fatalf("reference detail borrowed primitive char by spelling:\n%s", out)
		}
	}
}
