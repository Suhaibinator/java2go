package transpiler

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Test-only decomposition. Stored contexts are deliberately measurement snapshots,
// not the proposed production index and not authority after semantic mutation.
type demand188Event struct {
	node *sitter.Node
	ctx  Ctx
	file *symbol.FileScope
}
type demand188Counts struct{ Nodes, Candidates, Owners int }
type demand188Row struct {
	Phase          string
	Iterations     int
	Cgo            int64
	Mallocs, Bytes uint64
	Seeds          []string
	Counts         demand188Counts
}

func demand188Fixture(t *testing.T, sources map[string]string) (Ctx, []parsing.SourceFile, *cliownership.Session) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	names := make([]string, 0, len(sources))
	for n := range sources {
		names = append(names, n)
	}
	sort.Strings(names)
	files := make([]parsing.SourceFile, 0, len(names))
	for _, n := range names {
		file := parsing.SourceFile{Name: n, Source: []byte(sources[n])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		if file.Ast.HasError() {
			t.Fatalf("invalid calibration Java %s", n)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	for _, file := range files {
		ResolveFile(file)
	}
	session := cliownership.New()
	t.Cleanup(session.Close)
	ctx := Ctx{currentFile: files[0].Symbols, currentClass: files[0].Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}}
	ctx.syntaxInvocation = newSyntaxInvocation(session, files)
	if ctx.syntaxInvocation == nil {
		t.Fatal("owned fixture rejected")
	}
	t.Cleanup(ctx.syntaxInvocation.close)
	return ctx, files, session
}

func demand188Names(seeds []*symbol.ClassScope) []string {
	names := make([]string, len(seeds))
	for n, s := range seeds {
		names[n] = qualifiedSourceClassName(s)
	}
	return names
}
func demand188Same(t *testing.T, a, b []*symbol.ClassScope) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("seed length %v != %v", demand188Names(a), demand188Names(b))
	}
	for n := range a {
		if a[n] != b[n] {
			t.Fatal("actual source declaration identity changed")
		}
	}
}
func demand188Measure(t *testing.T, phase string, n int, fn func() []*symbol.ClassScope) demand188Row {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cgo := runtime.NumCgoCall()
	var got []*symbol.ClassScope
	for i := 0; i < n; i++ {
		got = fn()
	}
	cgo = runtime.NumCgoCall() - cgo
	runtime.ReadMemStats(&after)
	row := demand188Row{Phase: phase, Iterations: n, Cgo: cgo, Mallocs: after.Mallocs - before.Mallocs, Bytes: after.TotalAlloc - before.TotalAlloc, Seeds: demand188Names(got)}
	encoded, _ := json.Marshal(row)
	t.Log("DEMAND188_WORK", string(encoded))
	return row
}

// Faithful current walk/pruning/context routing with semantic demand resolution
// omitted. This isolates the traversal envelope; it does not make seed decisions.
func demand188SyntaxWitness(ctx Ctx, collect bool) ([]demand188Event, demand188Counts) {
	var events []demand188Event
	var counts demand188Counts
	for _, owner := range allSourceClassScopes() {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			continue
		}
		counts.Owners++
		root := owner.Class.DeclarationNode
		var walk func(*sitter.Node, Ctx)
		walk = func(node *sitter.Node, lexical Ctx) {
			if node == nil {
				return
			}
			if sourceGenericViewTypeDeclaration(node, ctx) && !node.Equal(root) {
				return
			}
			if resolvedSourceNodeType(node, ctx) == "class_body" {
				parent := node.Parent()
				if parent != nil && (resolvedSourceNodeType(parent, ctx) == "object_creation_expression" || resolvedSourceNodeType(parent, ctx) == "enum_constant") {
					return
				}
			}
			if resolvedSourceNodeType(node, ctx) == "method_declaration" || resolvedSourceNodeType(node, ctx) == "constructor_declaration" {
				lexical = lexical.Clone()
				lexical.localScope = nil
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.Equal(node) {
						lexical.localScope = method
						break
					}
				}
			}
			counts.Nodes++
			switch resolvedSourceNodeType(node, lexical) {
			case "generic_type", "type_identifier", "scoped_type_identifier":
				counts.Candidates++
				if collect {
					events = append(events, demand188Event{node, lexical, file})
				}
			}
			for _, child := range resolvedSourceNamedChildren(node, ctx) {
				childCtx := lexical
				if sourceGenericViewTypeDeclaration(node, ctx) {
					childCtx = classDeclarationChildCtx(node, child, lexical)
				}
				walk(child, childCtx)
			}
		}
		walk(root, classScopeCtx(owner, ctx))
	}
	return events, counts
}
func demand188Replay(events []demand188Event) []*symbol.ClassScope {
	seen := map[*symbol.ClassScope]bool{}
	var seeds []*symbol.ClassScope
	for _, event := range events {
		if seed := sourceGenericViewDemandType(event.node, event.ctx, event.file.Source); seed != nil && !seen[seed] {
			seen[seed] = true
			seeds = append(seeds, seed)
		}
	}
	sort.Slice(seeds, func(i, j int) bool { return qualifiedSourceClassName(seeds[i]) < qualifiedSourceClassName(seeds[j]) })
	return seeds
}

