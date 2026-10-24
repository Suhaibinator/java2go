package transpiler

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

type nonDeclaration209Fixture struct {
	files   []parsing.SourceFile
	trees   []*sitter.Tree
	ctx     Ctx
	session *cliownership.Session
	owner   *symbol.ClassScope
	method  *symbol.Definition
}

// The parser is explicitly closed by the existing parse helper; retain all
// trees until every work-counter sample is complete so finalizers add no Cgo.
func nonDeclaration209Build(t *testing.T, width int) nonDeclaration209Fixture {
	return nonDeclaration209BuildMode(t, width, false)
}

func nonDeclaration209BuildMode(t *testing.T, width int, rejectAnonymous bool) nonDeclaration209Fixture {
	t.Helper()
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	cell := "package p;public class Cell<T>{T value;}"
	noiseWidth := width
	if rejectAnonymous {
		// Unsupported invariant List<T> storage supplies the same actual rejection
		// witness as q.Cell. All Java syntax is valid; body work remains observable.
		cell = "package p;public class Cell<T>{T value;java.util.List<T> unsupported;Object early=new Cell<Object>(){};void noise(){int x=0;" + strings.Repeat("x=x+1;", width) + "}}"
		noiseWidth = 1
	}
	texts := []struct{ name, text string }{
		{"p/Cell.java", cell},
		{"q/Cell.java", "package q;public class Cell<T>{java.util.List<T> value;}"},
		{"app/Factory.java", "package app;import p.Cell;public interface Factory{<S>Cell<S> adapt(Cell<S> value);}"},
		{"app/Use.java", "package app;public class Use{p.Cell<?> field;}"},
		{"noise/Noise.java", "package noise;public class Noise{void work(){int x=0;" + strings.Repeat("x=x+1;", noiseWidth) + "}}"},
	}
	var f nonDeclaration209Fixture
	for _, row := range texts {
		file, tree := resolveImport205Parse(t, row.name, row.text)
		f.files = append(f.files, file)
		f.trees = append(f.trees, tree)
		symbol.AddSymbolsToPackage(file.Symbols)
	}
	ResolveFiles(f.files)
	f.owner = findQualifiedSourceClass("app.Factory")
	if f.owner == nil {
		t.Fatal("missing declaring owner")
	}
	for _, method := range f.owner.Methods {
		if method.OriginalName == "adapt" {
			f.method = method
		}
	}
	if f.method == nil || len(f.method.TypeParameters) != 1 || len(f.method.Parameters) != 1 || !f.method.RequiresHelper {
		t.Fatal("real generic method contract absent")
	}
	f.session = cliownership.New()
	t.Cleanup(f.session.Close)
	f.ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), Ctx{callableSubclasses: &callableSubclassSourceInventory{}})
	f.ctx.syntaxInvocation = newSyntaxInvocation(f.session, f.files)
	if f.ctx.syntaxInvocation == nil || !sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("owned syntax admission absent")
	}
	t.Cleanup(f.ctx.syntaxInvocation.close)
	// This is the existing native caller state at the selected seam: its
	// preceding direct-owner check has populated the existing render analysis.
	// Warm through the real API rather than installing an expected plan answer.
	f.ctx.genericFamilies = &genericFamilyAnalysis{}
	invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
	if !f.ctx.genericFamilies.ready {
		t.Fatal("existing API did not warm its render analysis")
	}
	if _, bypass := directOwnerMethodParameterInterfaceErasure(f.owner, f.method, 0, f.ctx); bypass {
		t.Fatal("fixture bypassed selected erased-entry seam")
	}
	if f.method.IsPrivate || executionParameterIsVariadic(f.method, 0) || len(f.method.TypeParameters[0].Bounds) != 0 || !javaTypeContainsParameter(f.method.OriginalType, f.method.TypeParameters[0].Name) || strings.TrimSpace(f.method.OriginalType) == f.method.TypeParameters[0].Name {
		t.Fatal("fixture did not reach nested erased-entry branch")
	}
	if genericMethodHasErasedEntry(f.method) == rejectAnonymous {
		t.Fatal("nested signature canonical/anonymous-rejection contract changed")
	}
	return f
}

func (f nonDeclaration209Fixture) resolution() *methodResolution {
	return &methodResolution{def: f.method, owner: f.owner}
}

func nonDeclaration209Agree(t *testing.T, f nonDeclaration209Fixture, ctx Ctx) string {
	t.Helper()
	// Independent existing dynamic contract: derive the declaring signature and
	// erasure from the selected definition, never caller lexical or cached plans.
	want := definitionParameterJavaSignatureType(f.method, 0)
	if genericMethodHasErasedEntry(f.method) {
		want = genericMethodErasedJavaType(f.method, want)
	}
	got := invocationPhysicalParameterJavaType(f.resolution(), 0, ctx)
	if got != want {
		t.Fatalf("physical signature %q want current declaring contract %q", got, want)
	}
	return got
}

