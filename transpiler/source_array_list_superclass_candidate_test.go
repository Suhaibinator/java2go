package transpiler

import "testing"

func TestSourceArrayListSuperclassCandidateRefusalOnly(t *testing.T) {
	for _, text := range []string{"", "Parent", "p.Parent", "Parent<Token>", "p.Parent<q.Token>", "Outer<T>.Other<U>"} {
		if sourceArrayListSuperclassCandidate(text) {
			t.Errorf("unrelated plain superclass remains candidate: %q", text)
		}
	}
	for _, text := range []string{"ArrayList", "java.util.ArrayList", "ArrayList<Object>", "java.util.ArrayList<java.lang.Object>", "Outer<T>.ArrayList<U>", "java.util./*comment*/ArrayList<Object>", "\\u0041rrayList<Object>", "ArrayList<É>", "java . util . ArrayList<Object>", "Broken<", "Parent>", "Parent<T><U>"} {
		if !sourceArrayListSuperclassCandidate(text) {
			t.Errorf("possible or unknown superclass lost full proof: %q", text)
		}
	}
}

func TestSourceArrayListDirectRootKeepsFullProof(t *testing.T) {
	for _, row := range []struct {
		name, source string
		want         bool
	}{
		{"qualified", `class Values extends java.util.ArrayList<Object>{}`, true},
		{"simple", `import java.util.ArrayList; class Values extends ArrayList<Object>{}`, true},
		{"commented-import", `import /*header*/ java.util./*part*/ArrayList; class Values extends ArrayList<Object>{}`, true},
		{"source-shadow", `class ArrayList<T>{} class Values extends ArrayList<Object>{}`, false},
		{"qualified-source-shadow", `package own; class ArrayList<T>{} class Values extends own.ArrayList<Object>{}`, false},
		{"unrelated", `class Parent<T>{} class Values extends Parent<Object>{}`, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			owner := h.File.Symbols.FindClassScope("Values")
			if got := sourceArrayListDirectRoot(owner, h.Ctx); got != row.want {
				t.Fatalf("direct root=%t want%t", got, row.want)
			}
			if row.want {
				if !sourceArrayListSuperclassCandidate(owner.Superclass) {
					t.Fatal("canonical candidate bypassed original proof")
				}
				detached := *owner
				if sourceArrayListDirectRoot(&detached, h.Ctx) {
					t.Fatal("candidate spelling donated ownership to detached scope")
				}
			}
		})
	}
	t.Run("unicode-as-parsed", func(t *testing.T) {
		h := setupParseHelper(t, `class Values extends java.util.\u0041rrayList<Object>{}`)
		owner := h.File.Symbols.FindClassScope("Values")
		if !sourceArrayListSuperclassCandidate(owner.Superclass) {
			t.Fatal("Unicode parser spelling bypassed the unchanged full proof")
		}
		// The parser retains escape spellings; this performance change must
		// preserve the original predicate rather than add Unicode capability.
		header, owned := sourceArrayListDeclaration(owner, h.Ctx)
		original := owned && sourceArrayListCanonicalEdge(owner.Superclass, header) && sourceArrayListNativeConstructors(owner, header) && !sourceArrayListUncoveredOverride(owner, header)
		if got := sourceArrayListDirectRoot(owner, h.Ctx); got != original {
			t.Fatal("Unicode spelling changed the existing full-proof result")
		}
	})
}

func TestSourceArrayListSuperclassKeepsInheritedWalk(t *testing.T) {
	h := setupParseHelper(t, `class Parent extends java.util.ArrayList<Object>{} class Values extends Parent{}`)
	child, parent := h.File.Symbols.FindClassScope("Values"), h.File.Symbols.FindClassScope("Parent")
	if sourceArrayListDirectRoot(child, h.Ctx) {
		t.Fatal("source inherited edge became direct native edge")
	}
	if sourceArrayListRoot(child, h.Ctx) != parent {
		t.Fatal("inherited source-root walk changed")
	}
}
