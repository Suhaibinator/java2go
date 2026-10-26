package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
)

// The actual pinned Java grammar names super but leaves extends unnamed. A
// source role is counted independently from the number of named AST children.
func TestDeclaredMethodWildcardBoundSyntaxRoles(t *testing.T) {
	for _, row := range []struct {
		name, formal, direction, bound string
		named, directions              int
		keywordNamed                   bool
	}{
		{"upper_extends_unnamed", "Collection<? extends E> values", "extends", "E", 1, 1, false},
		{"lower_super_named", "Collection<? super E> values", "super", "E", 2, 1, true},
		{"lower_wrong_direction_refused", "Collection<? super E> values", "extends", "", 2, 1, true},
		{"upper_wrong_direction_refused", "Collection<? extends E> values", "super", "", 1, 1, false},
		{"unbounded_refused", "Collection<?> values", "super", "", 0, 0, false},
		{"annotated_wildcard_refused", "Collection<@Deprecated ? super E> values", "super", "", 3, 1, true},
		{"nested_bound_refused", "Collection<? super java.util.List<E>> values", "super", "", 2, 1, true},
		{"scoped_bound_syntax", "Collection<? super java.lang.Object> values", "super", "java.lang.Object", 2, 1, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			f := wildcardStrictScoreSetup(t, "<E>", row.formal, "List<String> values=null;", "values", nil)
			wildcardStrictScoreFreshInputs(t, f)
			declaring, _ := genericMethodOwnerContext(f.method)
			parameters := nodeutil.SemanticNamedChildrenOf(f.method.DeclarationNode.ChildByFieldName("parameters"))
			if len(parameters) != 1 {
				t.Fatal("original formal parameter absent")
			}
			typ, _ := nodeutil.JavaParameterNodes(parameters[0])
			wildcard := declaredWildcardSingleTypeArgument(typ)
			if wildcard == nil || wildcard.Type() != "wildcard" {
				t.Fatal("original formal wildcard occurrence absent")
			}
			named := len(nodeutil.SemanticNamedChildrenOf(wildcard))
			directions := 0
			keywordNamed := false
			for index := 0; index < int(wildcard.ChildCount()); index++ {
				child := wildcard.Child(index)
				if child.Type() == "extends" || child.Type() == "super" {
					directions++
					keywordNamed = child.IsNamed()
				}
			}
			if named != row.named || directions != row.directions || keywordNamed != row.keywordNamed {
				t.Fatalf("original pinned grammar premise changed: named=%d directions=%d keywordNamed=%v", named, directions, keywordNamed)
			}
			bound := declaredWildcardBoundNode(wildcard, row.direction)
			actual := ""
			if bound != nil {
				actual = bound.Content(declaring.currentFile.Source)
			}
			t.Logf("WILDCARD_BOUND_SYNTAX case=%s namedChildren=%d directions=%d keywordNamed=%v bound=%q", row.name, named, directions, keywordNamed, actual)
			if actual != row.bound {
				t.Fatalf("original wildcard role mismatch: bound=%q want=%q", actual, row.bound)
			}
		})
	}
}
