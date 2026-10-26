package transpiler

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

type resolveImport205Fixture struct {
	files          []parsing.SourceFile
	trees          []*sitter.Tree
	child, parent  *symbol.ClassScope
	qToken, rToken *symbol.ClassScope
}

func resolveImport205State(t *testing.T) {
	t.Helper()
	previousGraph, previousImports, previousIndex := symbol.GlobalScope, activeImportSourceInventory, activeResolutionFiles
	withCleanDiagnostics(t)
	t.Cleanup(func() {
		symbol.GlobalScope, activeImportSourceInventory, activeResolutionFiles = previousGraph, previousImports, previousIndex
	})
}

// All ASCII fixtures use an explicitly closed owned parser. Trees remain live
// across every measured interval; parser/tree finalizers cannot add query Cgo.
func resolveImport205Parse(t *testing.T, name, text string) (parsing.SourceFile, *sitter.Tree) {
	t.Helper()
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if tree.RootNode().HasError() {
		t.Fatal("invalid fixture Java tree")
	}
	file := parsing.SourceFile{Name: name, Source: []byte(text), Ast: tree.RootNode()}
	file.ParseSymbols()
	return file, tree
}

func resolveImport205Build(t *testing.T, width, methods int, mode string) resolveImport205Fixture {
	t.Helper()
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	var parent, child strings.Builder
	parent.WriteString("package p;public class Parent<T>{")
	child.WriteString("package p;")
	formal, header := "Token", "import static q.Owner.Token;"
	switch mode {
	case "explicit":
	case "wildcard":
		header = "import static q.Owner.*;"
	case "ordinary":
		header, formal = "import q.*;", "Owner.Token"
	case "none":
		header, formal = "", "q.Owner.Token"
	default:
		t.Fatal("unknown fixture mode")
	}
	child.WriteString(strings.Repeat(header, width))
	fmt.Fprintf(&child, "public class Child extends Parent<%s>{", formal)
	for index := 0; index < methods; index++ {
		fmt.Fprintf(&parent, "public T echo%d(T value){return value;}", index)
		fmt.Fprintf(&child, "public %s echo%d(%s value){return value;}", formal, index, formal)
	}
	parent.WriteString("}")
	child.WriteString("}")
	texts := []struct{ name, text string }{
		{"q/Owner.java", "package q;public class Owner{public static class Token{}}"},
		{"r/Owner.java", "package r;public class Owner{public static class Token{}}"},
		{"p/Parent.java", parent.String()}, {"p/Child.java", child.String()},
	}
	var f resolveImport205Fixture
	for _, row := range texts {
		file, tree := resolveImport205Parse(t, row.name, row.text)
		symbol.AddSymbolsToPackage(file.Symbols)
		f.files = append(f.files, file)
		f.trees = append(f.trees, tree)
	}
	f.child = findQualifiedSourceClass("p.Child")
	f.parent = findQualifiedSourceClass("p.Parent")
	f.qToken = findQualifiedSourceClass("q.Owner.Token")
	f.rToken = findQualifiedSourceClass("r.Owner.Token")
	if f.child == nil || f.parent == nil || f.qToken == nil || f.rToken == nil || len(allSourceClassScopes()) != 6 {
		t.Fatal("complete source graph missing")
	}
	return f
}

func (f resolveImport205Fixture) ctx() Ctx {
	return Ctx{currentFile: f.files[3].Symbols, currentClass: f.child}
}

func resolveImport205Check(t *testing.T, f resolveImport205Fixture, mode string, methods int) {
	t.Helper()
	name := "Token"
	if mode == "ordinary" {
		name = "Owner.Token"
	}
	if mode == "none" {
		name = "q.Owner.Token"
	}
	if got := resolveClassScopeByQualifiedName(f.ctx(), name); got != f.qToken {
		t.Fatalf("exact imported declaration changed: got%p want%p", got, f.qToken)
	}
	for index := 0; index < methods; index++ {
		left := f.parent.FindMethodByName(fmt.Sprintf("echo%d", index), nil)
		right := f.child.FindMethodByName(fmt.Sprintf("echo%d", index), nil)
		if left == nil || right == nil || left.Name != right.Name || len(right.Parameters) != 1 || right.Parameters[0].OriginalType != name {
			t.Fatal("override/formal contract changed")
		}
	}
	if len(Diagnostics()) != 0 {
		t.Fatal("legal graph emitted diagnostics")
	}
}

