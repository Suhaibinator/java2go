package transpiler

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func ownershipMatcherCompleteGraph(t *testing.T, padding int) (Ctx, *symbol.ClassScope, *symbol.GlobalSymbols) {
	t.Helper()
	sources := map[string]string{
		"p/Base.java":   `package p;public class Base<T>{}`,
		"p/Parent.java": `package p;public class Parent{}`,
		"q/Use.java":    `package q;import p.Base;import p.Parent;public class Use extends Parent{Object make(){return new Base<String>(){};}}`,
	}
	for index := 0; index < padding; index++ {
		name := fmt.Sprintf("Noise%04d", index)
		sources["q/"+name+".java"] = "package q;public class " + name + "{}"
	}
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	files := make([]parsing.SourceFile, 0, len(names))
	for _, name := range names {
		file := parsing.SourceFile{Name: name, Source: []byte(sources[name])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	// Resolve the complete registered graph through the production batch entrypoint.
	ResolveFiles(files)
	ctx := Ctx{currentFile: files[0].Symbols, currentClass: files[0].Symbols.BaseClass}
	target := findQualifiedSourceClass("p.Base")
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	if target == nil || target.TypeParameters[0].Declaration == nil || len(allSourceClassScopes()) != padding+3 {
		t.Fatal("complete unchanged source graph or declaration identity was lost")
	}
	return ctx, target, symbol.GlobalScope
}

func TestFreshSuperclassMatcherOwnership_Semantics(t *testing.T) {
	t.Run("real-hierarchy-and-fresh-declaring-file", func(t *testing.T) {
		ctx, base, _ := ownershipMatcherCompleteGraph(t, 8)
		parent, use := findQualifiedSourceClass("p.Parent"), findQualifiedSourceClass("q.Use")
		binder := base.TypeParameters[0].Declaration
		for attempt := 0; attempt < 3; attempt++ {
			if !classHasSyntheticSubclass(base, ctx) || !classHasKnownSubclass(parent, ctx) || classHasSyntheticSubclass(use, ctx) || base.TypeParameters[0].Declaration != binder {
				t.Fatal("named hierarchy, fresh superclass resolution or binder identity changed")
			}
			ctx = ctx.Clone()
			ctx.genericFamilies = nil
		}
	})
	t.Run("caller-local-type-and-binder-never-hijack-matcher", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Base.java":     `package p;public class Base<T>{}`,
			"app/Caller.java": `package app;public class Caller<Base>{void active(){class Base{}}}`,
			"q/Use.java":      `package q;import p.Base;public class Use{Object make(){return new Base<String>(){};}}`,
		})
		target, caller := findQualifiedSourceClass("p.Base"), findQualifiedSourceClass("app.Caller")
		ctx.currentClass = caller
		ctx.currentFile = findFileScopeForClassScope(caller)
		for _, method := range caller.Methods {
			if method.OriginalName == "active" {
				ctx.localScope = method
			}
		}
		local := &symbol.ClassScope{Class: &symbol.Definition{Name: "HoistedBase", OriginalName: "Base"}, Enclosing: caller}
		ctx.localClasses = map[string]*localClassInfo{"Base": {scope: local}}
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		if !classHasSyntheticSubclass(target, ctx) || classHasSyntheticSubclass(local, ctx) || ctx.currentClass.TypeParameters[0].Declaration == target.TypeParameters[0].Declaration {
			t.Fatal("fresh matcher inherited caller local names or a class-owned binder")
		}
	})
	t.Run("source-shadow-and-fresh-import-resolution", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Base.java":  `package p;public class Base{}`,
			"q/Base.java":  `package q;public class Base{}`,
			"app/Use.java": `package app;import p.Base;public class Use{Object make(){return new Base(){};}}`,
		})
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		p, q, use := findQualifiedSourceClass("p.Base"), findQualifiedSourceClass("q.Base"), findQualifiedSourceClass("app.Use")
		if !classHasSyntheticSubclass(p, ctx) || classHasSyntheticSubclass(q, ctx) {
			t.Fatal("declaring import source owner changed")
		}
		findFileScopeForClassScope(use).Imports["Base"] = "q"
		if classHasSyntheticSubclass(p, ctx) || !classHasSyntheticSubclass(q, ctx) {
			t.Fatal("carried facts cached a resolved superclass")
		}
		shadow := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Base.java":  `package p;public class Base{}`,
			"app/Use.java": `package app;import p.Base;public class Use{static class Base{}Object make(){return new Base(){};}}`,
		})
		shadow.callableSubclasses = &callableSubclassSourceInventory{}
		imported, owner := findQualifiedSourceClass("p.Base"), findQualifiedSourceClass("app.Use")
		nested := owner.Subclasses[0]
		if classHasSyntheticSubclass(imported, shadow) || !classHasSyntheticSubclass(nested, shadow) {
			t.Fatal("source member shadow lost declaration identity")
		}
	})
	t.Run("graph-reset-and-later-synthetic-owner", func(t *testing.T) {
		ctx, target, _ := ownershipMatcherCompleteGraph(t, 8)
		if !classHasSyntheticSubclass(target, ctx) {
			t.Fatal("first graph lost anonymous edge")
		}
		oldFile := classScopeCtx(target, ctx).currentFile
		fresh := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Base.java": `package p;public class Base<T>{}`,
			"q/Use.java":  `package q;public class Use{}`,
		})
		fresh.callableSubclasses = ctx.callableSubclasses
		now := findQualifiedSourceClass("p.Base")
		if now == target || classHasSyntheticSubclass(now, fresh) || classHasSyntheticSubclass(target, fresh) || classScopeCtx(now, fresh).currentFile == oldFile {
			t.Fatal("reused graph context retained another declaration or source owner")
		}
		owner := findQualifiedSourceClass("q.Use")
		file := findFileScopeForClassScope(owner)
		_ = classScopeCtx(owner, fresh)
		local := &symbol.ClassScope{Class: &symbol.Definition{Name: "LaterSynthetic"}, Enclosing: owner}
		if classScopeCtx(local, fresh).currentFile != file {
			t.Fatal("unindexed synthetic enclosing owner fallback changed")
		}
		local.Enclosing = nil
		owner.Subclasses = append(owner.Subclasses, local)
		if classScopeCtx(local, fresh).currentFile != file {
			t.Fatal("later synthetic registration was negatively cached")
		}
	})
}

