package transpiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/printer"
	"go/token"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type demandPredicate170Fixture struct {
	files []parsing.SourceFile
	ctx   Ctx
}

func demandPredicate170Setup(t *testing.T, sources map[string]string, resolve bool) demandPredicate170Fixture {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous; setStrictMode(false); resetDiagnostics() })
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	f := demandPredicate170Fixture{}
	for _, name := range names {
		file := parsing.SourceFile{Name: name, Source: []byte(sources[name])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		f.files = append(f.files, file)
	}
	if resolve {
		ResolveFiles(f.files)
	}
	if len(f.files) != 0 {
		f.ctx = Ctx{currentFile: f.files[0].Symbols, currentClass: f.files[0].Symbols.BaseClass}
	}
	return f
}

type demandPredicate170Owner struct {
	scope *symbol.ClassScope
	class *symbol.Definition
	node  *sitter.Node
	file  *symbol.FileScope
}

type demandPredicate170Inputs struct {
	sources []string
	trees   []string
	roots   []*sitter.Node
	imports []map[string]string
	owners  []demandPredicate170Owner
}

func demandPredicate170Snapshot(f demandPredicate170Fixture) demandPredicate170Inputs {
	s := demandPredicate170Inputs{}
	for _, file := range f.files {
		s.sources = append(s.sources, string(file.Source))
		s.trees = append(s.trees, file.Ast.String())
		s.roots = append(s.roots, file.Symbols.ParsedRoot())
		imports := map[string]string{}
		for name, path := range file.Symbols.Imports {
			imports[name] = path
		}
		s.imports = append(s.imports, imports)
	}
	for _, scope := range allSourceClassScopes() {
		var node *sitter.Node
		if scope.Class != nil {
			node = scope.Class.DeclarationNode
		}
		s.owners = append(s.owners, demandPredicate170Owner{scope, scope.Class, node, findFileScopeForClassScope(scope, f.ctx)})
	}
	sort.Slice(s.owners, func(i, j int) bool {
		return qualifiedSourceClassName(s.owners[i].scope) < qualifiedSourceClassName(s.owners[j].scope)
	})
	return s
}

type demandPredicate170Outcome struct {
	seeds      []*symbol.ClassScope
	diags      []Diagnostic
	panicValue string
}

func demandPredicate170Capture(fn func(Ctx) []*symbol.ClassScope, ctx Ctx, strict bool) (out demandPredicate170Outcome) {
	setStrictMode(strict)
	resetDiagnostics()
	defer func() {
		if value := recover(); value != nil {
			out.panicValue = fmt.Sprintf("%T:%v", value, value)
		}
		out.diags = Diagnostics()
		setStrictMode(false)
	}()
	out.seeds = fn(ctx)
	return
}

func demandPredicate170Compare(t *testing.T, f demandPredicate170Fixture, ctx Ctx, strict bool) demandPredicate170Outcome {
	t.Helper()
	before := demandPredicate170Snapshot(f)
	oldCtx := ctx.Clone()
	oldCtx.callableSubclasses = &callableSubclassSourceInventory{}
	old := demandPredicate170Capture(demandPredicate170OldSeeds, oldCtx, strict)
	if !reflect.DeepEqual(before, demandPredicate170Snapshot(f)) {
		t.Fatal("old seed walk mutated source/tree/import/owner identity")
	}
	newCtx := ctx.Clone()
	newCtx.callableSubclasses = &callableSubclassSourceInventory{}
	got := demandPredicate170Capture(sourceGenericViewDemandSeeds, newCtx, strict)
	if !reflect.DeepEqual(got, old) {
		t.Fatalf("seed pointer/order/diagnostic/panic mismatch: got %#v old %#v", got, old)
	}
	if !reflect.DeepEqual(before, demandPredicate170Snapshot(f)) {
		t.Fatal("seed walk mutated source/tree/import/owner identity")
	}
	return got
}