func TestResolveImport205SourceSemantics(t *testing.T) {
	for _, mode := range []string{"explicit", "wildcard", "ordinary", "none"} {
		t.Run(mode, func(t *testing.T) {
			resolveImport205State(t)
			f := resolveImport205Build(t, 128, 4, mode)
			beforeImports, beforeIndex := activeImportSourceInventory, activeResolutionFiles
			ResolveFiles(f.files)
			if activeImportSourceInventory != beforeImports || activeResolutionFiles != beforeIndex {
				t.Fatal("ResolveFiles scope leaked")
			}
			resolveImport205Check(t, f, mode, 4)
			runtime.KeepAlive(f)
		})
	}
}

func TestResolveImport205HeaderNativeBudget(t *testing.T) {
	resolveImport205State(t)
	activeImportSourceInventory, activeResolutionFiles = nil, nil
	var retained []resolveImport205Fixture
	const smallWidth, largeWidth, methods = 1, 128, 16
	// The fixed ceiling permits one cold header traversal per file plus twice
	// the small graph cost. It rejects repeated complete-header scans per lookup.
	for sample := 0; sample < 3; sample++ {
		counts := make([]int64, 2)
		for index, width := range []int{smallWidth, largeWidth} {
			f := resolveImport205Build(t, width, methods, "explicit")
			retained = append(retained, f)
			start := runtime.NumCgoCall()
			ResolveFiles(f.files)
			counts[index] = runtime.NumCgoCall() - start
			resolveImport205Check(t, f, "explicit", methods)
			if activeImportSourceInventory != nil || activeResolutionFiles != nil {
				t.Fatal("native work sample retained resolution authority")
			}
		}
		ceiling := 2*counts[0] + 64*(largeWidth-smallWidth)
		t.Logf("RESOLVE_IMPORT205_NATIVE sample=%d small_width=%d large_width=%d methods=%d small_cgo=%d large_cgo=%d ceiling=%d", sample+1, smallWidth, largeWidth, methods, counts[0], counts[1], ceiling)
		if counts[1] > ceiling {
			t.Errorf("RESOLVE_IMPORT205_REPEATED_HEADER_NATIVE: sample%d large%d > ceiling%d", sample+1, counts[1], ceiling)
		}
	}
	runtime.KeepAlive(retained)
}

type resolveImport205Hook struct {
	callback func()
	hits     int
	fired    bool
}

func (*resolveImport205Hook) Levels() []log.Level { return []log.Level{log.WarnLevel} }
func (h *resolveImport205Hook) Fire(entry *log.Entry) error {
	if entry.Message == "Unbalanced angle brackets in type string: unclosed '<'" && entry.Data["typeStr"] == "Broken<" {
		h.hits++
		if !h.fired {
			h.fired = true
			h.callback()
		}
	}
	return nil
}

func resolveImport205WithHook(t *testing.T, f resolveImport205Fixture, callback func()) *resolveImport205Hook {
	t.Helper()
	method := f.child.FindMethodByName("echo0", nil)
	if method == nil || len(method.Parameters) != 1 {
		t.Fatal("callback formal missing")
	}
	// Exercise an existing logger callback through mutable public metadata. The
	// parsed Java fixture remains unchanged; warnings must still reach the hook.
	method.Parameters[0].OriginalType = "Broken<"
	logger := log.StandardLogger()
	hooks, level := logger.Hooks, logger.GetLevel()
	t.Cleanup(func() { logger.Hooks = hooks; logger.SetLevel(level) })
	logger.Hooks = log.LevelHooks{}
	logger.SetLevel(log.WarnLevel)
	hook := &resolveImport205Hook{callback: callback}
	logger.AddHook(hook)
	return hook
}

