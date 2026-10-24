package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Frozen original JDK controls before repairing the canonical Boolean ingress.
func booleanIngressJDKOracle(t *testing.T, name string) {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("testdata", "boolean_ingress89", name+".java"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(bytes)
	want := campaignRuntimeJavaOracle(t, name, source)
	oracle, err := strconv.ParseInt(want, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Boolean ingress original JDK oracle: %s:%s", name, want)
	generated := renderGoFileFromJava(t, source)
	if name == "BooleanStaticImportProbe" {
		t.Logf("frozen static import generated source:\n%s", generated)
	}
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestBooleanIngressParity(t *testing.T) { if got := Run(); got != int32(%d) { t.Fatalf("Go %%d != JVM %%d", got, int32(%d)) } }
`, oracle, oracle))
}
func TestBooleanIngressBooleanIngressProbeJDK21(t *testing.T) {
	booleanIngressJDKOracle(t, "BooleanIngressProbe")
}
func TestBooleanIngressBooleanStaticImportProbeJDK21(t *testing.T) {
	booleanIngressJDKOracle(t, "BooleanStaticImportProbe")
}
func TestBooleanIngressBooleanSourceShadowProbeJDK21(t *testing.T) {
	booleanIngressJDKOracle(t, "BooleanSourceShadowProbe")
}
