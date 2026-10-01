package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// These source fixtures are frozen before the private repair proposal. Each
// result comes from the actual JDK 21 program, including every observable byte.
func TestCampaignNIOHeapContracts(t *testing.T) {
	for _, name := range []string{"HeapWindowFlow", "HeapAliasFlow", "HeapFailureFlow"} {
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata", "nio_heap_contracts", name+".java"))
			if err != nil {
				t.Fatal(err)
			}
			want := campaignRuntimeJavaOracle(t, name, string(source))
			generated := renderGoFileFromJava(t, string(source))
			runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestNIOHeapOracle(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}
`, want))
		})
	}
}
