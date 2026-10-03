package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestFreshMatcherOwnershipFactsLifetime(t *testing.T) {
	t.Run("carried-holder-resets-on-graph-replacement", func(t *testing.T) {
		ctx, target, _ := ownershipMatcherCompleteGraph(t, 8)
		oldFile := findFileScopeForClassScope(target, ctx)
		carried := Ctx{currentFile: oldFile, currentClass: target, sourceOwnership: sourceOwnershipIndex(ctx)}.Clone()
		fresh := sourceGenericViewDemandTestContext(t, map[string]string{"p/Base.java": `package p;public class Base<T>{}`, "q/Use.java": `package q;public class Use{}`})
		now := findQualifiedSourceClass("p.Base")
		file := findFileScopeForClassScope(now, fresh)
		if now == target || file == oldFile || classScopeCtx(now, carried).currentFile != file || findFileScopeForClassScope(target, carried) != nil {
			t.Fatal("carried ownership crossed source graph/declaration identity")
		}
	})
	t.Run("carried-holder-keeps-late-registration-and-unindexed-fallback", func(t *testing.T) {
		ctx, _, _ := ownershipMatcherCompleteGraph(t, 8)
		owner := findQualifiedSourceClass("q.Use")
		file := findFileScopeForClassScope(owner, ctx)
		carried := Ctx{currentFile: ctx.currentFile, sourceOwnership: sourceOwnershipIndex(ctx)}.Clone()
		local := &symbol.ClassScope{Class: &symbol.Definition{Name: "Late"}, Enclosing: owner}
		if classScopeCtx(local, carried).currentFile != file {
			t.Fatal("unindexed enclosing source owner fallback changed")
		}
		local.Enclosing = nil
		if findFileScopeForClassScope(local, carried) != nil {
			t.Fatal("unregistered source owner was invented")
		}
		owner.Subclasses = append(owner.Subclasses, local)
		if classScopeCtx(local, carried).currentFile != file {
			t.Fatal("late registered owner was negatively cached")
		}
	})
}