func TestDemandPredicate170DeclarationBoundaries(t *testing.T) {
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java": `package p; public class Cell<T>{T value; public T get(){return null;}}`,
		"q/Cell.java": `package q; public class Cell<T>{T value;}`,
		"app/Use.java": `package app; import p.Cell;
@interface Tag {}
@Tag class Use<X>{
 Cell<?> field; Object cast(Object value){return (Cell)value;}
 <Cell> Cell binder(Object value){return (Cell)value;}
 static class Nested<Y>{Cell<?> nested;}
 class Inner<Z>{Cell<?> inner;}
 record Rec<R>(Cell<?> value){}
 interface Face{Cell<?> read();}
 enum E{A{q.Cell<?> anonymousOnly; Object read(){return null;}}; Cell<?> value;}
 void local(){class Cell{} class Local{q.Cell<?> localOnly;} Cell value=null; Object callback=new Object(){class Cell{} Cell value; q.Cell<?> anonymousOnly;};}
 void empty(){} void noise(int value){value++;value++;String label="δ unicode";}
}
`,
	}, true)
	p := findQualifiedSourceClass("p.Cell")
	if p == nil {
		t.Fatal("live source Cell missing")
	}
	for _, cached := range []bool{false, true} {
		ctx := f.ctx.Clone()
		if cached {
			ctx.callableSubclasses = &callableSubclassSourceInventory{}
			sourceGenericViewDemandSeeds(ctx)
		}
		out := demandPredicate170Compare(t, f, ctx, false)
		if out.panicValue != "" {
			t.Fatalf("valid source walk panic %s", out.panicValue)
		}
		found := false
		for _, seed := range out.seeds {
			found = found || seed == p
		}
		if !found {
			t.Fatal("positive source demand disappeared")
		}
		for _, seed := range out.seeds {
			if seed == findQualifiedSourceClass("q.Cell") {
				t.Fatal("local or anonymous-only demand crossed declaration boundary")
			}
		}
	}
	if len(allSourceClassScopes()) < 9 {
		t.Fatal("input declarations were discarded")
	}
}

func TestDemandPredicate170LiveBinderImportAndGlobalMutation(t *testing.T) {
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java":  `package p; public class Cell<T>{T value;}`,
		"q/Cell.java":  `package q; public class Cell<T>{T value;}`,
		"app/Use.java": `package app; import p.Cell; class Use{Cell<?> field; <B> Cell<?> read(B value){return null;}}`,
	}, true)
	p, q := findQualifiedSourceClass("p.Cell"), findQualifiedSourceClass("q.Cell")
	use := findQualifiedSourceClass("app.Use")
	if p == nil || q == nil || use == nil {
		t.Fatal("missing mutation fixture declarations")
	}
	file := findFileScopeForClassScope(use, f.ctx)
	ctx := f.ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	assertOnly := func(want *symbol.ClassScope) {
		t.Helper()
		out := demandPredicate170Compare(t, f, ctx, false)
		if len(out.seeds) != 1 || out.seeds[0] != want {
			t.Fatalf("live binding selected %#v instead of %p", out.seeds, want)
		}
	}
	assertOnly(p)
	file.Imports["Cell"] = "q"
	assertOnly(q)
	file.Imports["Cell"] = "p"
	assertOnly(p)
	method := use.Methods[0]
	for _, candidate := range use.Methods {
		if candidate.OriginalName == "read" {
			method = candidate
		}
	}
	if len(method.TypeParameters) != 1 {
		t.Fatal("live method binder missing")
	}
	oldName := method.TypeParameters[0].Name
	method.TypeParameters[0].Name = "Cell"
	demandPredicate170Compare(t, f, ctx, false)
	method.TypeParameters[0].Name = oldName
	delete(file.Imports, "Cell")
	demandPredicate170Compare(t, f, ctx, false)
	file.Imports["Cell"] = "p"
	oldGraph := symbol.GlobalScope
	g := demandPredicate170Setup(t, map[string]string{"late/Late.java": `package late; class Late<T>{T value;} class Use{Late<?> field;}`}, true)
	late := findQualifiedSourceClass("late.Late")
	if late == nil {
		t.Fatal("new graph not registered")
	}
	out := demandPredicate170Compare(t, g, ctx, false)
	if len(out.seeds) != 1 || out.seeds[0] != late {
		t.Fatal("old context retained old graph seeds")
	}
	symbol.GlobalScope = oldGraph
	assertOnly(p)
}

