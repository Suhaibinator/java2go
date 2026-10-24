package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Independent test roots let a bounded supervisor retain each workflow's
// original failure even when malformed frontend roles cause a compiler panic.
func TestCampaignStatementCommentsUpdateAssignment(t *testing.T) {
	runStatementCommentContractOracle(t, "StatementUpdateAssignmentFlow")
}

func TestCampaignStatementCommentsDeclarationFor(t *testing.T) {
	runStatementCommentContractOracle(t, "StatementDeclarationForFlow")
}

func TestCampaignStatementCommentsFormalControlLoops(t *testing.T) {
	runStatementCommentContractOracle(t, "StatementFormalControlFlow")
}

func runStatementCommentContractOracle(t *testing.T, name string) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", "statement_comment_contracts", name+".java"))
	if err != nil {
		t.Fatal(err)
	}
	want := campaignRuntimeJavaOracle(t, name, string(source))
	t.Logf("JDK statement comment observation: %q", want)
	generated := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestStatementCommentOracle(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}
`, want))
}
