package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
	"sort"

	"github.com/NickyBoy89/java2go/parsing"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// Exact original allocator: the oracle retains full proof and namespace scans.
func comparableFilterExhaustiveOracle(base string, scope *symbol.ClassScope, ctx Ctx) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}
		if generatedIdentifierExists(candidate, scope) {
			continue
		}
		occupied := false
		for _, owner := range allSourceClassScopes() {
			for _, method := range owner.Methods {
				if method != nil && (method.HelperName == candidate || !method.Constructor && executionImplementationName(method, owner, ctx) == candidate) {
					occupied = true
				}
			}
		}
		if !occupied {
			return candidate
		}
	}
}

func comparableFilterNegativeFixture(t *testing.T, methods int) (*symbol.ClassScope, Ctx) {
	t.Helper()
	var source strings.Builder
	source.WriteString("class CostOwner {")
	for i := 0; i < methods; i++ {
		fmt.Fprintf(&source, "void noise%d() {}", i)
	}
	source.WriteString("}")
	h := setupParseHelper(t, source.String())
	owner := h.File.Symbols.FindClassScope("CostOwner")
	if owner == nil {
		t.Fatal("missing cost owner")
	}
	return owner, h.Ctx
}

func TestComparableNameFilterAllocatorWorkAndAllocationBaseline(t *testing.T) {
	for _, methods := range []int{1, 16, 64} {
		t.Run(fmt.Sprint(methods), func(t *testing.T) {
			owner, ctx := comparableFilterNegativeFixture(t, methods)
			ctx.genericFamilies = &genericFamilyAnalysis{}
			const base = "UnrelatedAllocatedEntry"
			if got := sourceNativeComparableAllocatedIdentifier(base, owner, ctx); got != base {
				t.Fatalf("allocated %q, want %q", got, base)
			}
			warmed := ctx.genericFamilies.ready
			measure := func(oracle bool) float64 {
				return testing.AllocsPerRun(3, func() {
					fresh := ctx.Clone()
					fresh.genericFamilies = &genericFamilyAnalysis{}
					got := ""
					if oracle {
						got = comparableFilterExhaustiveOracle(base, owner, fresh)
					} else {
						got = sourceNativeComparableAllocatedIdentifier(base, owner, fresh)
					}
					if got != base {
						t.Fatalf("allocation measurement changed identifier: %q", got)
					}
				})
			}
			actual, oracle := measure(false), measure(true)
			t.Logf("METHODS=%d ACTUAL_ALLOCS=%.0f EXHAUSTIVE_ALLOCS=%.0f CACHE_READY=%v", methods, actual, oracle, warmed)
			if actual > 64 {
				t.Errorf("NEGATIVE_ALLOCATION_BUDGET: %.0f > frozen64 allocations/query", actual)
			}
			if warmed {
				t.Error("IMPOSSIBLE_NAME_WARMED_GENERIC_FAMILY_PROOF: unrelated execution spellings must not plan ancestry")
			}
		})
	}
}

type comparableFilterBranchCase struct {
	name, source, owner, method, suffix string
	specialized, generic, covariant     bool
}

