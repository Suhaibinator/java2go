package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/printer"
	"go/token"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

func superclassQuery4059OldOracle(name string, ctx Ctx) bool {
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			var visit func(*sitter.Node) bool
			visit = func(node *sitter.Node) bool {
				if node == nil {
					return false
				}
				if node.Type() == "identifier" || node.Type() == "type_identifier" {
					identifier := sanitizeGoIdent(node.Content(file.Source))
					if identifier == name || symbol.Uppercase(identifier) == name || symbol.Lowercase(identifier) == name {
						return true
					}
				}
				if node.Type() == "method_declaration" {
					method := synthAnonClassMethodDefinition(node, file.Source, false)
					if method != nil && (method.Name == name ||
						(strings.HasPrefix(name, method.Name+executionMethodSuffix) && collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, nil) == name) ||
						(strings.HasPrefix(name, method.Name+"Java2goExactExecution") && collisionSafeExecutionIdentifier(method.Name+"Java2goExactExecution", nil) == name)) {
						return true
					}
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					if visit(child) {
						return true
					}
				}
				return false
			}
			for _, top := range file.TopLevelClasses {
				if top != nil && top.Class != nil && visit(top.Class.DeclarationNode) {
					return true
				}
			}
		}
	}
	return false
}

func superclassQuery4059PlannerOracle(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceInterfaceVirtualEntry {
	if owner == nil || owner.Class == nil || !owner.IsInterface || len(owner.TypeParameters) != 0 || method == nil || method.DeclarationNode == nil || method.DeclarationNode.Type() != "method_declaration" || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper || len(method.TypeParameters) != 0 {
		return nil
	}
	declared := false
	for _, candidate := range owner.Methods {
		if candidate == method {
			declared = true
			break
		}
	}
	if !declared {
		return nil
	}
	signature := sourceInterfaceVirtualSignature(owner, method, ctx)
	if signature == nil {
		return nil
	}
	identity := javaClassBinaryName(owner) + "\x00" + method.OriginalName
	for index := range method.Parameters {
		identity += "\x00" + qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner)
	}
	identity += "\x01" + qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
	base := "Java2goInterfaceVirtual_" + fmt.Sprintf("%X", []byte(identity))
	name := base
	for suffix := 0; ; suffix++ {
		name = base
		if suffix > 0 {
			name += "_" + strconv.Itoa(suffix)
		}
		// Scan the source graph as well as Definitions: locals and anonymous
		// declarations have not necessarily been hoisted at this emission point.
		if !generatedIdentifierExists(name, nil) && !sourceMethodIdentifierExists(name, ctx) && !superclassQuery4059OldOracle(name, ctx) {
			break
		}
	}
	return &sourceInterfaceVirtualEntry{owner: owner, method: method, name: name, signature: signature}
}

// Written types are qualified in the declaration's scope, then lowered in the
// consuming file so imports and aliases belong to that file. No new erasure is
// admitted here; generic owner/method contracts are deliberately excluded.

type superclassQuery4059Fixture struct {
	files []*parsing.SourceFile
	ctx   Ctx
}

func superclassQuery4059Graph(t *testing.T, sources []string, raw bool) *superclassQuery4059Fixture {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	fixture := &superclassQuery4059Fixture{}
	for index, source := range sources {
		file := &parsing.SourceFile{Name: fmt.Sprintf("Fixture%d.java", index), Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		if raw {
			scope := &symbol.ClassScope{Class: &symbol.Definition{Name: fmt.Sprintf("Raw%d", index), OriginalName: fmt.Sprintf("Raw%d", index), DeclarationNode: file.Ast}}
			file.Symbols = &symbol.FileScope{Source: file.Source, Package: "query", Imports: map[string]string{}, BaseClass: scope, TopLevelClasses: []*symbol.ClassScope{scope}}
		} else {
			file.ParseSymbols()
		}
		symbol.AddSymbolsToPackage(file.Symbols)
		fixture.files = append(fixture.files, file)
	}
	fixture.ctx = Ctx{currentFile: fixture.files[0].Symbols, currentClass: fixture.files[0].Symbols.BaseClass, usedImports: map[string]bool{}, importAliases: map[string]string{}}
	return fixture
}

func superclassQuery4059Nodes(root *sitter.Node, kind string) []*sitter.Node {
	var result []*sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == kind {
			result = append(result, node)
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			visit(node.NamedChild(index))
		}
	}
	visit(root)
	return result
}