func TestDemandSyntax188Calibration(t *testing.T) {
	for _, noise := range []int{0, 8} {
		t.Run(fmt.Sprintf("noiseFiles%d", noise), func(t *testing.T) {
			sources := map[string]string{
				"p/Cell.java":  `package p;public class Cell<T>{T value;}`,
				"q/Cell.java":  `package q;public class Cell<T>{T value;}`,
				"app/Use.java": `package app;import p.Cell;public class Use extends p.Cell<String>{Cell<?> field;Object cast(Object value){return (Cell<?>)value;}<Cell>void shadow(Object value){Cell own=(Cell)value;}void local(){class Cell<T>{}Cell<?> own;p.Cell<?> qualified;}Object anon(){return new Object(){Cell<?> hidden;};}static class Nested<T>{T value;}record Pair(int value){}}`,
			}
			for i := 0; i < noise; i++ {
				var b strings.Builder
				fmt.Fprintf(&b, "package noise;public class Padding%d{", i)
				for m := 0; m < 8; m++ {
					fmt.Fprintf(&b, "int method%d(int value){", m)
					for x := 0; x < 128; x++ {
						b.WriteString("value++;")
					}
					b.WriteString("return value;}")
				}
				b.WriteString("}")
				sources[fmt.Sprintf("noise/Padding%d.java", i)] = b.String()
			}
			ctx, files, session := demand188Fixture(t, sources)
			var cold []*symbol.ClassScope
			demand188Measure(t, "actual_api_cold", 1, func() []*symbol.ClassScope { cold = sourceGenericViewDemandSeeds(ctx); return cold })
			if len(cold) != 1 || cold[0] != findQualifiedSourceClass("p.Cell") {
				t.Fatalf("unexpected actual seeds %v", demand188Names(cold))
			}
			demand188Measure(t, "actual_api_warm", 16, func() []*symbol.ClassScope {
				got := sourceGenericViewDemandSeeds(ctx)
				demand188Same(t, got, cold)
				return got
			})
			events, counts := demand188SyntaxWitness(ctx, true)
			demand188Same(t, demand188Replay(events), cold)
			encoded, _ := json.Marshal(counts)
			t.Log("DEMAND188_STRUCTURE", string(encoded))
			demand188Measure(t, "structural_walk_only_warm", 16, func() []*symbol.ClassScope {
				_, got := demand188SyntaxWitness(ctx, false)
				if got != counts {
					t.Fatal("structural witness changed")
				}
				return nil
			})
			demand188Measure(t, "semantic_event_replay_optimistic_lower_bound", 16, func() []*symbol.ClassScope { got := demand188Replay(events); demand188Same(t, got, cold); return got })
			// This is the existing ownership guard, measured separately. Production event
			// replay must pay this cost plus live routing, not only the optimistic replay.
			demand188Measure(t, "existing_whole_invocation_guard", 16, func() []*symbol.ClassScope {
				if _, ok := ctx.syntaxInvocation.queryBodies("calibration"); !ok {
					t.Fatal("owned graph lost")
				}
				return nil
			})
			ctx.syntaxInvocation.close()
			if ctx.syntaxInvocation.active {
				t.Fatal("close left active invocation")
			}
			if _, ok := ctx.syntaxInvocation.queryBodies("calibration"); ok {
				t.Fatal("retired invocation accepted")
			}
			demand188Same(t, sourceGenericViewDemandSeeds(ctx), cold)
			session.Close()
			if session.Active() {
				t.Fatal("session remained active")
			}
			runtime.KeepAlive(files)
		})
	}
}