// A cached source-event frontier must retain the graph's declaration ownership
// facts when it creates a fresh matching context. Both observations ask the
// same real hierarchy query after the same syntax warmup; all padding remains
// registered and only unrelated source declarations differ.
func TestFreshSuperclassMatcherOwnership_WarmCompleteGraphCost(t *testing.T) {
	const queries = 256
	measure := func(ctx Ctx, target *symbol.ClassScope, graph *symbol.GlobalSymbols) time.Duration {
		symbol.GlobalScope = graph
		if !classHasSyntheticSubclass(target, ctx) {
			t.Fatal("warmup lost actual anonymous superclass")
		}
		binder := target.TypeParameters[0].Declaration
		started := time.Now()
		for query := 0; query < queries; query++ {
			if !classHasSyntheticSubclass(target, ctx) {
				t.Fatal("measured query lost actual anonymous superclass")
			}
		}
		elapsed := time.Since(started)
		if target.TypeParameters[0].Declaration != binder {
			t.Fatal("query changed target declaration identity")
		}
		return elapsed
	}
	exceeded := 0
	for pair := 0; pair < 3; pair++ {
		sparse, st, sg := ownershipMatcherCompleteGraph(t, 0)
		padded, pt, pg := ownershipMatcherCompleteGraph(t, 4096)
		var sparseTime, paddedTime time.Duration
		if pair%2 == 0 {
			sparseTime = measure(sparse, st, sg)
			paddedTime = measure(padded, pt, pg)
		} else {
			paddedTime = measure(padded, pt, pg)
			sparseTime = measure(sparse, st, sg)
		}
		ratio := float64(paddedTime) / float64(sparseTime)
		t.Logf("OWNERSHIP_MATCHER_PAIR pair=%d queries=%d sparse_ns=%d padded_ns=%d ratio=%.4f", pair+1, queries, sparseTime.Nanoseconds(), paddedTime.Nanoseconds(), ratio)
		if paddedTime > 6*sparseTime {
			exceeded++
		}
	}
	if exceeded != 0 {
		t.Fatalf("FRESH_MATCHER_DROPPED_SOURCE_OWNERSHIP: %d/3 fresh paired ratios exceeded unchanged bound6; timing cost failure, not semantic failure", exceeded)
	}
}
