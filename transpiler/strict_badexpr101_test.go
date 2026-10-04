package transpiler

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func runStrictBadExpr101(t *testing.T, fixture string, flags ...string) (string, error) {
	t.Helper()
	withCleanDiagnostics(t)
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previousGlobal })
	var output bytes.Buffer
	args := append([]string{"-sync"}, flags...)
	args = append(args, filepath.Join("testdata", "strict_badexpr101", fixture))
	err := run(args, &output)
	return output.String(), err
}

func TestStrictBadExpr101_UnsupportedAssignmentFails(t *testing.T) {
	for _, fixture := range []string{"unsupported", "unsupported_nested"} {
		t.Run(fixture, func(t *testing.T) {
			output, err := runStrictBadExpr101(t, fixture, "-strict")
			if err == nil {
				t.Fatalf("strict accepted an unsupported emitted expression: %s", output)
			}
			if !strings.Contains(err.Error(), "unsupported") {
				t.Fatalf("missing structured unsupported error: %v", err)
			}
			if len(Diagnostics()) != 1 {
				t.Fatalf("diagnostics = %#v, want exactly one", Diagnostics())
			}
			if strings.Contains(output, "BadExpr") {
				t.Fatalf("strict published invalid Go: %s", output)
			}
		})
	}
}

func TestStrictBadExpr101_NonStrictPreservesPartialOutput(t *testing.T) {
	output, err := runStrictBadExpr101(t, "unsupported")
	if err != nil {
		t.Fatalf("permissive mode failed: %v", err)
	}
	if !strings.Contains(output, "BadExpr") || !strings.Contains(output, "return count") {
		t.Fatalf("permissive partial conversion changed: %s", output)
	}
}

func TestStrictBadExpr101_SupportedArithmeticAndComments(t *testing.T) {
	_, err := runStrictBadExpr101(t, "supported", "-strict")
	if err != nil {
		t.Fatalf("supported expression or comment rejected: %v", err)
	}
	if len(Diagnostics()) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", Diagnostics())
	}
}

func TestStrictBadExpr101_ActualBadExprIdentifierAndLiteral(t *testing.T) {
	output, err := runStrictBadExpr101(t, "identifier", "-strict")
	if err != nil {
		t.Fatalf("ordinary BadExpr name/text rejected: %v", err)
	}
	if !strings.Contains(output, "BadExpr") {
		t.Fatalf("control lost its ordinary identifier/text: %s", output)
	}
	if len(Diagnostics()) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", Diagnostics())
	}
}

func TestStrictBadExpr101_UnrelatedWarningDoesNotFail(t *testing.T) {
	_, err := runStrictBadExpr101(t, "warning", "-strict")
	if err != nil {
		t.Fatalf("warning without an emitted bad node rejected: %v", err)
	}
}

func TestStrictBadExpr101_ExcludedMethodDoesNotFail(t *testing.T) {
	output, err := runStrictBadExpr101(t, "ignored", "-strict", "-exclude-annotations", "@Deprecated")
	if err != nil {
		t.Fatalf("deliberately excluded method rejected: %v", err)
	}
	if strings.Contains(output, "Omitted") {
		t.Fatalf("excluded method was emitted: %s", output)
	}
}

func TestStrictBadExpr101_SourceTypeShadowDoesNotFail(t *testing.T) {
	_, err := runStrictBadExpr101(t, "shadow", "-strict")
	if err != nil {
		t.Fatalf("source type shadow rejected: %v", err)
	}
}
