package transpiler

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// Each workload retains its actual parser/trees and owns their cleanup. Native
// finalizer work must not become an unobserved measurement normalization.
type qualifiedLookup195Fixture struct {
	parser *sitter.Parser
	trees  []*sitter.Tree
	files  []parsing.SourceFile
	ctx    Ctx
}

func qualifiedLookup195Parsed(t *testing.T, sources []string) *qualifiedLookup195Fixture {
	t.Helper()
	previous := symbol.GlobalScope
	previousIndex := activeResolutionFiles
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	f := &qualifiedLookup195Fixture{parser: sitter.NewParser()}
	runtime.SetFinalizer(f.parser, nil)
	f.parser.SetLanguage(java.GetLanguage())
	t.Cleanup(func() {
		activeResolutionFiles = previousIndex
		symbol.GlobalScope = previous
		for _, tree := range f.trees {
			tree.Close()
		}
		f.parser.Close()
	})
	for index, source := range sources {
		tree, err := f.parser.ParseCtx(context.Background(), nil, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		runtime.SetFinalizer(tree.BaseTree, nil)
		f.trees = append(f.trees, tree)
		file := parsing.SourceFile{Name: fmt.Sprintf("Fixture%d.java", index), Source: []byte(source), Ast: tree.RootNode()}
		if file.Ast.HasError() {
			t.Fatalf("invalid actual parsed Java fixture %d", index)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		f.files = append(f.files, file)
	}
	f.ctx = Ctx{currentFile: f.files[0].Symbols, currentClass: f.files[0].Symbols.BaseClass}
	return f
}

func TestQualifiedLookup195SemanticTableAndFreshMutations(t *testing.T) {
	f := qualifiedLookup195Parsed(t, []string{
		`package app; import p.Outer; class Use {static class Local {static class Leaf {}}}`,
		`package p; public class Outer {public static class Member {}}`,
		`package app; class Peer {}`,
	})
	use, outer, peer := f.files[0].Symbols.BaseClass, f.files[1].Symbols.BaseClass, f.files[2].Symbols.BaseClass
	local, member := use.Subclasses[0], outer.Subclasses[0]
	tests := []struct {
		name string
		want *symbol.ClassScope
	}{
		{"Use", use}, {"Local", local}, {"Local.Leaf", local.Subclasses[0]},
		{"Outer", outer}, {"Outer.Member", member}, {"p.Outer.Member", member}, {"Peer", peer},
		{"", nil}, {"Missing", nil}, {"Outer.Missing", nil}, {"p.Outer.Missing", nil},
		{".Outer", nil}, {"Outer.", nil}, {"Outer..Member", nil}, {"β", nil},
	}
	for _, row := range tests {
		if got := resolveClassScopeByQualifiedName(f.ctx, row.name); got != row.want {
			t.Fatalf("query %q got%p want%p", row.name, got, row.want)
		}
	}
	// Same graph, same slice length, changed declaration name: dynamic queries
	// must observe the actual current identity, not pointer/length as an epoch.
	member.Class.OriginalName = "Renamed"
	if resolveClassScopeByQualifiedName(f.ctx, "Outer.Member") != nil || resolveClassScopeByQualifiedName(f.ctx, "Outer.Renamed") != member {
		t.Fatal("fresh query retained stale member spelling")
	}
	f.ctx.currentFile.Imports["Outer"] = "missing"
	if resolveClassScopeByQualifiedName(f.ctx, "Outer") != nil {
		t.Fatal("fresh query ignored Java import rebinding")
	}
	f.ctx.currentFile.Imports["Outer"] = "p"
	if resolveClassScopeByQualifiedName(f.ctx, "Outer.Renamed") != member {
		t.Fatal("fresh query failed restored import")
	}
	old := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	if resolveClassScopeByQualifiedName(f.ctx, "Peer") != nil {
		t.Fatal("fresh graph retained same-package declaration")
	}
	symbol.GlobalScope = old
	runtime.KeepAlive(f)
}

func qualifiedLookup195Work(t *testing.T, count int) {
	t.Helper()
	sources := []string{`package p; class Use extends Base {}`, `package p; class Base {public static class Inherited {}}`}
	for index := 2; index < count; index++ {
		sources = append(sources, fmt.Sprintf(`package p; class Cell%d extends Base {public static class Member {}}`, index))
	}
	f := qualifiedLookup195Parsed(t, sources)
	session := cliownership.New()
	invocation := newSyntaxInvocation(session, f.files)
	if invocation == nil {
		t.Fatal("owned standalone syntax seam unavailable; workload must not silently use a substitute")
	}
	t.Cleanup(func() { invocation.close(); session.Close() })
	f.ctx.syntaxInvocation = invocation
	f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
	activeResolutionFiles = sourceOwnershipIndex(f.ctx)
	last := f.files[len(f.files)-1].Symbols.BaseClass
	base := f.files[1].Symbols.BaseClass
	queries := []struct {
		name string
		want *symbol.ClassScope
	}{
		{fmt.Sprintf("Cell%d", count-1), last},
		{fmt.Sprintf("Cell%d.Member", count-1), last.Subclasses[0]},
		{fmt.Sprintf("p.Cell%d.Member", count-1), last.Subclasses[0]},
		{"Inherited", base.Subclasses[0]}, {"Missing", nil}, {"p.Absent.Member", nil},
	}
	batch := func() {
		for repeat := 0; repeat < 8; repeat++ {
			for _, row := range queries {
				if got := resolveClassScopeByQualifiedName(f.ctx, row.name); got != row.want {
					t.Fatalf("work query %q got%p want%p", row.name, got, row.want)
				}
			}
		}
		runtime.KeepAlive(f)
	}
	batch() // Warm wrappers/source facts outside every measurement.
	allocations := testing.AllocsPerRun(3, batch)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	cgoBefore := runtime.NumCgoCall()
	batch()
	cgoCalls := runtime.NumCgoCall() - cgoBefore
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	data, err := json.Marshal(map[string]any{
		"schema": "qualified-lookup195-native-baseline-v1", "actual_parsed_files": count,
		"queries_per_batch": len(queries) * 8, "allocs_per_batch": allocations,
		"measured_batch_cgo_calls": cgoCalls, "measured_batch_totalalloc_bytes": after.TotalAlloc - before.TotalAlloc,
		"measured_batch_mallocs": after.Mallocs - before.Mallocs, "elapsed_nanoseconds_observation_only": elapsed.Nanoseconds(),
		"index_construction_not_yet_implemented_or_claimed_free": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("QUALIFIED_LOOKUP195_BASELINE " + string(data))
	runtime.KeepAlive(f)
}

func TestQualifiedLookup195ParsedWork50(t *testing.T)  { qualifiedLookup195Work(t, 50) }
func TestQualifiedLookup195ParsedWork200(t *testing.T) { qualifiedLookup195Work(t, 200) }
func TestQualifiedLookup195ParsedWork800(t *testing.T) { qualifiedLookup195Work(t, 800) }