func TestDemandSyntax188LiveSemanticShields(t *testing.T) {
	t.Run("live-import", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "q/Cell.java": `package q;public class Cell<T>{T value;}`, "app/Use.java": `package app;import p.Cell;public class Use{Cell<?> value;}`})
		p, q := findQualifiedSourceClass("p.Cell"), findQualifiedSourceClass("q.Cell")
		use := findQualifiedSourceClass("app.Use")
		file := findFileScopeForClassScope(use, ctx)
		before := sourceGenericViewDemandSeeds(ctx)
		if len(before) != 1 || before[0] != p {
			t.Fatal("import baseline")
		}
		file.Imports["Cell"] = "q"
		after := sourceGenericViewDemandSeeds(ctx)
		if len(after) != 1 || after[0] != q {
			t.Fatalf("live import did not change actual API owner: %v imports%v", demand188Names(after), file.Imports)
		}
		file.Imports["Cell"] = "p"
		demand188Same(t, sourceGenericViewDemandSeeds(ctx), before)
		t.Log("DEMAND188_SHIELD live-import p.Cell -> q.Cell -> p.Cell")
	})
	t.Run("live-method-binder", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": `package app;import p.Cell;public class Use{<Cell>void shadow(Object value){Cell local=(Cell)value;}}`})
		use := findQualifiedSourceClass("app.Use")
		var method *symbol.Definition
		for _, m := range use.Methods {
			if m.OriginalName == "shadow" {
				method = m
			}
		}
		if method == nil || len(method.TypeParameters) != 1 {
			t.Fatal("missing actual method binder")
		}
		if got := sourceGenericViewDemandSeeds(ctx); len(got) != 0 {
			t.Fatal("binder did not shield")
		}
		saved := method.TypeParameters
		method.TypeParameters = nil
		got := sourceGenericViewDemandSeeds(ctx)
		if len(got) != 1 || got[0] != findQualifiedSourceClass("p.Cell") {
			t.Fatalf("live binder replacement not observed: %v", demand188Names(got))
		}
		method.TypeParameters = saved
		if len(sourceGenericViewDemandSeeds(ctx)) != 0 {
			t.Fatal("restored binder not observed")
		}
		t.Log("DEMAND188_SHIELD live-binder empty -> p.Cell -> empty")
	})
	t.Run("header-body-and-root-provenance", func(t *testing.T) {
		ctx, files, session := demand188Fixture(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": `package app;public class Use extends p.Cell<String>{static class p{static class Cell<T>{T value;}}p.Cell<?> body;}`})
		use := findQualifiedSourceClass("app.Use")
		base := classScopeCtx(use, ctx)
		root := use.Class.DeclarationNode
		header := classHeaderTypeCtx(use, base)
		body := classScopeCtx(use, base)
		hp := resolveClassScopeByQualifiedName(header, "p.Cell")
		bp := resolveClassScopeByQualifiedName(body, "p.Cell")
		if hp != findQualifiedSourceClass("p.Cell") || bp == nil || bp == hp {
			t.Fatal("actual header/body source identities collapsed")
		}
		got := sourceGenericViewDemandSeeds(ctx)
		if len(got) != 1 || got[0] != bp {
			t.Fatalf("actual demand header/body route %v", demand188Names(got))
		}
		flipped := body.Clone()
		flipped.memberTypeHeaderOwner = use
		if resolveClassScopeByQualifiedName(flipped, "p.Cell") != hp {
			t.Fatal("live header input ignored")
		}
		flipped.memberTypeHeaderOwner = nil
		if resolveClassScopeByQualifiedName(flipped, "p.Cell") != bp {
			t.Fatal("live body restoration ignored")
		}
		if _, ok := ctx.syntaxInvocation.queryBodies("guard"); !ok {
			t.Fatal("initial root provenance rejected")
		}
		raw := files[0].Symbols.Source
		saved := raw[0]
		raw[0] = 'x'
		if _, ok := ctx.syntaxInvocation.queryBodies("guard"); ok {
			t.Fatal("source byte mutation accepted")
		}
		raw[0] = saved
		if _, ok := ctx.syntaxInvocation.queryBodies("guard"); !ok {
			t.Fatal("restored source rejected")
		}
		ctx.syntaxInvocation.close()
		session.Close()
		if _, ok := ctx.syntaxInvocation.queryBodies("guard"); ok {
			t.Fatal("retired root accepted")
		}
		if use.Class.DeclarationNode != root {
			t.Fatal("diagnostic mutated source root")
		}
		t.Log("DEMAND188_SHIELD header/body live-input distinct; bytes rejected; invocation/session retired")
	})
}