func comparableFilterBranchCases() []comparableFilterBranchCase {
	return []comparableFilterBranchCase{
		{name: "ordinary", source: `class Plain {/*COLLISIONS*/ public int act(){return 1;}}`, owner: "Plain", method: "act", suffix: "Java2goExecution"},
		{name: "specialized", source: `interface Numbered {int number();} class First implements Numbered{public int number(){return 1;}} class Base<T extends Numbered>{T exchange(T x){return x;}} class Child extends Base<First>{/*COLLISIONS*/ First exchange(First x){return x;}}`, owner: "Child", method: "exchange", suffix: "Java2goExactExecution", specialized: true},
		{name: "generic-family", source: `interface Slot<T>{T read();void write(T value);} abstract class Adapter<T> implements Slot<T>{public abstract T read();public abstract void write(T value);} abstract class Middle<X> extends Adapter<X>{} class Specific extends Middle<String>{/*COLLISIONS*/ public String read(){return "ok";} public void write(String value){}} class Use{Adapter<?> demand;}`, owner: "Specific", method: "read", suffix: "Java2goExactExecution", specialized: true, generic: true},
		{name: "covariant", source: `class Item{} interface Source {Object read();} class Child implements Source {/*COLLISIONS*/ public Item read(){return null;}}`, owner: "Child", method: "read", suffix: "Java2goExactExecution", specialized: true, covariant: true},
		{name: "entry", source: `class Entry implements java.util.Map.Entry<String,String> {/*COLLISIONS*/ public String getKey(){return "key";} public String getValue(){return "value";} public String setValue(String value){return value;}}`, owner: "Entry", method: "getKey", suffix: "Java2goEntryBodyExecution"},
		{name: "iteration", source: `class Cursor implements java.util.Iterator<String>{/*COLLISIONS*/ public boolean hasNext(){return false;} public String next(){return "next";}}`, owner: "Cursor", method: "next", suffix: "Java2goIterationBodyExecution"},
	}
}

func comparableFilterFreshContext(ctx Ctx) Ctx {
	ctx = ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{}
	return ctx
}

func TestComparableNameFilterActualBranchesAndCollisions(t *testing.T) {
	for _, tc := range comparableFilterBranchCases() {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.source)
			owner := overrideBridgeTestScope(t, h, tc.owner)
			method := overrideBridgeTestMethod(t, owner, tc.method)
			base := method.Name + tc.suffix
			if got := executionImplementationName(method, owner, comparableFilterFreshContext(h.Ctx)); got != base {
				t.Fatalf("wrong actual naming branch: %q, want %q", got, base)
			}
			if _, ok := directOwnerSpecializedOverrideBridgeForMethod(owner, method, comparableFilterFreshContext(h.Ctx)); ok != tc.specialized {
				t.Fatalf("specialized branch=%v,want%v", ok, tc.specialized)
			}
			if tc.generic {
				if _, ok := genericFamilySpecializedMethod(owner, method, comparableFilterFreshContext(h.Ctx)); !ok {
					t.Fatal("generic-family path not exercised")
				}
			}
			if tc.covariant {
				if _, ok := specializedAncestorCovariantBridge(owner, method, comparableFilterFreshContext(h.Ctx)); !ok {
					t.Fatal("covariant path not exercised")
				}
			}
			// Real parsed fields force the hidden implementation to suffix2.
			visibility := ""
			if base[0] >= 'A' && base[0] <= 'Z' {
				visibility = "public "
			}
			collisions := visibility + "int " + base + ";" + visibility + "int " + base + "1;"
			h = setupParseHelper(t, strings.Replace(tc.source, "/*COLLISIONS*/", collisions, 1))
			owner = overrideBridgeTestScope(t, h, tc.owner)
			method = overrideBridgeTestMethod(t, owner, tc.method)
			if got := executionImplementationName(method, owner, comparableFilterFreshContext(h.Ctx)); got != base+"2" {
				t.Fatalf("real field collision chain=%q,want%q", got, base+"2")
			}
			ctx := comparableFilterFreshContext(h.Ctx)
			got := sourceNativeComparableAllocatedIdentifier(base+"2", owner, ctx)
			want := comparableFilterExhaustiveOracle(base+"2", owner, comparableFilterFreshContext(h.Ctx))
			if got != want || got == base+"2" {
				t.Fatalf("actual execution collision=%q,oracle%q", got, want)
			}
			if !ctx.genericFamilies.ready {
				t.Fatal("possible spelling bypassed full execution planner")
			}
			// Both unsuffixed and collision-numbered queries reach full naming.
			for _, candidate := range []string{base, base + "1", base + "2", base + "20", base + "201"} {
				got := sourceNativeComparableAllocatedIdentifier(candidate, owner, comparableFilterFreshContext(h.Ctx))
				want := comparableFilterExhaustiveOracle(candidate, owner, comparableFilterFreshContext(h.Ctx))
				if got != want {
					t.Fatalf("candidate%q got%q oracle%q", candidate, got, want)
				}
			}
		})
	}
}