func resolveImport205HeaderOracle(t *testing.T, ctx Ctx, owner string) {
	t.Helper()
	got := staticMethodImports(ctx)
	want := staticMethodImportsFromFile(ctx)
	if !reflect.DeepEqual(got, want) || len(got) != 1 || got[0].owner != owner || got[0].member != "Token" || got[0].wildcard {
		t.Fatalf("source tuple oracle changed: got%v want%v", got, want)
	}
	if !reflect.DeepEqual(ordinaryTypeOnDemandImports(ctx), ordinaryTypeOnDemandImportsFromFile(ctx)) {
		t.Fatal("ordinary tuple oracle changed")
	}
}

func TestResolveImport205MutationGuards(t *testing.T) {
	for _, mode := range []string{"same-length-header-body-append", "clone-root", "replace-tree-and-imports", "graph-file-namespace"} {
		t.Run(mode, func(t *testing.T) {
			resolveImport205State(t)
			f := resolveImport205Build(t, 8, 2, "explicit")
			sentinel := &struct{ label string }{mode}
			var observed any
			previousImports := &callableSubclassSourceInventory{graph: symbol.GlobalScope}
			previousIndex := newResolutionFileIndex()
			activeImportSourceInventory, activeResolutionFiles = previousImports, previousIndex
			hook := resolveImport205WithHook(t, f, func() {
				ctx := f.ctx()
				resolveImport205HeaderOracle(t, ctx, "q.Owner")
				caller := staticMethodImports(ctx)
				caller[0].owner = "caller.poison"
				resolveImport205HeaderOracle(t, ctx, "q.Owner")
				switch mode {
				case "same-length-header-body-append":
					text := strings.ReplaceAll(string(ctx.currentFile.Source), "q.Owner", "r.Owner")
					copy(ctx.currentFile.Source, text)
					resolveImport205HeaderOracle(t, ctx, "r.Owner")
					copy(ctx.currentFile.Source, strings.ReplaceAll(text, "value", "other"))
					resolveImport205HeaderOracle(t, ctx, "r.Owner")
					ctx.currentFile.Source = append(ctx.currentFile.Source, ' ')
					resolveImport205HeaderOracle(t, ctx, "r.Owner")
					if got, present := staticImportedSourceType("Token", false, ctx); !present || got != f.rToken {
						t.Fatal("header edit retained semantic answer")
					}
				case "clone-root":
					tree := f.trees[3].Copy()
					f.trees = append(f.trees, tree)
					original := ctx.currentFile.BaseClass.Class.DeclarationNode
					for i := 0; i < int(tree.RootNode().NamedChildCount()); i++ {
						node := tree.RootNode().NamedChild(i)
						if node.Type() == "class_declaration" {
							ctx.currentFile.BaseClass.Class.DeclarationNode = node
						}
					}
					if ctx.currentFile.BaseClass.Class.DeclarationNode == original {
						t.Fatal("clone premise failed")
					}
					resolveImport205HeaderOracle(t, ctx, "q.Owner")
				case "replace-tree-and-imports":
					file, tree := resolveImport205Parse(t, "p/Child.java", strings.ReplaceAll(string(ctx.currentFile.Source), "q.Owner", "r.Owner"))
					f.trees = append(f.trees, tree)
					ctx.currentFile.Source = file.Source
					ctx.currentFile.BaseClass.Class.DeclarationNode = file.Symbols.BaseClass.Class.DeclarationNode
					resolveImport205HeaderOracle(t, ctx, "r.Owner")
					empty, tree2 := resolveImport205Parse(t, "p/Child.java", "package p;public class Child{}")
					f.trees = append(f.trees, tree2)
					ctx.currentFile.Source = empty.Source
					ctx.currentFile.BaseClass.Class.DeclarationNode = empty.Symbols.BaseClass.Class.DeclarationNode
					if len(staticMethodImports(ctx)) != 0 || len(ordinaryTypeOnDemandImports(ctx)) != 0 {
						t.Fatal("removed header survived replacement")
					}
				case "graph-file-namespace":
					mismatched := ctx
					mismatched.currentClass = findQualifiedSourceClass("q.Owner")
					resolveImport205HeaderOracle(t, mismatched, "q.Owner")
					copiedFile := *ctx.currentFile
					detached := ctx
					detached.currentFile = &copiedFile
					if importSourceInventory(detached) != nil {
						t.Fatal("unregistered copied file acquired scoped authority")
					}
					resolveImport205HeaderOracle(t, detached, "q.Owner")
					oldGraph := symbol.GlobalScope
					symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
					if importSourceInventory(ctx) != nil {
						t.Fatal("replacement graph retained scoped authority")
					}
					resolveImport205HeaderOracle(t, ctx, "q.Owner")
					symbol.GlobalScope = oldGraph
					if staticMethodImports(Ctx{}) != nil || ordinaryTypeOnDemandImports(Ctx{}) != nil {
						t.Fatal("nil context acquired imports")
					}
				}
				panic(sentinel)
			})
			func() { defer func() { observed = recover() }(); ResolveFiles(f.files) }()
			if observed != sentinel || !hook.fired || hook.hits != 1 {
				t.Fatalf("wrong callback panic/warnings: %v/%d", observed, hook.hits)
			}
			if activeImportSourceInventory != previousImports || activeResolutionFiles != previousIndex {
				t.Fatal("mutation panic failed to restore outer scope")
			}
			t.Logf("RESOLVE_IMPORT205_CALLBACK mutation=%s warnings=%d", mode, hook.hits)
			runtime.KeepAlive(f)
		})
	}
}

