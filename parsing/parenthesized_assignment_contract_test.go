package parsing

import (
	"bytes"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// Frozen after actual JDK acceptance of parenthesized assignments and before
// parser repair. Syntax acceptance must retain input bytes and assignment nodes;
// a recovery binary expression must not replace a value-producing assignment.
func TestParenthesizedAssignmentGrammarContract(t *testing.T) {
	cases := []struct {
		name, source string
		wantError    bool
		assignments  int
	}{
		{"simple_statement", `class Probe { int value; void f(){ ((value)) = 7; } }`, false, 1},
		{"simple_result", `class Probe { int value; int f(){ return (((value)) = 7); } }`, false, 1},
		{"compound_statement", `class Probe { int value; void f(){ ((value)) += 4; } }`, false, 1},
		{"compound_result", `class Probe { int value; int f(){ return (((value)) += 4); } }`, false, 1},
		{"array_simple", `class Probe { int[] values; int f(){ return (((values[0])) = 7); } }`, false, 1},
		{"qualified_compound", `class Probe { int value; Probe other; int f(){ return (((other.value)) += 1); } }`, false, 1},
		{"comments", `class Probe { int value; int f(){ return ((/*a*/value/*b*/)) /*c*/ += 1; } }`, false, 1},
		{"literal_and_comment_control", `class Probe { String text="((value)) += 4"; int value; int f(){ /*((value))=7;*/ return value; } }`, false, 0},
		{"binary_value_control", `class Probe { int value; int f(){ return ((value)) + 4; } }`, false, 0},
		{"malformed_group_control", `class Probe { int value; int f(){ return ((value); } }`, true, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := []byte(tc.source)
			file := SourceFile{Name: "Probe.java", Source: append([]byte(nil), original...)}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			t.Logf("original source: %s\nAST: %s", tc.source, file.Ast.String())
			if !bytes.Equal(file.Source, original) {
				t.Error("parser repair changed source bytes or offsets")
			}
			if file.Ast.HasError() != tc.wantError {
				t.Errorf("syntax error=%v, want %v", file.Ast.HasError(), tc.wantError)
			}
			if tc.assignments < 0 {
				return
			}
			count := 0
			var visit func(*sitter.Node)
			visit = func(node *sitter.Node) {
				if node.Type() == "assignment_expression" {
					count++
				}
				for i := 0; i < int(node.NamedChildCount()); i++ {
					visit(node.NamedChild(i))
				}
			}
			visit(file.Ast)
			if count != tc.assignments {
				t.Errorf("assignment nodes=%d, want %d", count, tc.assignments)
			}
		})
	}
}
