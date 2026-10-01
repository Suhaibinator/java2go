package transpiler

import (
	"strings"
	"testing"
)

func TestSourceCovariantInterfacePublicAndExecutionDescriptors(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class CovariantInterfaceProbe {
 interface Broad { Object flag(); int other(); }
 interface Narrow extends Broad { String flag(); }
 interface Further extends Narrow {}
 static class Item implements Further {
  public String flag(){return "value";}
  public int other(){return 7;}
 }
 static String call(Narrow value){return value.flag();}
}
`)
	for _, want := range []string{
		"type CovariantInterfaceProbenarrow interface", "Broad", "Flag() any",
		"FlagJava2goExactExecution(__java2goExecution *stdjava.Execution) *stdjava.JavaString",
		"ObjectView[*stdjava.JavaString]",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in covariant interface:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Flag() *stdjava.JavaString") {
		t.Fatalf("incompatible narrow public selector:\n%s", out)
	}
}