func TestNonDeclaration209DeclaringContract(t *testing.T) {
	resolveImport205State(t)
	f := nonDeclaration209Build(t, 4)
	want := nonDeclaration209Agree(t, f, f.ctx)
	if !strings.Contains(want, "Cell<") || !strings.Contains(want, "java.lang.Object") || strings.Contains(want, "<S>") {
		t.Fatalf("non-erased nested contract %q", want)
	}
	caller := f.ctx.Clone()
	caller.genericFamilies = &genericFamilyAnalysis{ready: true}
	caller.localScope = &symbol.Definition{TypeParameters: f.method.TypeParameters}
	caller.syntheticTypeParameters = f.method.TypeParameters
	caller.expectedType = "q.Cell<S>"
	caller.importAliases = map[string]string{"kept/path": "kept"}
	caller.usedImports = map[string]bool{"kept/path": true}
	aliases, used := maps.Clone(caller.importAliases), maps.Clone(caller.usedImports)
	if nonDeclaration209Agree(t, f, caller) != want {
		t.Fatal("caller context escaped into declaring signature")
	}
	if !reflect.DeepEqual(aliases, caller.importAliases) || !reflect.DeepEqual(used, caller.usedImports) || len(caller.genericFamilies.plans) != 0 {
		t.Fatal("query changed caller answers/import ownership")
	}
	for _, index := range []int{-1, 1, 9} {
		if invocationPhysicalParameterJavaType(f.resolution(), index, caller) != "" {
			t.Fatal("argument boundary changed")
		}
	}
	if invocationPhysicalParameterJavaType(nil, 0, caller) != "" {
		t.Fatal("nil resolution changed")
	}
	saved := f.method.RequiresHelper
	f.method.RequiresHelper = false
	nonDeclaration209Agree(t, f, caller)
	f.method.RequiresHelper = saved
	if nonDeclaration209Agree(t, f, caller) != want {
		t.Fatal("restored helper contract changed")
	}
	runtime.KeepAlive(f)
}