func superclassQuery4059State(fixture *superclassQuery4059Fixture) string {
	var state []string
	for packageName, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			state = append(state, packageName+"=nil")
			continue
		}
		for fileName, file := range pkg.Files {
			if file == nil {
				state = append(state, packageName+"/"+fileName+"=nil")
				continue
			}
			imports, _ := json.Marshal(file.Imports)
			state = append(state, fmt.Sprintf("%s/%s:%p:%s:%s", packageName, fileName, file, file.Source, imports))
			var visit func(*symbol.ClassScope)
			visit = func(scope *symbol.ClassScope) {
				if scope == nil || scope.Class == nil {
					return
				}
				state = append(state, fmt.Sprintf("class:%p:%s:%s:%p", scope, scope.Class.Name, scope.Class.OriginalName, scope.Class.DeclarationNode))
				for _, method := range scope.Methods {
					if method != nil {
						state = append(state, fmt.Sprintf("method:%p:%s:%s:%s:%s", method, method.Name, method.OriginalName, method.OriginalType, method.HelperName))
					}
				}
				for _, field := range scope.Fields {
					if field != nil {
						state = append(state, fmt.Sprintf("field:%p:%s:%s", field, field.Name, field.OriginalName))
					}
				}
				for _, child := range scope.Subclasses {
					visit(child)
				}
			}
			for _, top := range file.TopLevelClasses {
				visit(top)
			}
		}
	}
	for _, file := range fixture.files {
		state = append(state, fmt.Sprintf("tree:%d:%s", file.Ast.ID(), file.Ast.String()))
	}
	imports, _ := json.Marshal(struct {
		Used    map[string]bool
		Aliases map[string]string
	}{fixture.ctx.usedImports, fixture.ctx.importAliases})
	state = append(state, string(imports))
	sort.Strings(state)
	return strings.Join(state, "\n")
}

type superclassQuery4059Observation struct {
	Found                  bool
	PanicType, Panic, Logs string
}

func superclassQuery4059Observe(query string, ctx Ctx, oracle bool) (result superclassQuery4059Observation) {
	logger := log.StandardLogger()
	previous := logger.Out
	var output bytes.Buffer
	logger.SetOutput(&output)
	defer func() {
		if caught := recover(); caught != nil {
			result.PanicType = fmt.Sprintf("%T", caught)
			result.Panic = fmt.Sprint(caught)
		}
		result.Logs = output.String()
		logger.SetOutput(previous)
	}()
	if oracle {
		result.Found = superclassQuery4059OldOracle(query, ctx)
	} else {
		result.Found = superclassEmbeddingSourceIdentifierExists(query, ctx)
	}
	return result
}
func superclassQuery4059Compare(t *testing.T, fixture *superclassQuery4059Fixture, query string, want *bool) superclassQuery4059Observation {
	t.Helper()
	before := superclassQuery4059State(fixture)
	actual := superclassQuery4059Observe(query, fixture.ctx, false)
	middle := superclassQuery4059State(fixture)
	oracle := superclassQuery4059Observe(query, fixture.ctx, true)
	after := superclassQuery4059State(fixture)
	if !reflect.DeepEqual(actual, oracle) {
		t.Fatalf("query %q actual=%+v oracle=%+v", query, actual, oracle)
	}
	if before != middle || before != after {
		t.Fatalf("query %q changed source/declaration/import/binder state", query)
	}
	if actual.Logs != "" {
		t.Fatalf("query %q emitted logs %q", query, actual.Logs)
	}
	if want != nil && (actual.PanicType != "" || actual.Found != *want) {
		t.Fatalf("query %q got=%+v want=%v", query, actual, *want)
	}
	runtime.KeepAlive(fixture.files)
	return actual
}
func superclassQuery4059Bool(value bool) *bool { return &value }