func TestComparableNameFilterHelpersConstructorsAndFreshGraphs(t *testing.T) {
	h := setupParseHelper(t, `class Owner{Owner(){} void act(){} void make(){class Local{void noise(){}}}}`)
	owner := overrideBridgeTestScope(t, h, "Owner")
	const base = "UnrelatedAllocatedEntry"
	verify := func(want string) {
		t.Helper()
		for _, cached := range []bool{false, true} {
			ctx := h.Ctx.Clone()
			if cached {
				ctx.genericFamilies = &genericFamilyAnalysis{}
			}
			got := sourceNativeComparableAllocatedIdentifier(base, owner, ctx)
			oracle := comparableFilterExhaustiveOracle(base, owner, comparableFilterFreshContext(h.Ctx))
			if got != want || got != oracle {
				t.Fatalf("fresh graph cached=%v got%q want%q oracle%q", cached, got, want, oracle)
			}
		}
	}
	verify(base)
	var constructor *symbol.Definition
	for _, m := range owner.Methods {
		if m != nil && m.Constructor {
			constructor = m
		}
	}
	if constructor == nil {
		t.Fatal("missing actual constructor")
	}
	constructor.HelperName = base
	verify(base + "1")
	method := overrideBridgeTestMethod(t, owner, "act")
	method.HelperName = base + "1"
	verify(base + "2")
	// Add a parsed scope to the lowering-owned nested/local graph after a query.
	local := parsing.SourceFile{Name: "Local.java", Source: []byte(`class LocalAdded{void act(){}}`)}
	if err := local.ParseAST(); err != nil {
		t.Fatal(err)
	}
	local.ParseSymbols()
	nested := local.Symbols.FindClassScope("LocalAdded")
	if nested == nil {
		t.Fatal("missing parsed local lowering scope")
	}
	nested.Enclosing = owner
	nested.Methods[0].HelperName = base + "2"
	owner.Subclasses = append(owner.Subclasses, nested)
	verify(base + "3")
	// A later resolved top-level file is visible at the next render boundary.
	later := parsing.SourceFile{Name: "Later.java", Source: []byte(`package later;class Later{void added(){}}`)}
	if err := later.ParseAST(); err != nil {
		t.Fatal(err)
	}
	later.ParseSymbols()
	symbol.AddSymbolsToPackage(later.Symbols)
	ResolveFile(later)
	later.Symbols.FindClassScope("Later").Methods[0].HelperName = base + "3"
	verify(base + "4")
	// A new conversion resets the graph and cannot retain any prior collision.
	fresh := setupParseHelper(t, `class Fresh{void other(){}}`)
	scope := overrideBridgeTestScope(t, fresh, "Fresh")
	if got := sourceNativeComparableAllocatedIdentifier(base, scope, comparableFilterFreshContext(fresh.Ctx)); got != base {
		t.Fatalf("prior graph leaked: %q", got)
	}
}

type comparableFilterObservation struct {
	Name    string
	AST     string
	Imports []string
	Logs    string
	Panic   string
}