func TestNonDeclaration209LiveImportsAndDefinitions(t *testing.T) {
	resolveImport205State(t)
	f := nonDeclaration209Build(t, 4)
	nonDeclaration209Agree(t, f, f.ctx)
	file := findFileScopeForClassScope(f.owner, Ctx{})
	saved := file.Imports["Cell"]
	file.Imports["Cell"] = "q"
	if genericMethodHasErasedEntry(f.method) {
		t.Fatal("unsupported live imported family unexpectedly admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	file.Imports["Cell"] = saved
	if !genericMethodHasErasedEntry(f.method) {
		t.Fatal("restored import lost canonical contract")
	}
	savedMethods := f.owner.Methods
	copyMethod := *f.method
	f.owner.Methods = []*symbol.Definition{&copyMethod}
	if genericMethodHasErasedEntry(f.method) {
		t.Fatal("removed definition retained declaring identity")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	f.owner.Methods = savedMethods
	nonDeclaration209Agree(t, f, f.ctx)
	runtime.KeepAlive(f)
}

func TestNonDeclaration209SourceRootAndGraphGuards(t *testing.T) {
	resolveImport205State(t)
	f := nonDeclaration209Build(t, 4)
	nonDeclaration209Agree(t, f, f.ctx)
	file := f.files[0].Symbols
	at := bytes.Index(file.Source, []byte("value"))
	if at < 0 {
		t.Fatal("fixture source missing")
	}
	file.Source[at] = 'w'
	if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("same-length source edit admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	file.Source[at] = 'v'
	source := file.Source
	file.Source = append([]byte(nil), source...)
	nonDeclaration209Agree(t, f, f.ctx)
	file.Source = source
	twin, tree := resolveImport205Parse(t, "Twin.java", string(f.files[2].Source))
	defer tree.Close()
	root := f.owner.Class.DeclarationNode
	f.owner.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
	nonDeclaration209Agree(t, f, f.ctx)
	f.owner.Class.DeclarationNode = root
	graph := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	nonDeclaration209Agree(t, f, f.ctx)
	symbol.GlobalScope = graph
	savedFiles := graph.Packages["app"].Files
	duplicateFiles := maps.Clone(savedFiles)
	for _, file := range savedFiles {
		duplicateFiles["duplicate.java"] = file
		break
	}
	graph.Packages["app"].Files = duplicateFiles
	if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("duplicate registered file admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	graph.Packages["app"].Files = savedFiles
	nonDeclaration209Agree(t, f, f.ctx)
	runtime.KeepAlive(f)
}

func TestNonDeclaration209LifetimeAndPublicFallback(t *testing.T) {
	resolveImport205State(t)
	f := nonDeclaration209Build(t, 4)
	want := nonDeclaration209Agree(t, f, f.ctx)
	public := f.ctx.Clone()
	public.syntaxInvocation = nil
	public.callableSubclasses = nil
	if nonDeclaration209Agree(t, f, public) != want {
		t.Fatal("public dynamic contract changed")
	}
	logger := log.StandardLogger()
	output := logger.Out
	logger.Out = io.Discard
	if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("extensible logger admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	logger.Out = output
	other := newSyntaxInvocation(f.session, f.files)
	if other == nil {
		t.Fatal("nested owned invocation rejected")
	}
	nested := f.ctx.Clone()
	nested.syntaxInvocation = other
	nested.callableSubclasses = &callableSubclassSourceInventory{}
	nonDeclaration209Agree(t, f, nested)
	other.close()
	nonDeclaration209Agree(t, f, nested)
	if !f.ctx.syntaxInvocation.active {
		t.Fatal("nested close retired outer invocation")
	}
	f.ctx.syntaxInvocation.close()
	nonDeclaration209Agree(t, f, f.ctx)
	f.session.Close()
	if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("closed session admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	runtime.KeepAlive(f)
}

func TestNonDeclaration209EditedTreeAndPanic(t *testing.T) {
	resolveImport205State(t)
	f := nonDeclaration209Build(t, 4)
	nonDeclaration209Agree(t, f, f.ctx)
	at := uint32(bytes.Index(f.files[0].Source, []byte("value")))
	f.trees[0].Edit(sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}})
	if !f.files[0].Symbols.ParsedRoot().HasChanges() || sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("edited parse tree was admitted")
	}
	nonDeclaration209Agree(t, f, f.ctx)
	public := f.ctx.Clone()
	public.syntaxInvocation, public.callableSubclasses = nil, nil
	invoke := func(ctx Ctx) (value, panicValue string) {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicValue = fmt.Sprintf("%T:%v", recovered, recovered)
			}
		}()
		return invocationPhysicalParameterJavaType(f.resolution(), 0, ctx), ""
	}
	file := f.files[2].Symbols
	source := file.Source
	file.Source = nil
	want, wantPanic := invoke(public)
	got, gotPanic := invoke(f.ctx)
	file.Source = source
	if got != want || gotPanic != wantPanic {
		t.Fatalf("malformed-source fallback changed: (%q,%q) want (%q,%q)", got, gotPanic, want, wantPanic)
	}
	if !f.ctx.syntaxInvocation.active || !f.session.Active() {
		t.Fatal("query panic retired outer lifetime")
	}
	nonDeclaration209Agree(t, f, public)
	runtime.KeepAlive(f)
}

// Compare the same existing API in an admitted context and its legacy dynamic
// context. The two contexts differ only in syntaxInvocation/callableSubclasses.
// Each width uses ABBA or BAAB chunks, eight calls each: Q16 per arm.
func TestNonDeclaration209ExistingAPINativeBudget(t *testing.T) {
	resolveImport205State(t)
	var retained []nonDeclaration209Fixture
	const queries = 16
	for sample := 1; sample <= 3; sample++ {
		var counts [2][2]int64 // width, then dynamic/owned
		var orders [2]string
		for index, width := range []int{1, 128} {
			f := nonDeclaration209BuildMode(t, width, true)
			retained = append(retained, f)
			owned := f.ctx
			dynamic := owned
			dynamic.syntaxInvocation = nil
			dynamic.callableSubclasses = nil
			// Both arms are warm before any counter window. All parser Close calls
			// occurred in construction, and every owned tree stays retained below.
			want := nonDeclaration209Agree(t, f, owned)
			if nonDeclaration209Agree(t, f, dynamic) != want || want != definitionParameterJavaSignatureType(f.method, 0) || genericMethodHasErasedEntry(f.method) {
				t.Fatal("anonymous rejection did not preserve the current source signature")
			}
			order := [4]int{0, 1, 1, 0}
			orders[index] = "DOOD"
			if (sample+index)%2 == 1 {
				order = [4]int{1, 0, 0, 1}
				orders[index] = "ODDO"
			}
			for _, arm := range order {
				ctx := dynamic
				if arm == 1 {
					ctx = owned
				}
				start := runtime.NumCgoCall()
				for query := 0; query < queries/2; query++ {
					if got := invocationPhysicalParameterJavaType(f.resolution(), 0, ctx); got != want {
						t.Fatalf("query changed physical signature %q want%q", got, want)
					}
				}
				counts[index][arm] += runtime.NumCgoCall() - start
			}
		}
		smallDynamic, smallOwned, largeDynamic, largeOwned := counts[0][0], counts[0][1], counts[1][0], counts[1][1]
		if smallDynamic <= 0 || largeDynamic <= 0 {
			t.Fatal("native witness absent")
		}
		ceiling := 9 * largeDynamic / 10
		t.Logf("ERASED_CONTEXT205_NATIVE sample=%d small_width=1 large_width=128 queries=%d small_order=%s large_order=%s small_dynamic_cgo=%d small_owned_cgo=%d large_dynamic_cgo=%d large_owned_cgo=%d ceiling=%d small_native_ratio=%.6f large_native_ratio=%.6f", sample, queries, orders[0], orders[1], smallDynamic, smallOwned, largeDynamic, largeOwned, ceiling, float64(smallOwned)/float64(smallDynamic), float64(largeOwned)/float64(largeDynamic))
		if largeOwned > ceiling {
			t.Errorf("ERASED_CONTEXT205_REPEATED_SYNTAX_NATIVE: sample%d owned%d > ceiling%d dynamic%d", sample, largeOwned, ceiling, largeDynamic)
		}
	}
	runtime.KeepAlive(retained)
}

func nonDeclaration209Outcome(f nonDeclaration209Fixture, ctx Ctx) (value, failure string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			failure = fmt.Sprintf("%T:%v", recovered, recovered)
		}
	}()
	return invocationPhysicalParameterJavaType(f.resolution(), 0, ctx), ""
}