func TestSuperclassQuery4059SemanticOracle(t *testing.T) {
	cases := []struct{ name, source, emitted string }{
		{"public-keyword", `class Owner {public void type(int marker){int map=marker;}}`, "Type"},
		{"private-keyword", `class Owner {private void type(int marker){int map=marker;}}`, "type_"},
		{"protected-keyword", `class Owner {protected void If(int marker){}}`, "if_"},
		{"package-keyword", `class Owner {void map(int marker){}}`, "map_"},
		{"static-native", `class Owner {public static native void range(int marker);}`, "Range"},
		{"abstract-generic", `abstract class Owner {abstract <T> T func(T marker);}`, "func_"},
		{"annotations", `class Owner {@Deprecated public void var(@Deprecated int marker){}}`, "Var"},
		{"implicit-interface", `interface Owner {void Small(int marker);}`, "small"},
		{"explicit-interface", `interface Owner {public void Small(int marker);}`, "Small"},
		{"default-interface", `interface Owner {default void Go(int marker){}}`, "go_"},
		{"public-dollar", `class Owner {public void small$(int marker){}}`, "Java2go$Public/small$"},
		{"private-dollar", `class Owner {void Small$(int marker){}}`, "java2go$Private/Small$"},
		{"unicode-public", `class Owner {public void éclair(int marker){}}`, "Éclair"},
		{"unicode-private", `class Owner {void Éclair(int marker){}}`, "éclair"},
		{"escape-namespace", `class Owner {void Java2goIdentifier_ab(int marker){}}`, "java2goIdentifier_ab"},
		{"unicode-source", `class Owner {public void \u0074ype(int marker){}}`, "U0074ype"},
		{"zero-formals", `class Owner {public void none(){int marker=1;}}`, "None"},
		{"array-return-varargs", `class Owner {public int[] type(@Deprecated int... marker){return null;}}`, "Type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := superclassQuery4059Graph(t, []string{tc.source}, false)
			nodes := superclassQuery4059Nodes(f.files[0].Ast, "method_declaration")
			if len(nodes) != 1 {
				t.Fatalf("methods=%d", len(nodes))
			}
			if tc.name == "unicode-source" {
				if !nodes[0].HasError() {
					t.Fatal("retained Unicode parser ERROR fallback shield disappeared")
				}
				superclassQuery4059Compare(t, f, "TypeJava2goExecution", superclassQuery4059Bool(false))
			}
			method := synthAnonClassMethodDefinition(nodes[0], f.files[0].Symbols.Source, false)
			if method == nil || method.Name != tc.emitted {
				t.Fatalf("actual full header name=%+v want=%q", method, tc.emitted)
			}
			for _, query := range []string{tc.emitted + executionMethodSuffix, tc.emitted + "Java2goExactExecution"} {
				superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(true))
			}
			for _, query := range []string{"UnrelatedAllocatedEntry", tc.emitted + executionMethodSuffix + "NearMiss", tc.emitted + "Java2goExactExecutionNearMiss"} {
				superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(false))
			}
			original := nodes[0].ChildByFieldName("name").Content(f.files[0].Symbols.Source)
			other := sanitizeGoIdent(symbol.HandleExportStatus(true, original))
			if other == tc.emitted {
				other = sanitizeGoIdent(symbol.HandleExportStatus(false, original))
			}
			if other != tc.emitted {
				superclassQuery4059Compare(t, f, other+executionMethodSuffix, superclassQuery4059Bool(false))
			}
		})
	}
}