// The baseline and successor drivers compare these complete observations byte for byte.
func TestComparableNameFilterRenderEquivalence(t *testing.T) {
	cases := comparableFilterBranchCases()
	cases = append(cases,
		comparableFilterBranchCase{name: "native-comparable", source: `package p; class Item{} class Value<T extends Item> implements java.lang.Comparable<T>{public int compareTo(T x){return 7;} public int compareTo(String x){return 8;}}`},
		comparableFilterBranchCase{name: "namespace-shield", source: `package p;interface Comparable<T>{int compareTo(T x);}class Value implements Comparable<String>{public int compareTo(String x){return 1;}}`},
		comparableFilterBranchCase{name: "binder-shield", source: `package p;class Value<java> implements java.lang.Comparable<String>{public int compareTo(String x){return 1;}}`},
		comparableFilterBranchCase{name: "import-shield", source: `package p;import unknown.*;class Value implements java.lang.Comparable<String>{public int compareTo(String x){return 1;}}`},
		comparableFilterBranchCase{name: "array-control", source: `class Value implements java.lang.Comparable<String[]>{public int compareTo(String[] x){return 1;}}`},
		comparableFilterBranchCase{name: "unicode-dollar", source: `class Écho$Owner {void $act(){} class Inner{void act(){}}}`},
	)
	logger := log.StandardLogger()
	oldOutput, oldFormatter := logger.Out, logger.Formatter
	t.Cleanup(func() { logger.SetOutput(oldOutput); logger.SetFormatter(oldFormatter) })
	logger.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableColors: true})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture := func() (out comparableFilterObservation) {
				out.Name = tc.name
				var logs bytes.Buffer
				logger.SetOutput(&logs)
				defer func() {
					if r := recover(); r != nil {
						out.Panic = fmt.Sprint(r)
					}
					out.Logs = logs.String()
				}()
				h := setupParseHelper(t, tc.source)
				ctx := comparableFilterFreshContext(h.Ctx)
				node := ParseNode(h.File.Ast, h.File.Source, ctx)
				file, ok := node.(*ast.File)
				if !ok {
					t.Fatalf("expected AST file,got%T", node)
				}
				var text bytes.Buffer
				if err := printer.Fprint(&text, token.NewFileSet(), file); err != nil {
					t.Fatal(err)
				}
				out.AST = text.String()
				for _, spec := range file.Imports {
					out.Imports = append(out.Imports, spec.Path.Value)
				}
				sort.Strings(out.Imports)
				return out
			}
			first, second := capture(), capture()
			if !reflect.DeepEqual(first, second) {
				t.Fatal("cold render observations differ")
			}
			if first.AST == "" && first.Panic == "" {
				t.Fatal("neither output nor control outcome captured")
			}
			encoded, err := json.Marshal(first)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("EXACT_OBSERVATION %s", encoded)
		})
	}
}

func TestComparableNameFilterSpellingLanguage(t *testing.T) {
	for _, name := range []string{"Act", "act", "Écho$Act", "$Act", "type_", "OuterInnerAct", "NameJava2goExecution"} {
		for _, suffix := range []string{"Java2goExecution", "Java2goExactExecution", "Java2goEntryBodyExecution", "Java2goIterationBodyExecution"} {
			for _, number := range []string{"", "1", "9", "10", "123456789", "9999999999999999999999999999999999999999"} {
				candidate := name + suffix + number
				if !sourceNativeComparableExecutionNameMayEqual(name, candidate) {
					t.Errorf("possible emitted spelling rejected: %q %q", name, candidate)
				}
			}
			for _, number := range []string{"0", "01", "-1", "+1", "١", "1x", "1Java2goExecution"} {
				candidate := name + suffix + number
				if sourceNativeComparableExecutionNameMayEqual(name, candidate) {
					t.Errorf("impossible number accepted: %q %q", name, candidate)
				}
			}
		}
		for _, candidate := range []string{"OtherJava2goExecution", name, name + "Java2go", name + "Java2goExactExecutio", name + "Java2goExecutionx", name + "Java2goUnprovedExecution"} {
			if sourceNativeComparableExecutionNameMayEqual(name, candidate) {
				t.Errorf("unrelated spelling accepted: %q %q", name, candidate)
			}
		}
	}
	for _, args := range [][2]string{{"", ""}, {"", "Other"}, {"Act", ""}} {
		if !sourceNativeComparableExecutionNameMayEqual(args[0], args[1]) {
			t.Errorf("degenerate shape did not fall through: %q", args)
		}
	}
	if allocations := testing.AllocsPerRun(100, func() {
		if sourceNativeComparableExecutionNameMayEqual("Act", "UnrelatedAllocatedEntry") {
			t.Fatal("impossible prefix accepted")
		}
		if !sourceNativeComparableExecutionNameMayEqual("Act", "ActJava2goExactExecution123") {
			t.Fatal("possible exact spelling rejected")
		}
	}); allocations != 0 {
		t.Fatalf("spelling-only filter allocated: %.0f", allocations)
	}
}