func TestNonDeclaration209FreshOwnershipAndReusedHolder(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(fmt.Sprint("class-move-active-", active), func(t *testing.T) {
			resolveImport205State(t)
			f := nonDeclaration209Build(t, 4)
			if _, known := sourceGenericViewDemandSeedsFromSyntax(f.ctx); !known {
				t.Fatal("initial owned routes absent")
			}
			sourceOwnershipIndex(f.ctx)
			if active {
				activeResolutionFiles = newResolutionFileIndex()
			}
			left, right := f.files[4].Symbols, f.files[3].Symbols
			oldLeft, oldRight := left.TopLevelClasses, right.TopLevelClasses
			defer func() { left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight }()
			moved := oldLeft[0]
			left.TopLevelClasses = nil
			right.TopLevelClasses = append(append([]*symbol.ClassScope(nil), oldRight...), moved)
			if !sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
				t.Fatal("control did not isolate topology from unchanged file/source/root anchors")
			}
			public := f.ctx.Clone()
			public.syntaxInvocation = nil
			public.callableSubclasses = nil
			want, wantFailure := nonDeclaration209Outcome(f, public)
			got, gotFailure := nonDeclaration209Outcome(f, f.ctx)
			if got != want || gotFailure != wantFailure {
				t.Fatalf("moved class changed dynamic outcome (%q,%q) want(%q,%q)", got, gotFailure, want, wantFailure)
			}
			left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight
			nonDeclaration209Agree(t, f, f.ctx)
			runtime.KeepAlive(f)
		})
	}
	t.Run("nondeclaring-root", func(t *testing.T) {
		resolveImport205State(t)
		f := nonDeclaration209Build(t, 4)
		if _, known := sourceGenericViewDemandSeedsFromSyntax(f.ctx); !known {
			t.Fatal("initial routes absent")
		}
		sourceOwnershipIndex(f.ctx)
		owner := f.files[4].Symbols.BaseClass
		root := owner.Class.DeclarationNode
		twin, tree := resolveImport205Parse(t, "TwinNoise.java", string(f.files[4].Source))
		defer tree.Close()
		owner.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
		defer func() { owner.Class.DeclarationNode = root }()
		public := f.ctx.Clone()
		public.syntaxInvocation = nil
		public.callableSubclasses = nil
		want, wantFailure := nonDeclaration209Outcome(f, public)
		got, gotFailure := nonDeclaration209Outcome(f, f.ctx)
		if got != want || gotFailure != wantFailure {
			t.Fatal("nondeclaring foreign root changed dynamic outcome")
		}
		owner.Class.DeclarationNode = root
		nonDeclaration209Agree(t, f, f.ctx)
		runtime.KeepAlive(f)
	})
	t.Run("closed-old-fresh-invocation-reused-holder", func(t *testing.T) {
		resolveImport205State(t)
		f := nonDeclaration209Build(t, 4)
		sourceGenericViewDemandSeedsFromSyntax(f.ctx)
		sourceOwnershipIndex(f.ctx)
		holder := f.ctx.callableSubclasses
		holder.selectorsReady = true
		holder.selectors = map[string]struct{}{"__old_source_selector": {}}
		f.ctx.syntaxInvocation.close()
		replacement, tree := resolveImport205Parse(t, "p/Cell.java", "package p;public class Cell<T>{java.util.List<T> unsupported;}")
		defer tree.Close()
		for key, file := range symbol.GlobalScope.Packages["p"].Files {
			if file == f.files[0].Symbols {
				symbol.GlobalScope.Packages["p"].Files[key] = replacement.Symbols
			}
		}
		files := append([]parsing.SourceFile(nil), f.files...)
		files[0] = replacement
		fresh := newSyntaxInvocation(f.session, files)
		if fresh == nil || !sourceGenericDemandOwnedFiles(fresh) {
			t.Fatal("fresh reparsed owned invocation absent")
		}
		defer fresh.close()
		ctx := f.ctx.Clone()
		ctx.syntaxInvocation = fresh
		ctx.genericFamilies = &genericFamilyAnalysis{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, ctx)
		if !ctx.genericFamilies.ready {
			t.Fatal("fresh render analysis did not warm through existing API")
		}
		oldNodes := maps.Clone(holder.nodes)
		oldOwnership := holder.ownership
		oldSelectors := maps.Clone(holder.selectors)
		if genericMethodHasErasedEntry(f.method) {
			t.Fatal("reparsed unsupported family retained old admission")
		}
		nonDeclaration209Agree(t, f, ctx)
		if holder.ownership != oldOwnership || !reflect.DeepEqual(holder.nodes, oldNodes) || !reflect.DeepEqual(holder.selectors, oldSelectors) {
			t.Fatal("query borrowed or changed stale caller holder")
		}
		runtime.KeepAlive(f)
		runtime.KeepAlive(files)
	})
}

