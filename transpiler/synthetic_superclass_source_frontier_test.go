package transpiler

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestSyntheticSuperclassSourceFrontier_Semantics(t *testing.T) {
	t.Run("named-anonymous-local-and-declaration-identity", func(t *testing.T) {
		h := setupParseHelper(t, `class Base<T>{} class Child extends Base<String>{} class Other{} class Use{Base<String> make(){return new Child(){};} void local(){class Local extends Child{}}}`)
		ctx := h.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		base, child, other := h.File.Symbols.FindClassScope("Base"), h.File.Symbols.FindClassScope("Child"), h.File.Symbols.FindClassScope("Other")
		binder := base.TypeParameters[0].Declaration
		for query := 0; query < 3; query++ {
			if !classHasSyntheticSubclass(base, ctx) || !classHasSyntheticSubclass(child, ctx) || classHasSyntheticSubclass(other, ctx) || classHasSyntheticSubclass(nil, ctx) || base.TypeParameters[0].Declaration != binder {
				t.Fatal("subclass source facts changed named/anonymous/local/binder identity")
			}
			ctx = ctx.Clone()
			ctx.genericFamilies = nil
		}
	})
	t.Run("fresh-target-resolution", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Base.java":  `package p;public class Base{}`,
			"q/Base.java":  `package q;public class Base{}`,
			"app/Use.java": `package app;import p.Base;public class Use{Object make(){return new Base(){};}}`,
		})
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		p, q, use := findQualifiedSourceClass("p.Base"), findQualifiedSourceClass("q.Base"), findQualifiedSourceClass("app.Use")
		file := findFileScopeForClassScope(use)
		if !classHasSyntheticSubclass(p, ctx) || classHasSyntheticSubclass(q, ctx) {
			t.Fatal("initial anonymous superclass resolution lost source owner")
		}
		file.Imports["Base"] = "q"
		if classHasSyntheticSubclass(p, ctx) || !classHasSyntheticSubclass(q, ctx) {
			t.Fatal("source inventory retained a resolved superclass/admission result")
		}
	})
	t.Run("graph-lifetime-and-fileless-fallback", func(t *testing.T) {
		h := setupParseHelper(t, `class Base{} class Use{Object make(){return new Base(){};}}`)
		ctx := h.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		old := h.File.Symbols.FindClassScope("Base")
		if !classHasSyntheticSubclass(old, ctx) {
			t.Fatal("missing first source edge")
		}
		fresh := setupParseHelper(t, `class Base{} class Use{}`)
		freshCtx := fresh.Ctx.Clone()
		freshCtx.callableSubclasses = ctx.callableSubclasses
		target := fresh.File.Symbols.FindClassScope("Base")
		if target == old || classHasSyntheticSubclass(target, freshCtx) || classHasSyntheticSubclass(old, freshCtx) || classHasSyntheticSubclass(target, Ctx{}) {
			t.Fatal("fresh graph or fileless query reused another graph's source facts")
		}
	})
	t.Run("active-local-source-file-boundary", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"app/Use.java":    `package app;public class Use{void make(){class Base{}new Base(){};}}`,
			"other/Base.java": `package other;public class Base{}`,
			"other/Use.java":  `package other;public class Use{Object make(){return new Base(){};}}`,
		})
		owner := findQualifiedSourceClass("app.Use")
		file := findFileScopeForClassScope(owner)
		var method *symbol.Definition
		for _, m := range owner.Methods {
			if m.OriginalName == "make" {
				method = m
			}
		}
		var localNode *sitter.Node
		var walk func(*sitter.Node)
		walk = func(n *sitter.Node) {
			if n.Type() == "class_declaration" {
				localNode = n
				return
			}
			for _, child := range nodeutil.NamedChildrenOf(n) {
				walk(child)
			}
		}
		walk(method.DeclarationNode)
		local := &symbol.ClassScope{Class: &symbol.Definition{Name: "HoistedBase", OriginalName: "Base", DeclarationNode: localNode}, Enclosing: owner}
		ctx.currentFile, ctx.currentClass, ctx.localScope = file, owner, method
		ctx.localClasses = map[string]*localClassInfo{"Base": {scope: local}}
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		if !classHasSyntheticSubclass(local, ctx) || !classHasSyntheticSubclass(findQualifiedSourceClass("other.Base"), ctx) {
			t.Fatal("active local source edge or another file's declaration identity was lost")
		}
		delete(ctx.localClasses, "Base")
		if classHasSyntheticSubclass(local, ctx) {
			t.Fatal("source frontier cached volatile active-local resolution")
		}
	})
}

// Both observations use the same declaration inventory and identical negative
// queries. Only unrelated complete method bodies differ. Initial syntax work is
// warmed identically; this tests the cost of repeated lowering queries after
// that work, while the existing cold allocation guards remain in force.
func TestSyntheticSuperclassSourceFrontier_WarmUnrelatedSyntaxCost(t *testing.T) {
	const queries = 128
	makeGraph := func(padded bool) (Ctx, *symbol.ClassScope, *symbol.GlobalSymbols, string) {
		var source strings.Builder
		source.WriteString("class Target{} class Padding{")
		for method := 0; method < 4; method++ {
			fmt.Fprintf(&source, "void noise%d(){int value=0;", method)
			if padded {
				source.WriteString(strings.Repeat("value++;", 4096))
			}
			source.WriteString("}")
		}
		source.WriteString("}")
		h := setupParseHelper(t, source.String())
		ctx := h.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		return ctx, h.File.Symbols.FindClassScope("Target"), symbol.GlobalScope, string(h.File.Source)
	}
	measure := func(ctx Ctx, target *symbol.ClassScope, graph *symbol.GlobalSymbols) time.Duration {
		symbol.GlobalScope = graph
		if classHasSyntheticSubclass(target, ctx) {
			t.Fatal("warmup invented subclass")
		}
		started := time.Now()
		for query := 0; query < queries; query++ {
			if classHasSyntheticSubclass(target, ctx) {
				t.Fatal("measured query invented subclass")
			}
		}
		return time.Since(started)
	}
	exceeded := 0
	for pair := 0; pair < 3; pair++ {
		sparse, st, sg, ss := makeGraph(false)
		dense, dt, dg, ds := makeGraph(true)
		if len(allSourceClassScopes()) != 2 || len(sg.Packages) != len(dg.Packages) || ss == ds || st.Class.OriginalName != dt.Class.OriginalName {
			t.Fatal("complete matched source graph control changed")
		}
		// Alternate order so one graph is not systematically favored by drift.
		var sparseTime, denseTime time.Duration
		if pair%2 == 0 {
			sparseTime = measure(sparse, st, sg)
			denseTime = measure(dense, dt, dg)
		} else {
			denseTime = measure(dense, dt, dg)
			sparseTime = measure(sparse, st, sg)
		}
		ratio := float64(denseTime) / float64(sparseTime)
		t.Logf("SYNTHETIC_FRONTIER_PAIR pair=%d queries=%d sparse_ns=%d padded_ns=%d ratio=%.4f", pair+1, queries, sparseTime.Nanoseconds(), denseTime.Nanoseconds(), ratio)
		if denseTime > 6*sparseTime {
			exceeded++
		}
	}
	if exceeded != 0 {
		t.Fatalf("SYNTHETIC_SUPERCLASS_REPEATED_UNRELATED_AST_WORK: %d/3 fresh paired ratios exceeded unchanged bound6; timing cost failure, not semantic failure", exceeded)
	}
}
