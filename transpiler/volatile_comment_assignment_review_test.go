package transpiler

import (
	"strings"
	"testing"
)

// Frozen source review witness, not executed. The accepted parser comments AST
// proves that comments are named children before the unnamed assignment token.
// A strict generated-program gate must also compare Java/Go output 7:7:1.
func TestVolatileCommentAssignmentOperatorTrivia(t *testing.T) {
	got := renderGoFileFromJava(t, `public class CommentedVolatile {
        volatile int value = 3;
        static int calls;
        int rhs() { calls++; return 4; }
        int update() { return (((value)) /* operator trivia */ += rhs()); }
    }`)
	if !strings.Contains(got, "VolatileLoad") || !strings.Contains(got, "VolatileStore") {
		t.Fatalf("comment trivia lost volatile read/write lowering: %s", got)
	}
}