func metadata209BuildMode(t *testing.T, width int, rejectAnonymous bool, metadata string) nonDeclaration209Fixture {
	t.Helper()
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	cell := "package p;public class Cell<T>{T value;}"
	noiseWidth := width
	if rejectAnonymous {
		// Unsupported invariant List<T> storage supplies the same actual rejection
		// witness as q.Cell. All Java syntax is valid; body work remains observable.
		cell = "package p;public class Cell<T>{T value;java.util.List<T> unsupported;Object early=new Cell<Object>(){};void noise(){int x=0;" + strings.Repeat("x=x+1;", width) + "}}"
		noiseWidth = 1
	}
	texts := []struct{ name, text string }{
		{"p/Cell.java", cell},
		{"q/Cell.java", "package q;public class Cell<T>{java.util.List<T> value;}"},
		{"app/Factory.java", "package app;import p.Cell;public interface Factory{<S>Cell<S> adapt(Cell<S> value);}"},
		{"app/Use.java", "package app;public class Use{p.Cell<?> field;}"},
		{"noise/Noise.java", "package noise;public class Noise{void work(){int x=0;" + strings.Repeat("x=x+1;", noiseWidth) + "}}"},
	}
	texts = append(texts, struct{ name, text string }{"meta/Marker.java", "package meta;public @interface Marker{}"}, struct{ name, text string }{"p/package-info.java", metadata})
	var f nonDeclaration209Fixture
	var resolving []parsing.SourceFile
	for _, row := range texts {
		file, tree := metadata209ParseUnit(t, row.name, row.text)
		f.files = append(f.files, file)
		f.trees = append(f.trees, tree)
		symbol.AddSymbolsToPackage(file.Symbols)
		if file.Symbols.BaseClass != nil {
			resolving = append(resolving, file)
		}
	}
	ResolveFiles(resolving)
	f.owner = findQualifiedSourceClass("app.Factory")
	if f.owner == nil {
		t.Fatal("missing declaring owner")
	}
	for _, method := range f.owner.Methods {
		if method.OriginalName == "adapt" {
			f.method = method
		}
	}
	if f.method == nil || len(f.method.TypeParameters) != 1 || len(f.method.Parameters) != 1 || !f.method.RequiresHelper {
		t.Fatal("real generic method contract absent")
	}
	f.session = cliownership.New()
	t.Cleanup(f.session.Close)
	f.ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), Ctx{callableSubclasses: &callableSubclassSourceInventory{}})
	f.ctx.syntaxInvocation = newSyntaxInvocation(f.session, f.files)
	if f.ctx.syntaxInvocation == nil || sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
		t.Fatal("metadata invocation absent or legacy count guard unexpectedly true")
	}
	t.Cleanup(f.ctx.syntaxInvocation.close)
	// This is the existing native caller state at the selected seam: its
	// preceding direct-owner check has populated the existing render analysis.
	// Warm through the real API rather than installing an expected plan answer.
	f.ctx.genericFamilies = &genericFamilyAnalysis{}
	invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
	if !f.ctx.genericFamilies.ready {
		t.Fatal("existing API did not warm its render analysis")
	}
	if _, bypass := directOwnerMethodParameterInterfaceErasure(f.owner, f.method, 0, f.ctx); bypass {
		t.Fatal("fixture bypassed selected erased-entry seam")
	}
	if f.method.IsPrivate || executionParameterIsVariadic(f.method, 0) || len(f.method.TypeParameters[0].Bounds) != 0 || !javaTypeContainsParameter(f.method.OriginalType, f.method.TypeParameters[0].Name) || strings.TrimSpace(f.method.OriginalType) == f.method.TypeParameters[0].Name {
		t.Fatal("fixture did not reach nested erased-entry branch")
	}
	if genericMethodHasErasedEntry(f.method) == rejectAnonymous {
		t.Fatal("nested signature canonical/anonymous-rejection contract changed")
	}
	return f
}