func TestResolveImport205NestedLifetime(t *testing.T) {
	for _, mode := range []string{"same-graph", "new-graph"} {
		t.Run(mode, func(t *testing.T) {
			resolveImport205State(t)
			f := resolveImport205Build(t, 4, 2, "explicit")
			sentinel := &callableSubclassSourceInventory{graph: symbol.GlobalScope}
			index := newResolutionFileIndex()
			activeImportSourceInventory, activeResolutionFiles = sentinel, index
			var inner resolveImport205Fixture
			hook := resolveImport205WithHook(t, f, func() {
				outerImports, outerIndex, outerGraph := activeImportSourceInventory, activeResolutionFiles, symbol.GlobalScope
				resolveImport205HeaderOracle(t, f.ctx(), "q.Owner")
				if mode == "same-graph" {
					ResolveFiles(f.files[:1])
				} else {
					inner = resolveImport205Build(t, 2, 1, "wildcard")
					ResolveFiles(inner.files)
					if importSourceInventory(inner.ctx()) != nil {
						t.Fatal("restored outer holder served inner graph")
					}
					symbol.GlobalScope = outerGraph
				}
				if activeImportSourceInventory != outerImports || activeResolutionFiles != outerIndex {
					t.Fatal("nested ResolveFiles leaked authority")
				}
				resolveImport205HeaderOracle(t, f.ctx(), "q.Owner")
				ResolveFiles(nil)
				if activeImportSourceInventory != outerImports || activeResolutionFiles != outerIndex {
					t.Fatal("empty nested call displaced authority")
				}
			})
			ResolveFiles(f.files)
			if !hook.fired || hook.hits < 1 {
				t.Fatal("existing warning callback was suppressed")
			}
			if activeImportSourceInventory != sentinel || activeResolutionFiles != index {
				t.Fatal("outer ResolveFiles did not restore prior scope")
			}
			t.Logf("RESOLVE_IMPORT205_CALLBACK nested=%s warnings=%d", mode, hook.hits)
			runtime.KeepAlive(f)
			runtime.KeepAlive(inner)
		})
	}
}
