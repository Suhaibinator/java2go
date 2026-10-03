package transpiler

import (
	"strings"
	"testing"
)

func TestStrictBadExpr101_FinalASTRejectsUnreportedFields(t *testing.T) {
	for _, fixture := range []string{"unsupported_static", "unsupported_volatile"} {
		t.Run(fixture, func(t *testing.T) {
			output, err := runStrictBadExpr101(t, fixture, "-strict")
			if err == nil {
				t.Fatalf("strict accepted an unreported emitted bad node: %s", output)
			}
			diagnostics := Diagnostics()
			if len(diagnostics) != 1 || diagnostics[0].Kind != "expression" || diagnostics[0].NodeType != "ast.BadExpr" {
				t.Fatalf("missing generated AST diagnostic: %#v, error %v", diagnostics, err)
			}
			if strings.Contains(output, "BadExpr") {
				t.Fatalf("strict published invalid Go: %s", output)
			}
		})
	}
}

func TestStrictBadExpr101_UnreportedFieldsStayPermissive(t *testing.T) {
	for _, fixture := range []string{"unsupported_static", "unsupported_volatile"} {
		t.Run(fixture, func(t *testing.T) {
			output, err := runStrictBadExpr101(t, fixture)
			if err != nil || !strings.Contains(output, "BadExpr") {
				t.Fatalf("permissive fallback changed: error %v, output %s", err, output)
			}
			if len(Diagnostics()) != 0 {
				t.Fatalf("strict-only validation changed permissive diagnostics: %#v", Diagnostics())
			}
		})
	}
}