func metadata209Compare(t *testing.T, f nonDeclaration209Fixture) string {
	t.Helper()
	owned, ownedFailure := nonDeclaration209Outcome(f, f.ctx)
	dynamic := f.ctx
	dynamic.syntaxInvocation, dynamic.callableSubclasses = nil, nil
	dynamicValue, dynamicFailure := nonDeclaration209Outcome(f, dynamic)
	// This helper derives the expected signature from the current definition
	// and the legacy declaring-context query, independently of both arms.
	want := nonDeclaration209Agree(t, f, dynamic)
	if ownedFailure != "" || dynamicFailure != "" || owned != want || dynamicValue != want {
		t.Fatalf("fresh dynamic contract changed owned(%q,%q) dynamic(%q,%q)", owned, ownedFailure, dynamicValue, dynamicFailure)
	}
	return owned
}

func metadata209ParseUnit(t *testing.T, name, text string) (parsing.SourceFile, *sitter.Tree) {
	t.Helper()
	file, tree := resolveImport205Parse(t, name, text)
	unit, err := classifyCompilationUnit(file)
	if err != nil {
		t.Fatal(err)
	}
	if unit.packageMetadata {
		file.Symbols.Package = unit.packageName
	}
	return file, tree
}

func TestMetadata209SemanticFamilies(t *testing.T) {
	for _, metadata := range []string{"package p;", "@meta.Marker package p;import q.Cell;", "@Marker package p;import meta.Marker;import q.Cell;", "package p;import q.*;", "package p;import static java.lang.Integer.*;"} {
		t.Run(fmt.Sprint("metadata-", len(metadata)), func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, metadata)
			want := metadata209Compare(t, f)
			if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
				t.Fatal("legacy all-file guard changed")
			}
			meta := f.files[len(f.files)-1].Symbols
			savedMarker, hadMarker := meta.Imports["Marker"]
			meta.Imports["Marker"] = "q"
			if metadata209Compare(t, f) != want {
				t.Fatal("metadata annotation-import provider leaked")
			}
			if hadMarker {
				meta.Imports["Marker"] = savedMarker
			} else {
				delete(meta.Imports, "Marker")
			}
			if metadata209Compare(t, f) != want {
				t.Fatal("metadata annotation-import provider restore leaked")
			}
			meta.Imports["Cell"] = "q"
			if metadata209Compare(t, f) != want {
				t.Fatal("metadata import namespace leaked")
			}
			meta.Imports["Cell"] = "p"
			if metadata209Compare(t, f) != want {
				t.Fatal("metadata provider restoration changed owner")
			}
			ownerFile := findFileScopeForClassScope(f.owner, Ctx{})
			saved := ownerFile.Imports["Cell"]
			ownerFile.Imports["Cell"] = "q"
			if genericMethodHasErasedEntry(f.method) {
				t.Fatal("fresh unsupported owner provider not observed")
			}
			metadata209Compare(t, f)
			ownerFile.Imports["Cell"] = saved
			if metadata209Compare(t, f) != want {
				t.Fatal("restored owner provider lost current signature")
			}
			runtime.KeepAlive(f)
		})
	}
	t.Run("fresh-owner-binder-shadow", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@Marker package p;import meta.Marker;import q.Cell;")
		want := metadata209Compare(t, f)
		saved := f.owner.TypeParameters
		f.owner.TypeParameters = []symbol.TypeParam{{Name: "Cell"}}
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		metadata209Compare(t, f)
		f.owner.TypeParameters = saved
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		if metadata209Compare(t, f) != want {
			t.Fatal("restored owner binder did not restore dynamic signature")
		}
		runtime.KeepAlive(f)
	})

	t.Run("metadata-classliteral-dynamic-only", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@meta.Marker package p;import q.Cell;")
		want := metadata209Compare(t, f)
		// Replace both declaration and metadata with legal current trees. The old
		// invocation must remain ineligible; current dynamic semantics still apply.
		marker, mt := resolveImport205Parse(t, "meta/Marker.java", "package meta;public @interface Marker{Class<?> value();}")
		meta, pt := metadata209ParseUnit(t, "p/package-info.java", "@meta.Marker(q.Cell.class) package p;import q.Cell;")
		f.trees = append(f.trees, mt, pt)
		symbol.GlobalScope.Packages["meta"].Files["Marker"] = marker.Symbols
		f.files[len(f.files)-2], f.files[len(f.files)-1] = marker, meta
		if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
			t.Fatal("old invocation accepted replacement files")
		}
		metadata209Compare(t, f)
		// A fresh invocation owns this legal classliteral root, but the new proposed
		// nondemand capability must reject it in candidate-specific direct guards.
		fresh := newSyntaxInvocation(f.session, f.files)
		if fresh == nil {
			t.Fatal("fresh classliteral setup rejected")
		}
		defer fresh.close()
		f.ctx.syntaxInvocation = fresh
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		if metadata209Compare(t, f) != want {
			t.Fatal("package annotation classliteral affected source owner")
		}
		runtime.KeepAlive(f)
	})
	t.Run("same-length-metadata-source-edit", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@meta.Marker package p;import q.Cell;")
		meta := f.files[len(f.files)-1].Symbols
		at := bytes.Index(meta.Source, []byte("q.Cell"))
		if at < 0 {
			t.Fatal("missing metadata import")
		}
		meta.Source[at] = 'p'
		metadata209Compare(t, f)
		meta.Source[at] = 'q'
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("metadata-to-class-and-unregistered-class", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		replacement, tree := resolveImport205Parse(t, "p/Added.java", "package p;class Added{q.Cell<?> demanded;}")
		f.trees = append(f.trees, tree)
		f.files[len(f.files)-1] = replacement
		fresh := newSyntaxInvocation(f.session, f.files)
		if fresh == nil {
			t.Fatal("owned unregistered class setup rejected")
		}
		defer fresh.close()
		f.ctx.syntaxInvocation = fresh
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		if sourceGenericDemandOwnedFiles(fresh) {
			t.Fatal("unregistered class unexpectedly admitted")
		}
		metadata209Compare(t, f)
		symbol.AddSymbolsToPackage(replacement.Symbols)
		if !sourceGenericDemandOwnedFiles(fresh) {
			t.Fatal("fully registered owned class graph should satisfy the legacy guard")
		}
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("closed-nested-and-same-graph-fresh", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@meta.Marker package p;import q.Cell;")
		metadata209Compare(t, f)
		old := f.ctx.syntaxInvocation
		old.close()
		metadata209Compare(t, f)
		fresh := newSyntaxInvocation(f.session, f.files)
		if fresh == nil {
			t.Fatal("same graph fresh invocation absent")
		}
		defer fresh.close()
		f.ctx.syntaxInvocation = fresh
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		metadata209Compare(t, f)
		if old.active || old.demandSyntax != nil {
			t.Fatal("fresh invocation revived retired old syntax")
		}
		f.session.Close()
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
}

func TestMetadata209ExistingAPINativeBudget(t *testing.T) {
	resolveImport205State(t)
	var retained []nonDeclaration209Fixture
	const queries = 16
	for sample := 1; sample <= 3; sample++ {
		var counts [2][2]int64 // width, then dynamic/owned
		var orders [2]string
		for index, width := range []int{1, 128} {
			f := metadata209BuildMode(t, width, true, "@meta.Marker package p;import q.Cell;")
			retained = append(retained, f)
			owned := f.ctx
			dynamic := owned
			dynamic.syntaxInvocation = nil
			dynamic.callableSubclasses = nil
			// Both arms are warm before any counter window. All parser Close calls
			// occurred in construction, and every owned tree stays retained below.
			want := nonDeclaration209Agree(t, f, owned)
			if nonDeclaration209Agree(t, f, dynamic) != want || want != definitionParameterJavaSignatureType(f.method, 0) || genericMethodHasErasedEntry(f.method) {
				t.Fatal("anonymous rejection did not preserve the current source signature")
			}
			order := [4]int{0, 1, 1, 0}
			orders[index] = "DOOD"
			if (sample+index)%2 == 1 {
				order = [4]int{1, 0, 0, 1}
				orders[index] = "ODDO"
			}
			for _, arm := range order {
				ctx := dynamic
				if arm == 1 {
					ctx = owned
				}
				start := runtime.NumCgoCall()
				for query := 0; query < queries/2; query++ {
					if got := invocationPhysicalParameterJavaType(f.resolution(), 0, ctx); got != want {
						t.Fatalf("query changed physical signature %q want%q", got, want)
					}
				}
				counts[index][arm] += runtime.NumCgoCall() - start
			}
		}
		smallDynamic, smallOwned, largeDynamic, largeOwned := counts[0][0], counts[0][1], counts[1][0], counts[1][1]
		if smallDynamic <= 0 || largeDynamic <= 0 {
			t.Fatal("native witness absent")
		}
		ceiling := 9 * largeDynamic / 10
		t.Logf("METADATA209_NATIVE sample=%d small_width=1 large_width=128 queries=%d small_order=%s large_order=%s small_dynamic_cgo=%d small_owned_cgo=%d large_dynamic_cgo=%d large_owned_cgo=%d ceiling=%d small_native_ratio=%.6f large_native_ratio=%.6f", sample, queries, orders[0], orders[1], smallDynamic, smallOwned, largeDynamic, largeOwned, ceiling, float64(smallOwned)/float64(smallDynamic), float64(largeOwned)/float64(largeDynamic))
		if largeOwned > ceiling {
			t.Errorf("METADATA209_REPEATED_SYNTAX_NATIVE: sample%d owned%d > ceiling%d dynamic%d", sample, largeOwned, ceiling, largeDynamic)
		}
	}
	runtime.KeepAlive(retained)
}

func TestMetadata209MutationAndForeignRoots(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(fmt.Sprint("warmed-holder-class-move-active-", active), func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, "@meta.Marker package p;import q.Cell;")
			sourceOwnershipIndex(f.ctx)
			if active {
				activeResolutionFiles = newResolutionFileIndex()
			}
			left, right := f.files[4].Symbols, f.files[3].Symbols
			oldLeft, oldRight := left.TopLevelClasses, right.TopLevelClasses
			defer func() { left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight }()
			left.TopLevelClasses = nil
			right.TopLevelClasses = append(append([]*symbol.ClassScope(nil), oldRight...), oldLeft[0])
			metadata209Compare(t, f)
			left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight
			metadata209Compare(t, f)
			runtime.KeepAlive(f)
		})
	}
	t.Run("nondeclaring-same-spelling-foreign-root", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		sourceOwnershipIndex(f.ctx)
		twin, tree := resolveImport205Parse(t, "TwinNoise.java", string(f.files[4].Source))
		f.trees = append(f.trees, tree)
		owner := f.files[4].Symbols.BaseClass
		root := owner.Class.DeclarationNode
		owner.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
		defer func() { owner.Class.DeclarationNode = root }()
		metadata209Compare(t, f)
		owner.Class.DeclarationNode = root
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("foreign-registered-class", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		extra, tree := resolveImport205Parse(t, "foreign/Added.java", "package foreign;class Added{q.Cell<?> demanded;}")
		f.trees = append(f.trees, tree)
		symbol.AddSymbolsToPackage(extra.Symbols)
		if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
			t.Fatal("foreign file admitted")
		}
		metadata209Compare(t, f)
		delete(symbol.GlobalScope.Packages, "foreign")
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("duplicate-file-membership", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		pkg := symbol.GlobalScope.Packages["noise"]
		pkg.Files["duplicate"] = f.files[4].Symbols
		defer delete(pkg.Files, "duplicate")
		if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
			t.Fatal("duplicate file admitted")
		}
		metadata209Compare(t, f)
		delete(pkg.Files, "duplicate")
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("edited-metadata-owned-tree", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;import q.Cell;")
		meta := f.files[len(f.files)-1]
		at := uint32(bytes.Index(meta.Source, []byte("q.Cell")))
		tree := f.trees[len(f.trees)-1]
		tree.Edit(sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}})
		if !meta.Symbols.ParsedRoot().HasChanges() {
			t.Fatal("real metadata Tree.Edit witness absent")
		}
		metadata209Compare(t, f)
		runtime.KeepAlive(f)
	})
	t.Run("fresh-reparse-keeps-old-retired", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@meta.Marker package p;import q.Cell;")
		old := f.ctx.syntaxInvocation
		old.close()
		replacement, tree := metadata209ParseUnit(t, "p/package-info.java", "@meta.Marker package p;import p.Cell;")
		f.trees = append(f.trees, tree)
		f.files[len(f.files)-1] = replacement
		fresh := newSyntaxInvocation(f.session, f.files)
		if fresh == nil {
			t.Fatal("fresh metadata reparse setup absent")
		}
		defer fresh.close()
		f.ctx.syntaxInvocation = fresh
		f.ctx.genericFamilies = &genericFamilyAnalysis{}
		f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
		invocationPhysicalParameterJavaType(f.resolution(), 0, f.ctx)
		metadata209Compare(t, f)
		if old.active || old.demandSyntax != nil {
			t.Fatal("old invocation gained new lifetime")
		}
		runtime.KeepAlive(f)
	})
}

