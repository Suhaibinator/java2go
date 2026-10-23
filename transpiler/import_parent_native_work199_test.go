package transpiler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// Frozen PUBLIC199 root oracle. This intentionally retains the duplicate native
// Parent reads, and runs outside the measured existing-API query loop.
func importParent199OldRoot(node *sitter.Node) *sitter.Node {
	for node.Parent() != nil {
		node = node.Parent()
	}
	return node
}

// The native counter is process-global. Own the parser explicitly and retain
// all trees until the parent test returns, so parser/tree finalizers from prior
// samples cannot add unrelated C calls to a later sample. This changes fixture
// lifetime only; every measured query still uses the production existing API.
func importParent199OwnedContext(t *testing.T, parser *sitter.Parser, active string, retained *[]*sitter.Tree) Ctx {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	sources := []struct{ name, source string }{
		{"alpha/Owner.java", "package alpha;public class Owner{public static int answer(){return 11;}}"},
		{"bravo/Owner.java", "package bravo;public class Owner{public static int answer(){return 19;}}"},
		{"extra/Owner.java", "package extra;public class Owner{public static int answer(){return 23;}}"},
		{"p/Active.java", active},
	}
	var files []parsing.SourceFile
	for _, source := range sources {
		tree, err := parser.ParseCtx(context.Background(), nil, []byte(source.source))
		if err != nil {
			t.Fatal(err)
		}
		*retained = append(*retained, tree)
		file := parsing.SourceFile{Name: source.name, Source: []byte(source.source), Ast: tree.RootNode()}
		if file.Ast.HasError() {
			t.Fatal("legal import fixture did not parse")
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	for _, file := range files {
		ResolveFile(file)
	}
	owner := findQualifiedSourceClass("p.Active")
	return Ctx{currentFile: findFileScopeForClassScope(owner), currentClass: owner}
}

// The measured API is the real warmed importSourceFacts guard, including its
// existing source-prefix and root-bounds checks. No replacement guard or cached
// answer is used. Native work is bounded by one parent read per edge, one final
// nil-parent read, and the two unchanged structural-bounds observations.
func TestImportParentNativeWork199(t *testing.T) {
	const queries = 1024
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	defer parser.Close()
	var retainedTrees []*sitter.Tree
	defer func() { runtime.KeepAlive(retainedTrees) }()
	for _, padding := range []int{0, 64, 512} {
		for _, anchor := range []string{"top", "nested", "root"} {
			t.Run(fmt.Sprintf("padding%d/%s", padding, anchor), func(t *testing.T) {
				text := strings.Replace(campaignImportSource("alpha"), "class Active", strings.Repeat("import alpha.*;", padding)+"class Active", 1)
				ctx := importParent199OwnedContext(t, parser, text, &retainedTrees)
				ctx.callableSubclasses = &callableSubclassSourceInventory{}
				original := ctx.currentFile.BaseClass.Class.DeclarationNode
				if original == nil || len(ctx.currentClass.Subclasses) != 1 {
					t.Fatal("legal top and nested declarations are required")
				}
				switch anchor {
				case "nested":
					ctx.currentFile.BaseClass.Class.DeclarationNode = ctx.currentClass.Subclasses[0].Class.DeclarationNode
				case "root":
					ctx.currentFile.BaseClass.Class.DeclarationNode = importParent199OldRoot(original)
				}
				defer func() { ctx.currentFile.BaseClass.Class.DeclarationNode = original }()
				declaration := ctx.currentFile.BaseClass.Class.DeclarationNode
				root := importParent199OldRoot(declaration)
				depth := 0
				for node := declaration; ; {
					parent := node.Parent()
					if parent == nil {
						break
					}
					depth++
					node = parent
				}
				if anchor == "root" && depth != 0 || anchor != "root" && depth == 0 {
					t.Fatalf("anchor %s has unexpected depth %d", anchor, depth)
				}
				campaignAssertImportText(t, ctx, "alpha")
				inventory, want, ok := importSourceFacts(ctx)
				if !ok || inventory == nil || !want.importsReady || !want.ordinaryReady || want.root != root || want.declaration != declaration {
					t.Fatal("warm existing-API facts did not preserve the exact old root")
				}
				clone := ctx.Clone()
				if clone.callableSubclasses != ctx.callableSubclasses {
					t.Fatal("clone lost the existing shared syntax holder")
				}
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				calls := runtime.NumCgoCall()
				for query := 0; query < queries; query++ {
					gotInventory, got, known := importSourceFacts(clone)
					if !known || gotInventory != inventory || got.root != root || got.declaration != declaration ||
						got.rootEnd != want.rootEnd || got.rootChildren != want.rootChildren ||
						got.sourceLength != want.sourceLength || !bytes.Equal(got.prefix, want.prefix) ||
						!got.importsReady || !got.ordinaryReady || len(got.imports) != len(want.imports) || len(got.ordinary) != len(want.ordinary) {
						t.Fatal("real warmed guard changed exact source facts")
					}
				}
				calls = runtime.NumCgoCall() - calls
				runtime.ReadMemStats(&after)
				budget := int64(queries * (depth + 3))
				row, err := json.Marshal(map[string]any{
					"queries": queries, "padding": padding, "anchor": anchor, "parent_edges": depth,
					"actual_native_calls": calls, "native_budget": budget,
					"old_source_native_prediction": queries * (2*depth + 3),
					"total_alloc_bytes":            after.TotalAlloc - before.TotalAlloc, "mallocs": after.Mallocs - before.Mallocs,
					"real_existing_API": "importSourceFacts", "fresh_semantic_result_cache": false,
				})
				if err != nil {
					t.Fatal(err)
				}
				t.Log("IMPORT_PARENT_NATIVE_WORK199", string(row))
				campaignAssertImportText(t, ctx, "alpha")
				if !reflect.DeepEqual(want.imports, staticMethodImports(ctx)) || !reflect.DeepEqual(want.ordinary, ordinaryTypeOnDemandImports(ctx)) || inventory.graph != symbol.GlobalScope {
					t.Fatal("measured source tuples or graph ownership changed")
				}
				runtime.KeepAlive(ctx)
				if calls > budget {
					t.Errorf("IMPORT_PARENT_DUPLICATE_NATIVE_READ199: calls %d exceed one-parent-per-edge guard budget %d", calls, budget)
				}
			})
		}
	}
}
