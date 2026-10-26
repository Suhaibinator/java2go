package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
	"testing"
)

func namePath168OldLexicalMemberType(name string, ctx Ctx) *symbol.ClassScope {
	parts := strings.Split(name, ".")
	for owner := ctx.currentClass; owner != nil; owner = owner.Enclosing {
		if owner.Class != nil && owner.Class.OriginalName == parts[0] {
			if found := directMemberTypePath(owner, parts[1:], ctx); found != nil {
				return found
			}
		}
		if owner == ctx.memberTypeHeaderOwner && !classHeaderIncludesMemberTypes(owner) {
			continue
		}
		if member := memberTypeInHierarchy(owner, parts[0], ctx); member != nil {
			return directMemberTypePath(member, parts[1:], ctx)
		}
	}
	return nil
}

func TestNamePath168LexicalPartsAllSpellings(t *testing.T) {
	helper := setupParseHelper(t, "class Alpha { class Beta { class Gamma {} } } class Child extends Alpha {}")
	alpha := helper.File.Symbols.FindClassScope("Alpha")
	beta := helper.File.Symbols.FindClassScope("Beta")
	child := helper.File.Symbols.FindClassScope("Child")
	if alpha == nil || beta == nil || child == nil {
		t.Fatal("missing source scopes")
	}
	for _, name := range []string{"β", "$X", "type", "\xffinvalid"} {
		alpha.Subclasses = append(alpha.Subclasses, &symbol.ClassScope{Class: &symbol.Definition{Name: name, OriginalName: name}, Enclosing: alpha})
	}
	names := []string{"", "plain", "Alpha", "Child", "Beta", "Beta.Gamma", "Alpha.Beta", "Alpha.Beta.Gamma", "β", "$X", "type", "\xffinvalid", ".", "..", ".Alpha", "Alpha.", "Alpha..Beta", "Alpha.Beta.", "Alpha...Beta", ".Alpha.Beta"}
	for _, owner := range []*symbol.ClassScope{alpha, beta, child} {
		for _, header := range []bool{false, true} {
			ctx := helper.Ctx.Clone()
			ctx.currentClass = owner
			if header {
				ctx = classHeaderTypeCtx(owner, ctx)
			}
			for _, name := range names {
				got := lexicalMemberType(name, ctx)
				want := namePath168OldLexicalMemberType(name, ctx)
				if got != want {
					t.Fatalf("owner %s header %v name %q got %p old %p", owner.Class.OriginalName, header, name, got, want)
				}
			}
		}
	}
	ctx := helper.Ctx.Clone()
	ctx.currentClass = child
	if lexicalMemberType("Beta", ctx) != beta {
		t.Fatal("inherited simple positive identity")
	}
	ctx.currentClass = beta
	if lexicalMemberType("Alpha.Beta.Gamma", ctx) != beta.Subclasses[0] {
		t.Fatal("qualified positive identity")
	}
	if lexicalMemberType("Alpha..Beta", ctx) != nil {
		t.Fatal("consecutive dot must remain miss")
	}
}