func TestMetadata209SymbolOnlyExtraClass(t *testing.T) {
	for _, mode := range []string{"base-only", "top-level-only", "both"} {
		t.Run(mode, func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, "@Marker package p;import meta.Marker;import q.Cell;")
			want := metadata209Compare(t, f)
			metadata := f.files[len(f.files)-1].Symbols
			if metadata.BaseClass != nil || len(metadata.TopLevelClasses) != 0 {
				t.Fatal("metadata setup has source classes")
			}
			originalBase, originalTops := metadata.BaseClass, metadata.TopLevelClasses
			defer func() { metadata.BaseClass, metadata.TopLevelClasses = originalBase, originalTops }()
			extra := f.files[4].Symbols.BaseClass
			if mode == "base-only" || mode == "both" {
				metadata.BaseClass = extra
			}
			if mode == "top-level-only" || mode == "both" {
				metadata.TopLevelClasses = []*symbol.ClassScope{extra}
			}
			if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) {
				t.Fatal("legacy guard changed")
			}
			if metadata209Compare(t, f) != want {
				t.Fatal("unregistered metadata symbol alias affected declaring signature")
			}
			metadata.BaseClass, metadata.TopLevelClasses = originalBase, originalTops
			if metadata209Compare(t, f) != want {
				t.Fatal("metadata symbol closure restoration changed signature")
			}
			// Candidate-specific direct guards must refuse the proposed new capability
			// in each altered state, not rely on this unchanged dynamic outcome.
			runtime.KeepAlive(f)
		})
	}
}
