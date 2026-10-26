package transpiler

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// All fields imported below are genuine source declarations. The measured
// consumer asks for a distinct name, so it must read the complete import list
// without resolving any imported owner or member candidate.
func importSource195Context(t *testing.T, count int) Ctx {
	t.Helper()
	var owner, active strings.Builder
	owner.WriteString("package q;public class Owner{")
	active.WriteString("package p;")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&owner, "public static int field%03d=%d;", i, i)
		fmt.Fprintf(&active, "import static q.Owner.field%03d;", i)
	}
	owner.WriteString("}")
	active.WriteString("public class Active{}")
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Active.java": active.String(), "q/Owner.java": owner.String(),
	})
	ctx.currentClass = findQualifiedSourceClass("p.Active")
	ctx.currentFile = findFileScopeForClassScope(ctx.currentClass)
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	if ctx.currentClass == nil || ctx.currentFile == nil || len(allSourceClassScopes()) != 2 {
		t.Fatal("complete two-class source graph was not retained")
	}
	imports := staticMethodImports(ctx)
	if len(imports) != count {
		t.Fatalf("source imports=%d, want %d", len(imports), count)
	}
	for i, entry := range imports {
		if entry.owner != "q.Owner" || entry.member != fmt.Sprintf("field%03d", i) || entry.wildcard {
			t.Fatalf("ordered import %d = %#v", i, entry)
		}
	}
	if got, present := staticImportedSourceType("Unimported", false, ctx); got != nil || present {
		t.Fatal("unimported name acquired a source type")
	}
	return ctx
}

func TestImportSource195ReadOnlyConsumerBytesGrowth(t *testing.T) {
	const queries = 1024
	type measured struct {
		bytes, cgo int64
	}
	measure := func(count int) measured {
		ctx := importSource195Context(t, count)
		// Warm both tuple and consumer paths; construction and cold scans are
		// outside this hot-query budget, and are not claimed to be eliminated.
		for i := 0; i < 32; i++ {
			if got, present := staticImportedSourceType("Unimported", false, ctx); got != nil || present {
				t.Fatal("warm query changed the source result")
			}
		}
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		calls := runtime.NumCgoCall()
		for i := 0; i < queries; i++ {
			if got, present := staticImportedSourceType("Unimported", false, ctx); got != nil || present {
				t.Fatal("measured query changed the source result")
			}
		}
		calls = runtime.NumCgoCall() - calls
		runtime.ReadMemStats(&after)
		runtime.KeepAlive(ctx)
		return measured{bytes: int64(after.TotalAlloc-before.TotalAlloc) / queries, cgo: calls / queries}
	}
	small, large := measure(64), measure(512)
	t.Logf("warm existing source-type miss, bytes/query64=%d bytes/query512=%d cgo/query64=%d cgo/query512=%d; queries=%d; no forced GC", small.bytes, large.bytes, small.cgo, large.cgo, queries)
	budget := small.bytes
	if budget < 256 {
		budget = 256
	}
	if large.bytes > 6*budget {
		t.Fatalf("IMPORT_SOURCE195_READ_ONLY_COPY_GROWTH: bytes/query512 %d >6*max(bytes/query64 %d,256)", large.bytes, small.bytes)
	}
}