func TestDemandPredicate170MalformedUnknownAndDiagnostics(t *testing.T) {
	for _, source := range []string{`class Broken<T>{Object value=(;}`, `class Unknown{foreign.Missing<?> value;}`, `class Unicode<T>{String text="δ"; T value;}`} {
		f := demandPredicate170Setup(t, map[string]string{"Shield.java": source}, false)
		for _, strict := range []bool{false, true} {
			demandPredicate170Compare(t, f, f.ctx, strict)
		}
	}
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java":  `package p; public class Cell<T>{T value;}`,
		"q/Cell.java":  `package q; public class Cell<T>{T value;}`,
		"app/Use.java": `package app; import p.*; import q.*; class Use{Cell value;}`,
	}, false)
	for _, strict := range []bool{false, true} {
		out := demandPredicate170Compare(t, f, f.ctx, strict)
		if len(out.diags) == 0 {
			t.Fatal("ambiguous type diagnostics disappeared")
		}
		if strict && out.panicValue == "" {
			t.Fatal("strict first diagnostic no longer panics")
		}
	}
}

func TestDemandPredicate170GeneratedASTAndImports(t *testing.T) {
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java":  `package p; public class Cell<T>{public T get(){return null;}}`,
		"app/Use.java": `package app; import p.Cell; public class Use{Cell<?> value; Object read(Object input){return (Cell)input;}}`,
	}, true)
	render := func(fn func(Ctx) []*symbol.ClassScope) ([]*symbol.ClassScope, string, []Diagnostic) {
		ctx := f.ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		ctx.genericFamilies = &genericFamilyAnalysis{}
		setStrictMode(false)
		resetDiagnostics()
		seeds := fn(ctx)
		var output bytes.Buffer
		for _, file := range f.files {
			local := ctx.Clone()
			local.currentFile = file.Symbols
			local.currentClass = file.Symbols.BaseClass
			node, err := convertFileNode(file, local)
			if err != nil {
				t.Fatal(err)
			}
			if err := printer.Fprint(&output, token.NewFileSet(), node); err != nil {
				t.Fatal(err)
			}
		}
		return seeds, output.String(), Diagnostics()
	}
	oldSeeds, oldAST, oldDiags := render(demandPredicate170OldSeeds)
	seeds, generated, diags := render(sourceGenericViewDemandSeeds)
	if !reflect.DeepEqual(seeds, oldSeeds) || generated != oldAST || !reflect.DeepEqual(diags, oldDiags) {
		t.Fatal("generated AST/import/diagnostic oracle mismatch")
	}
	sum := sha256.Sum256([]byte(generated))
	t.Log("GENERATED_AST170 " + hex.EncodeToString(sum[:]))
}

type demandPredicate170Work struct {
	Name   string
	Allocs float64
	Bytes  uint64
	Cgo    int64
	Scopes int
}

func demandPredicate170Measure(t *testing.T) demandPredicate170Work {
	t.Helper()
	noise := "package app; class Noise{void empty(){} void noise(int value){" + strings.Repeat("value++;", 256) + "}}"
	f := demandPredicate170Setup(t, map[string]string{
		"p/Cell.java":    `package p; public class Cell<T>{public T get(){return null;}}`,
		"q/Other.java":   `package q; public class Other<T>{T value;}`,
		"app/Use.java":   `package app; import p.Cell; class Use{Cell<?> value;}`,
		"app/Noise.java": noise,
	}, true)
	ctx := f.ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	cell := findQualifiedSourceClass("p.Cell")
	if len(allSourceClassScopes()) != 4 || cell == nil {
		t.Fatal("workload source classes must all remain registered")
	}
	work := func() {
		seeds := sourceGenericViewDemandSeeds(ctx)
		if len(seeds) != 1 || seeds[0] != cell {
			t.Fatal("workload demand identity changed")
		}
	}
	work()
	demandPredicate170OldSeeds(ctx)
	allocs := testing.AllocsPerRun(5, work)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cgo := runtime.NumCgoCall()
	for i := 0; i < 5; i++ {
		work()
	}
	cgo = runtime.NumCgoCall() - cgo
	runtime.ReadMemStats(&after)
	r := demandPredicate170Work{"live-four-class-getter-and-noisy-body", allocs, (after.TotalAlloc - before.TotalAlloc) / 5, cgo / 5, len(allSourceClassScopes())}
	encoded, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("DEMAND_WORK170 " + string(encoded))
	runtime.KeepAlive(f)
	return r
}

func TestDemandPredicate170WorkBaseline(t *testing.T) { demandPredicate170Measure(t) }
