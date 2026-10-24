package transpiler

import (
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestNamePath168ImmutableFrameCloneAndSiblings(t *testing.T) {
	a, b, c := &symbol.ClassScope{}, &symbol.ClassScope{}, &symbol.ClassScope{}
	parent := &memberTypeLookupFrame{owner: a}
	ctx := Ctx{memberTypeLookupFrame: parent}
	retained := ctx.Clone()
	left := ctx.Clone()
	left.memberTypeLookupFrame = &memberTypeLookupFrame{owner: b, parent: parent}
	right := ctx.Clone()
	right.memberTypeLookupFrame = &memberTypeLookupFrame{owner: c, parent: parent}
	for _, scope := range []*symbol.ClassScope{a, b, c} {
		if memberTypeLookupFrameActive(parent, scope) != (scope == a) {
			t.Fatal("parent history mutated")
		}
		if memberTypeLookupFrameActive(left.memberTypeLookupFrame, scope) != (scope == a || scope == b) {
			t.Fatal("left path identity")
		}
		if memberTypeLookupFrameActive(right.memberTypeLookupFrame, scope) != (scope == a || scope == c) {
			t.Fatal("right path identity")
		}
	}
	if retained.memberTypeLookupFrame != parent || retained.memberTypeLookupFrame.parent != nil {
		t.Fatal("Clone did not preserve immutable history")
	}
	cgo := runtime.NumCgoCall()
	for i := 0; i < 100; i++ {
		if !memberTypeLookupFrameActive(left.memberTypeLookupFrame, a) {
			t.Fatal("lost ancestor")
		}
	}
	if runtime.NumCgoCall() != cgo {
		t.Fatal("frame predicate must be pure")
	}
}

func TestNamePath168FrameCycleDirectMemberAndSeededMap(t *testing.T) {
	helper := setupParseHelper(t, "class Root { class Mark {} } class Child extends Root {}")
	root := helper.File.Symbols.FindClassScope("Root")
	child := helper.File.Symbols.FindClassScope("Child")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = child
	ctx.memberTypeLookupFrame = &memberTypeLookupFrame{owner: root}
	retained := ctx.Clone()
	if memberTypeInHierarchy(root, "Mark", ctx) != root.Subclasses[0] {
		t.Fatal("direct declaration must precede frame guard")
	}
	if memberTypeInHierarchy(root, "Absent", ctx) != nil {
		t.Fatal("active cycle must stop")
	}
	if memberTypeInHierarchy(child, "Mark", ctx) != root.Subclasses[0] {
		t.Fatal("inherited direct declaration identity")
	}
	if ctx.memberTypeLookupFrame != retained.memberTypeLookupFrame || ctx.memberTypeLookupFrame.parent != nil {
		t.Fatal("retained frame history changed")
	}
	ctx.memberTypeLookupPath = map[*symbol.ClassScope]bool{root: false}
	if memberTypeInHierarchy(child, "Mark", ctx) != root.Subclasses[0] {
		t.Fatal("seeded false-key direct declaration identity")
	}
	if len(ctx.memberTypeLookupPath) != 1 || ctx.memberTypeLookupPath[root] {
		t.Fatal("legacy seed map mutated")
	}
	if ctx.memberTypeLookupFrame != retained.memberTypeLookupFrame {
		t.Fatal("seed branch replaced caller frame")
	}
}
