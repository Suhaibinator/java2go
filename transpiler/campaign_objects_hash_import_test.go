package transpiler

import (
	"fmt"
	"os"
	"testing"
	"unicode/utf16"
)

func TestCampaignObjectsHashStaticImportJDK21(t *testing.T) {
	raw, err := os.ReadFile("testdata/campaign_objects_hash/ImportedObjectsHashProbe.java")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	want := campaignRuntimeJavaOracle(t, "ImportedObjectsHashProbe", source)
	t.Logf("JDK imported Objects hash observation: %q", want)
	out := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, out, fmt.Sprintf(`package main
 import("testing";"slices")
 func TestImportedObjectsHashRuntime(t *testing.T){got:=Run();want:=%#v;if got==nil||!slices.Equal(got.UTF16Copy(),want){t.Fatalf("imported Objects hash parity: %%v",got)}}
 `, utf16.Encode([]rune(want))))
}
