package transpiler

import "testing"

func TestStringContainsUsesCallerExecution(t *testing.T) {
	out := renderGoFileFromJava(t, `public class ContainsProgram { static boolean run(String text, CharSequence target) { return text.contains(target); } }`)
	assertContains(t, out, "stdjava.JavaStringContainsExecution(__java2goExecution,")
}