func TestImportSource195ParentAndSourceGuards(t *testing.T) {
	t.Run("same-tree-parent-identity-and-file-pairing", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 64)
		declaration := ctx.currentFile.BaseClass.Class.DeclarationNode
		root := declaration
		for {
			first, second := root.Parent(), root.Parent()
			if first != second || first != nil && !first.Equal(second) {
				t.Fatal("pinned same-tree Parent did not reuse the same node wrapper")
			}
			if first == nil {
				break
			}
			root = first
		}
		for i := 0; i < 16; i++ {
			if !reflect.DeepEqual(staticMethodImports(ctx), staticImportInventoryExpectedTuples()) {
				t.Fatal("same-tree warm query changed tuple order or duplicate elimination")
			}
		}
		otherClass := ctx
		otherClass.currentClass = findQualifiedSourceClass("q.Owner")
		if !reflect.DeepEqual(staticMethodImports(otherClass), staticImportInventoryExpectedTuples()) {
			t.Fatal("current class displaced current file's imports")
		}
		ownerCtx := classScopeCtx(otherClass.currentClass, ctx)
		if len(staticMethodImports(ownerCtx)) != 0 {
			t.Fatal("cross-file context borrowed caller imports")
		}
	})
	t.Run("copy-header-body-and-reparse-mutations", func(t *testing.T) {
		text := campaignImportSource("alpha")
		ctx := campaignImportControlContext(t, text)
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		campaignAssertImportText(t, ctx, "alpha")
		copied := staticMethodImports(ctx)
		copied[0].owner = "caller.poison"
		ordinary := ordinaryTypeOnDemandImports(ctx)
		ordinary[0] = "caller.poison"
		campaignAssertImportText(t, ctx, "alpha")
		// The body-only byte change has the same length and header. The old
		// walkers see the same imports, without granting any semantic cache.
		copy(ctx.currentFile.Source, strings.Replace(text, "Member", "Worker", 1))
		campaignAssertImportText(t, ctx, "alpha")
		copy(ctx.currentFile.Source, strings.ReplaceAll(text, "alpha", "bravo"))
		campaignAssertImportText(t, ctx, "bravo")
		oldDeclaration := ctx.currentFile.BaseClass.Class.DeclarationNode
		campaignReplaceImportTree(t, ctx, text)
		if ctx.currentFile.BaseClass.Class.DeclarationNode == oldDeclaration {
			t.Fatal("reparse control reused original declaration pointer")
		}
		campaignAssertImportText(t, ctx, "alpha")
		campaignReplaceImportTree(t, ctx, strings.Replace(text, "class Active", "import extra.*;import static extra.Owner.answer;class Active", 1))
		if got := ordinaryTypeOnDemandImports(ctx); !reflect.DeepEqual(got, []string{"alpha", "extra"}) {
			t.Fatalf("appended import result=%v", got)
		}
		if got := staticMethodImports(ctx); len(got) != 3 || got[2].owner != "extra.Owner" {
			t.Fatalf("appended static import result=%v", got)
		}
		campaignReplaceImportTree(t, ctx, "package p;class Active<T>{}")
		if len(staticMethodImports(ctx)) != 0 || len(ordinaryTypeOnDemandImports(ctx)) != 0 {
			t.Fatal("removed imports survived")
		}
		ctx.currentFile.Source = append(ctx.currentFile.Source, []byte(" ")...)
		if len(staticMethodImports(ctx)) != 0 || len(ordinaryTypeOnDemandImports(ctx)) != 0 {
			t.Fatal("appended body bytes invented imports")
		}
	})
}

func TestImportSource195BareContextAndGraphLifetime(t *testing.T) {
	oldImports, oldOwnership := activeImportSourceInventory, activeResolutionFiles
	t.Cleanup(func() { activeImportSourceInventory, activeResolutionFiles = oldImports, oldOwnership })
	ctx := campaignImportControlContext(t, campaignImportSource("alpha"))
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	activeImportSourceInventory = resolvedSourceInventory(ctx)
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	fresh := ctx
	fresh.callableSubclasses = nil
	if importSourceInventory(fresh) != activeImportSourceInventory {
		t.Fatal("registered bare context lost scoped syntax inventory")
	}
	campaignAssertImportText(t, fresh, "alpha")
	activeResolutionFiles.files = nil
	if importSourceInventory(fresh) != nil {
		t.Fatal("unregistered bare context borrowed scoped facts")
	}
	campaignAssertImportText(t, fresh, "alpha")
	activeImportSourceInventory, activeResolutionFiles = nil, nil
	campaignAssertImportText(t, fresh, "alpha")
	if fresh.callableSubclasses != nil || importSourceInventory(fresh) != nil {
		t.Fatal("standalone fallback acquired retained inventory")
	}
	// Build a real replacement graph while keeping the old explicit holder.
	reused := ctx.callableSubclasses
	oldGraph := symbol.GlobalScope
	next := campaignImportControlContext(t, campaignImportSource("bravo"))
	if symbol.GlobalScope == oldGraph {
		t.Fatal("graph replacement premise failed")
	}
	next.callableSubclasses = reused
	campaignAssertImportText(t, next, "bravo")
	if reused.graph != symbol.GlobalScope {
		t.Fatal("explicit source holder did not follow graph replacement")
	}
	activeImportSourceInventory = &callableSubclassSourceInventory{graph: oldGraph}
	activeResolutionFiles = sourceOwnershipIndex(next)
	next.callableSubclasses = nil
	if importSourceInventory(next) != nil {
		t.Fatal("stale global graph authorized bare syntax inventory")
	}
	campaignAssertImportText(t, next, "bravo")
	if staticMethodImports(Ctx{}) != nil || ordinaryTypeOnDemandImports(Ctx{}) != nil {
		t.Fatal("nil context invented imports")
	}
}
