package transpiler

import (
	"fmt"
	"os"
	"testing"
	"unicode/utf16"
)

func TestCampaignObjectsHashJDK21(t *testing.T) {
	raw, err := os.ReadFile("testdata/campaign_objects_hash/ObjectsHashProbe.java")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	want := campaignRuntimeJavaOracle(t, "ObjectsHashProbe", source)
	t.Logf("JDK Objects hash observation: %q", want)
	out := renderGoFileFromJava(t, source)
	units := utf16.Encode([]rune(want))
	runGoTestInTempModule(t, out, fmt.Sprintf(`package main
import("testing";"slices")
func TestObjectsHashRuntime(t *testing.T){got:=Run();want:=%#v;if got==nil{t.Fatal("null Objects hash transcript")};if !slices.Equal(got.UTF16Copy(),want){t.Fatalf("JVM %%v != Go %%v",want,got.UTF16Copy())}}
`, units))
}