func TestSuperclassQuery4059RawCollisionAndDynamicGraphs(t *testing.T) {
	t.Run("raw-and-nested", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {int field; FieldType typed; void act(int formal){int local=formal;mark:while(false){break mark;}class Local {void inside(int n){int inner=n;}} Object anon=new Object(){public void nested(int n){int body=n;}};}}`}, false)
		for _, q := range []string{"field", "FieldType", "formal", "local", "mark", "Local", "inner", "body", "InsideJava2goExecution", "NestedJava2goExecution"} {
			// Local package method is lowercased; the anonymous method is explicitly public.
			if q == "InsideJava2goExecution" {
				q = "insideJava2goExecution"
			}
			superclassQuery4059Compare(t, f, q, superclassQuery4059Bool(true))
		}
	})
	t.Run("real-collision-suffixes", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {void act(int marker){}} class Collision {int actJava2goExecution;int actJava2goExecution1;int actJava2goExactExecution;int actJava2goExactExecution1;}`}, false)
		for _, base := range []string{"actJava2goExecution", "actJava2goExactExecution"} {
			if got := collisionSafeExecutionIdentifier(base, nil); got != base+"2" {
				t.Fatalf("real allocator returned %q", got)
			}
			superclassQuery4059Compare(t, f, base+"2", superclassQuery4059Bool(true))
			superclassQuery4059Compare(t, f, base+"3", superclassQuery4059Bool(false))
		}
	})
	t.Run("dynamic-file-local-import-binder-root", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner<T>{void act(int marker){}}`}, false)
		query := "laterJava2goExecution"
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(false))
		later := superclassQuery4059Graph(t, []string{`class Added {void host(int marker){class Local {void later(int n){}}}}`}, false)
		addedGraph := symbol.GlobalScope
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{"": {Files: map[string]*symbol.FileScope{"Owner": f.files[0].Symbols}}}}
		for _, pkg := range addedGraph.Packages {
			for name, file := range pkg.Files {
				symbol.GlobalScope.Packages[""].Files[name] = file
			}
		}
		f.files = append(f.files, later.files...)
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(true))
		f.files[0].Symbols.Imports["Added"] = "other.package"
		f.ctx.importAliases["other.package"] = "Other"
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(true))
		delete(symbol.GlobalScope.Packages[""].Files, "added")
		if len(symbol.GlobalScope.Packages[""].Files) != 1 {
			t.Fatal("dynamic graph removal did not remove its actual registered key")
		}
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(false))
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(false))
		symbol.AddSymbolsToPackage(later.files[0].Symbols)
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(true))
	})
}

func TestSuperclassQuery4059MalformedAndOwnershipOracle(t *testing.T) {
	for _, source := range []string{`class Broken {void act(int marker){int later=marker;`, `class Broken {void (int marker){}}`, `class Broken {void act( ){}`, `class Broken {act(int marker){}}`} {
		t.Run(fmt.Sprintf("syntax-%d", len(source)), func(t *testing.T) {
			f := superclassQuery4059Graph(t, []string{source}, true)
			superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", nil)
		})
	}
	t.Run("truncated-after-header-still-panics", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {void act(int marker){int later=marker;}}`}, false)
		method := superclassQuery4059Nodes(f.files[0].Ast, "method_declaration")[0]
		end := method.ChildByFieldName("name").EndByte()
		f.files[0].Symbols.Source = f.files[0].Symbols.Source[:end:end]
		observation := superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", nil)
		if observation.PanicType == "" {
			t.Fatal("old negative query no longer exposes parameter/body Content panic")
		}
	})
	t.Run("nil-source", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {void act(int marker){}}`}, false)
		f.files[0].Symbols.Source = nil
		if superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", nil).PanicType == "" {
			t.Fatal("expected old Content panic")
		}
	})
	t.Run("nil-shields", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {}`}, false)
		symbol.GlobalScope.Packages["nil"] = nil
		symbol.GlobalScope.Packages[""].Files["nil"] = nil
		file := f.files[0].Symbols
		file.TopLevelClasses = append(file.TopLevelClasses, nil, &symbol.ClassScope{}, &symbol.ClassScope{Class: &symbol.Definition{}})
		superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", superclassQuery4059Bool(false))
	})
	t.Run("borrowed-live-tree-and-native-equality", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {void act(int marker){int later=marker;}}`, `class Other {void borrowed(int marker){}}`}, false)
		method := superclassQuery4059Nodes(f.files[1].Ast, "method_declaration")[0]
		first := method.ChildByFieldName("name")
		id := first.ID()
		f.files[0].Symbols.TopLevelClasses[0].Class.DeclarationNode = method
		f.files[0].Symbols.Source = f.files[1].Symbols.Source
		superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", superclassQuery4059Bool(false))
		second := method.ChildByFieldName("name")
		if first != second || first.ID() != id || !first.Equal(second) {
			t.Fatal("native tree wrapper identity changed")
		}
	})
	t.Run("edited-node-falls-back", func(t *testing.T) {
		f := superclassQuery4059Graph(t, []string{`class Owner {void act(int marker){int later=marker;}}`}, false)
		method := superclassQuery4059Nodes(f.files[0].Ast, "method_declaration")[0]
		method.Edit(sitter.EditInput{StartIndex: method.StartByte(), OldEndIndex: method.EndByte(), NewEndIndex: method.EndByte() + 3})
		superclassQuery4059Compare(t, f, "UnrelatedAllocatedEntry", nil)
	})
}

func superclassQuery4059CostFixture(t *testing.T, methods, statements, formals int, nested bool) *superclassQuery4059Fixture {
	t.Helper()
	var source strings.Builder
	source.WriteString("interface Probe {void invoke(int marker);} class Work {")
	for method := 0; method < methods; method++ {
		fmt.Fprintf(&source, "void noise%d(", method)
		if formals > 0 {
			source.WriteString("int marker")
		}
		source.WriteString("){int value=0;")
		for statement := 0; statement < statements; statement++ {
			source.WriteString("value=value+1;")
		}
		if nested {
			fmt.Fprintf(&source, "class Local%d {void nested%d(int formal){int capture=formal;}} Object anonymous=new Object(){public void accept(int formal){int capture=formal;}};", method, method)
		}
		source.WriteString("}")
	}
	source.WriteString("}")
	return superclassQuery4059Graph(t, []string{source.String()}, false)
}

type superclassQuery4059WorkRow struct {
	Label                        string
	Methods, Statements, Formals int
	Nested                       bool
	ActualAllocs, OracleAllocs   float64
	ActualCgo, OracleCgo         int64
	Output                       string
}

func superclassQuery4059Measure(t *testing.T, fn func() string) (float64, int64, string) {
	t.Helper()
	want := fn()
	for warm := 0; warm < 2; warm++ {
		if fn() != want {
			t.Fatal("unstable warm output")
		}
	}
	const repetitions = 3
	before := runtime.NumCgoCall()
	for query := 0; query < repetitions; query++ {
		if fn() != want {
			t.Fatal("unstable native-work output")
		}
	}
	delta := runtime.NumCgoCall() - before
	allocations := testing.AllocsPerRun(repetitions, func() {
		if fn() != want {
			panic("unstable allocation output")
		}
	})
	if delta%repetitions != 0 {
		t.Fatalf("unexpected native finalizer/concurrent noise: delta=%d", delta)
	}
	return allocations, delta / repetitions, want
}
func superclassQuery4059Work(t *testing.T, check func(*testing.T, superclassQuery4059WorkRow)) {
	var owners []*superclassQuery4059Fixture
	defer func() { runtime.KeepAlive(owners) }()
	cases := []struct {
		methods, statements, formals int
		nested                       bool
	}{{1, 4, 1, false}, {16, 4, 1, false}, {64, 4, 1, false}, {1, 24, 1, false}, {16, 24, 1, false}, {64, 24, 1, false}, {64, 24, 0, false}, {16, 4, 1, true}}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("m%d-b%d-f%d-n%v", tc.methods, tc.statements, tc.formals, tc.nested), func(t *testing.T) {
			f := superclassQuery4059CostFixture(t, tc.methods, tc.statements, tc.formals, tc.nested)
			owners = append(owners, f)
			owner := f.files[0].Symbols.FindClassScope("Probe")
			if owner == nil || len(owner.Methods) != 1 {
				t.Fatal("missing real source-interface planner fixture")
			}
			method := owner.Methods[0]
			// Warm all native wrappers outside both measured queries; retain owning trees.
			for _, file := range f.files {
				_ = superclassQuery4059Nodes(file.Ast, "identifier")
			}
			for _, label := range []string{"scanner", "allocator"} {
				query := func(oracle bool) func() string {
					return func() string {
						if label == "scanner" {
							found := false
							if oracle {
								found = superclassQuery4059OldOracle("UnrelatedAllocatedEntry", f.ctx)
							} else {
								found = superclassEmbeddingSourceIdentifierExists("UnrelatedAllocatedEntry", f.ctx)
							}
							if found {
								panic("unrelated name became reserved")
							}
							return "absent"
						}
						var plan *sourceInterfaceVirtualEntry
						if oracle {
							plan = superclassQuery4059PlannerOracle(owner, method, f.ctx)
						} else {
							plan = planSourceInterfaceVirtualEntry(owner, method, f.ctx)
						}
						if plan == nil {
							panic("actual allocator rejected valid source interface")
						}
						return plan.name
					}
				}
				actualA, actualC, actualOutput := superclassQuery4059Measure(t, query(false))
				oracleA, oracleC, oracleOutput := superclassQuery4059Measure(t, query(true))
				if actualOutput != oracleOutput {
					t.Fatal("actual planner output differs from full old-scanner planner")
				}
				row := superclassQuery4059WorkRow{label, tc.methods, tc.statements, tc.formals, tc.nested, actualA, oracleA, actualC, oracleC, actualOutput}
				encoded, _ := json.Marshal(row)
				t.Logf("SUPERCLASS_QUERY_WORK %s", encoded)
				if check != nil {
					check(t, row)
				}
			}
			runtime.KeepAlive(f.files)
		})
	}
}
func TestSuperclassQuery4059WorkBaseline(t *testing.T) {
	if superclass4059IsolatedEntry(t) {
		return
	}
	superclassQuery4059Work(t, nil)
}

func TestSuperclassQuery4059AllocatorASTImports(t *testing.T) {
	f := superclassQuery4059Graph(t, []string{`interface Probe {void invoke(int marker);} class Work {void noise(int marker){int value=marker;}}`}, false)
	owner := f.files[0].Symbols.FindClassScope("Probe")
	method := owner.Methods[0]
	actualCtx, oracleCtx := f.ctx.Clone(), f.ctx.Clone()
	actualCtx.usedImports, actualCtx.importAliases = map[string]bool{}, map[string]string{}
	oracleCtx.usedImports, oracleCtx.importAliases = map[string]bool{}, map[string]string{}
	actual := planSourceInterfaceVirtualEntry(owner, method, actualCtx)
	oracle := superclassQuery4059PlannerOracle(owner, method, oracleCtx)
	if actual == nil || oracle == nil || actual.name != oracle.name {
		t.Fatal("planner output changed")
	}
	render := func(value any) string {
		var output bytes.Buffer
		if err := printer.Fprint(&output, token.NewFileSet(), value); err != nil {
			t.Fatal(err)
		}
		return output.String()
	}
	if render(sourceInterfaceVirtualOptionalType(actual)) != render(sourceInterfaceVirtualOptionalType(oracle)) {
		t.Fatal("optional interface emitted AST differs")
	}
	if got := render(sourceInterfaceVirtualOptionalType(actual)); !strings.Contains(got, "int32") {
		t.Fatalf("lost formal in real emitted AST: %s", got)
	}
	if !reflect.DeepEqual(actualCtx.usedImports, oracleCtx.usedImports) || !reflect.DeepEqual(actualCtx.importAliases, oracleCtx.importAliases) {
		t.Fatalf("actual allocator imports differ from old oracle: %v/%v versus %v/%v", actualCtx.usedImports, actualCtx.importAliases, oracleCtx.usedImports, oracleCtx.importAliases)
	}
	if !reflect.DeepEqual(actualCtx.usedImports, map[string]bool{stdjavaImportPath: true}) || !reflect.DeepEqual(actualCtx.importAliases, map[string]string{stdjavaImportPath: "stdjava"}) {
		t.Fatalf("existing runtime execution-context import changed: %v %v", actualCtx.usedImports, actualCtx.importAliases)
	}
	runtime.KeepAlive(f.files)
}

// Budgets were frozen from retained actual measurements before this assertion or any production edit.
func TestSuperclassQuery4059FrozenWorkBudget(t *testing.T) {
	if superclass4059IsolatedEntry(t) {
		return
	}
	budgets := map[string]struct {
		allocs float64
		cgo    int64
	}{
		"scanner/m1/b4/f1/nfalse":     {268, 411},
		"allocator/m1/b4/f1/nfalse":   {1204, 1757},
		"scanner/m16/b4/f1/nfalse":    {3316, 5043},
		"allocator/m16/b4/f1/nfalse":  {11766, 18160},
		"scanner/m64/b4/f1/nfalse":    {13070, 19865},
		"allocator/m64/b4/f1/nfalse":  {45550, 72009},
		"scanner/m1/b24/f1/nfalse":    {908, 1339},
		"allocator/m1/b24/f1/nfalse":  {3436, 5105},
		"scanner/m16/b24/f1/nfalse":   {13556, 19891},
		"allocator/m16/b24/f1/nfalse": {47479, 71728},
		"scanner/m64/b24/f1/nfalse":   {54030, 79257},
		"allocator/m64/b24/f1/nfalse": {188400, 286281},
		"scanner/m16/b4/f1/ntrue":     {7758, 12096},
		"allocator/m16/b4/f1/ntrue":   {24582, 39558},
	}
	superclassQuery4059Work(t, func(t *testing.T, row superclassQuery4059WorkRow) {
		key := fmt.Sprintf("%s/m%d/b%d/f%d/n%t", row.Label, row.Methods, row.Statements, row.Formals, row.Nested)
		budget, ok := budgets[key]
		if !ok {
			if row.Formals != 0 {
				t.Fatalf("missing frozen budget %s", key)
			}
			return
		}
		if row.OracleAllocs <= budget.allocs || row.OracleCgo <= budget.cgo {
			t.Errorf("OLD_ORACLE_WORK_SHIELD: %s old alloc=%.0f cgo=%d unexpectedly meets frozen budgets %.0f/%d", key, row.OracleAllocs, row.OracleCgo, budget.allocs, budget.cgo)
		}
		if row.ActualAllocs > budget.allocs {
			t.Errorf("ACTUAL_ALLOCATION_WORK_RED: %s actual=%.0f frozen_max=%.0f", key, row.ActualAllocs, budget.allocs)
		}
		if row.ActualCgo > budget.cgo {
			t.Errorf("ACTUAL_NATIVE_WORK_RED: %s actual=%d frozen_max=%d", key, row.ActualCgo, budget.cgo)
		}
	})
}

// Binder and namespace mutations remain visible to every fresh source query.
func TestSuperclassQuery4059BinderAndNamespaceMutation(t *testing.T) {
	f := superclassQuery4059Graph(t, []string{`class Owner<T> {<U> U act(U marker){return marker;}}`}, false)
	owner := f.files[0].Symbols.BaseClass
	if len(owner.TypeParameters) != 1 || len(owner.Methods) != 1 || len(owner.Methods[0].TypeParameters) != 1 {
		t.Fatal("missing real class/method binder declarations")
	}
	classBinder, methodBinder := owner.TypeParameters[0].Declaration, owner.Methods[0].TypeParameters[0].Declaration
	if classBinder == nil || methodBinder == nil || classBinder == methodBinder {
		t.Fatal("class/method binder identities missing")
	}
	query := "actJava2goExecution"
	observe := func(want bool) {
		beforeClass, beforeMethod := owner.TypeParameters[0].Declaration, owner.Methods[0].TypeParameters[0].Declaration
		classNames, methodNames := *beforeClass, *beforeMethod
		superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(want))
		if owner.TypeParameters[0].Declaration != beforeClass || owner.Methods[0].TypeParameters[0].Declaration != beforeMethod || *beforeClass != classNames || *beforeMethod != methodNames {
			t.Fatal("query mutated binder identity/names")
		}
	}
	observe(true)
	classBinder.GoName = "RenamedClassBinder"
	methodBinder.GoName = "RenamedMethodBinder"
	observe(true)
	owner.TypeParameters[0].Declaration = &symbol.TypeParamDeclaration{SourceName: "T", GoName: "FreshClassBinder"}
	owner.Methods[0].TypeParameters[0].Declaration = &symbol.TypeParamDeclaration{SourceName: "U", GoName: "FreshMethodBinder"}
	observe(true)
	f.ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("CallerBinder", nil)}
	f.ctx.rawGenericParameterTypes = map[string]string{"marker": "CallerBinder"}
	observe(true)
	// Allocate a real collision in the generated declaration namespace, then remove it.
	owner.Fields = append(owner.Fields, &symbol.Definition{Name: query, OriginalName: query, OriginalType: "int"})
	observe(false)
	superclassQuery4059Compare(t, f, query+"1", superclassQuery4059Bool(true))
	owner.Fields = owner.Fields[:len(owner.Fields)-1]
	observe(true)
	// Replace the declaration subtree/source without retaining a negative source result.
	newer := superclassQuery4059Graph(t, []string{`class Other<V> {void later(int marker){class Local {void local(int n){}}}}`}, false)
	replacement := newer.files[0].Symbols
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{"": {Files: map[string]*symbol.FileScope{"replacement": replacement}}}}
	f.files = append(f.files, newer.files...)
	superclassQuery4059Compare(t, f, query, superclassQuery4059Bool(false))
	superclassQuery4059Compare(t, f, "localJava2goExecution", superclassQuery4059Bool(true))
	runtime.KeepAlive(f.files)
}
