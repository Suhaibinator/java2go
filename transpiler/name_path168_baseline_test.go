package transpiler

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

type namePath168Work struct {
	Name    string
	Allocs  float64
	Bytes   uint64
	Mallocs uint64
	Cgo     int64
	Calls   int
}

func namePath168Measure(t *testing.T, name string, work func()) namePath168Work {
	t.Helper()
	work()
	a := testing.AllocsPerRun(5, work)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cgo := runtime.NumCgoCall()
	for repeat := 0; repeat < 5; repeat++ {
		work()
	}
	cgo = runtime.NumCgoCall() - cgo
	runtime.ReadMemStats(&after)
	r := namePath168Work{name, a, (after.TotalAlloc - before.TotalAlloc) / 5, (after.Mallocs - before.Mallocs) / 5, cgo / 5, 5}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("WORK168 " + string(b))
	return r
}

func TestNamePath168DerivedNamesAndMutation(t *testing.T) {
	helper := setupParseHelper(t, "class Original { int field; void method(){} } enum E { ITEM }")
	owner := helper.Ctx.currentClass
	check := func(query string) {
		t.Helper()
		if got, want := generatedIdentifierExists(query, owner), namePath168OldGeneratedIdentifierExists(query, owner); got != want {
			t.Fatalf("query %q got %v old %v", query, got, want)
		}
	}
	names := []string{"", "Original", "original", "δelta", "Δelta", "$Dollar", "type", "If", "\xffbroken"}
	for _, name := range names {
		owner.Class.Name = name
		for _, query := range []string{name, "", "missing", classDispatchTypeName(owner), classDispatchFieldName(owner), classSelfSetterName(owner), interfaceDefaultCarrierName(owner), "Java2goWrongSelf", "__java2goWrongSelf", "java2goSetWrongSelf", "MissingJava2goDispatch", "MissingJava2goDefaults"} {
			check(query)
		}
	}
	owner.Class.Name = "Original"
	query := "lateJava2goExecution"
	check(query)
	owner.Fields = append(owner.Fields, &symbol.Definition{Name: query})
	check(query)
	if !generatedIdentifierExists(query, owner) {
		t.Fatal("late field invisible")
	}
	owner.Fields = owner.Fields[:len(owner.Fields)-1]
	owner.Methods = append(owner.Methods, &symbol.Definition{Name: query})
	check(query)
	owner.Methods = owner.Methods[:len(owner.Methods)-1]
	nested := &symbol.ClassScope{Class: &symbol.Definition{Name: query, OriginalName: "Late"}, Enclosing: owner}
	owner.Subclasses = append(owner.Subclasses, nested)
	check(query)
	owner.Subclasses = owner.Subclasses[:len(owner.Subclasses)-1]
	nilClass := &symbol.ClassScope{Fields: []*symbol.Definition{{Name: "field"}}}
	for _, scope := range []*symbol.ClassScope{nil, nilClass, owner} {
		for _, q := range []string{"", "field", "missing"} {
			if generatedIdentifierExists(q, scope) != namePath168OldGeneratedIdentifierExists(q, scope) {
				t.Fatal("nil/empty mismatch")
			}
		}
	}
	base := "OccupiedJava2goExecution"
	owner.Fields = append(owner.Fields, &symbol.Definition{Name: base}, &symbol.Definition{Name: base + "1"})
	if got := collisionSafeExecutionIdentifier(base, owner); got != base+"2" {
		t.Fatalf("collision identity %q", got)
	}
	owner.Fields = owner.Fields[:len(owner.Fields)-2]
	if got := collisionSafeExecutionIdentifier(base, owner); got != base {
		t.Fatalf("removed collision %q", got)
	}
	// A malformed enum field panics before the derived-name arm in both versions.
	owner.EnumConstants = append(owner.EnumConstants, symbol.EnumConstant{Name: "malformed"})
	panics := func(f func()) (value any) { defer func() { value = recover() }(); f(); return }
	if panics(func() { generatedIdentifierExists("missing", owner) }) == nil || panics(func() { namePath168OldGeneratedIdentifierExists("missing", owner) }) == nil {
		t.Fatal("old enum panic must remain")
	}
	owner.EnumConstants = owner.EnumConstants[:len(owner.EnumConstants)-1]
}

func TestNamePath168DerivedWorkBaseline(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 128; i++ {
		fmt.Fprintf(&source, "class Scope%03d {}\n", i)
	}
	helper := setupParseHelper(t, source.String())
	work := func() {
		for i := 0; i < 16; i++ {
			if generatedIdentifierExists(fmt.Sprintf("Absent%03dJava2goExecution", i), helper.Ctx.currentClass) {
				t.Fatal("negative collision")
			}
		}
	}
	namePath168Measure(t, "derived-negative-128x16", work)
	runtime.KeepAlive(helper)
}

func namePath168Parse(t *testing.T, name, source string) parsing.SourceFile {
	t.Helper()
	file := parsing.SourceFile{Name: name, Source: []byte(source)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	return file
}

func TestNamePath168HierarchyBehaviorAndMutation(t *testing.T) {
	helper := setupParseHelper(t, "interface Root { class Mark {} } interface Left extends Root {} interface Right extends Root {} class Child implements Left, Right {} class Hidden extends Child { class Mark {} }")
	root := helper.File.Symbols.FindClassScope("Root")
	child := helper.File.Symbols.FindClassScope("Child")
	hidden := helper.File.Symbols.FindClassScope("Hidden")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = child
	retained := ctx.Clone()
	for i := 0; i < 3; i++ {
		if got := memberTypeInHierarchy(child, "Mark", ctx); got != root.Subclasses[0] {
			t.Fatalf("diamond identity %v", got)
		}
		if got := memberTypeInHierarchy(hidden, "Mark", ctx); got != hidden.Subclasses[0] {
			t.Fatalf("hidden identity %v", got)
		}
		if got := memberTypeInHierarchy(child, "Absent", ctx); got != nil {
			t.Fatal("unexpected member")
		}
	}
	if len(ctx.memberTypeLookupPath) != 0 || len(retained.memberTypeLookupPath) != 0 {
		t.Fatal("caller path escaped")
	}
	late := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Late", Name: "Late"}, Enclosing: child}
	child.Subclasses = append(child.Subclasses, late)
	if memberTypeInHierarchy(child, "Late", ctx) != late {
		t.Fatal("late nested scope invisible")
	}
	child.Subclasses = child.Subclasses[:len(child.Subclasses)-1]
	if memberTypeInHierarchy(child, "Late", ctx) != nil {
		t.Fatal("removed nested scope retained")
	}
	old := child.ImplementedInterfaces
	child.ImplementedInterfaces = nil
	if memberTypeInHierarchy(child, "Mark", ctx) != nil {
		t.Fatal("removed interfaces retained")
	}
	child.Superclass = "Hidden"
	if memberTypeInHierarchy(child, "Mark", ctx) != hidden.Subclasses[0] {
		t.Fatal("new superclass invisible")
	}
	child.Superclass = ""
	child.ImplementedInterfaces = old
	ctx.memberTypeLookupPath = map[*symbol.ClassScope]bool{child: false, root: false}
	before := map[*symbol.ClassScope]bool{child: false, root: false}
	memberTypeInHierarchy(child, "Absent", ctx)
	if !reflect.DeepEqual(ctx.memberTypeLookupPath, before) {
		t.Fatal("seeded false-key caller mutated")
	}
	ctx.memberTypeLookupPath[child] = true
	if memberTypeInHierarchy(child, "Absent", ctx) != nil {
		t.Fatal("active cycle admitted")
	}
	if memberTypeInHierarchy(hidden, "Mark", Ctx{memberTypeLookupPath: map[*symbol.ClassScope]bool{hidden: true}}) != hidden.Subclasses[0] {
		t.Fatal("direct member must precede cycle guard")
	}
	t.Log("IDENTITY168 diamond=Root.Mark hidden=Hidden.Mark late=live superclass=live falseSeeds=preserved directBeforeCycle=true")
	runtime.KeepAlive(helper)
}

func TestNamePath168HierarchyDiagnosticOrderAndPanic(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous; setStrictMode(false); resetDiagnostics() })
	a := namePath168Parse(t, "A.java", "package p; public class A { public static class Clash {} }")
	b := namePath168Parse(t, "B.java", "package p; public class B { public static class Clash {} }")
	file := namePath168Parse(t, "Owner.java", "import static p.A.*; import static p.B.*; class Owner extends Clash {}")
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, className: "Owner"}
	setStrictMode(false)
	resetDiagnostics()
	for i := 0; i < 2; i++ {
		if memberTypeInHierarchy(ctx.currentClass, "Absent", ctx) != nil {
			t.Fatal("ambiguous owner selected")
		}
	}
	d := Diagnostics()
	if len(d) < 2 {
		t.Fatalf("expected repeated ambiguity diagnostics: %#v", d)
	}
	for _, diag := range d {
		if diag.Kind != "ambiguous imported type Clash" {
			t.Fatalf("diagnostic order/kind %#v", d)
		}
	}
	encoded, _ := json.Marshal(d)
	t.Log("DIAGNOSTICS168 " + string(encoded))
	setStrictMode(true)
	resetDiagnostics()
	var caught any
	func() { defer func() { caught = recover() }(); memberTypeInHierarchy(ctx.currentClass, "Absent", ctx) }()
	if _, ok := caught.(strictModeError); !ok {
		t.Fatalf("strict panic %T", caught)
	}
	if got := Diagnostics(); len(got) != 1 || got[0] != d[0] {
		t.Fatalf("strict first diagnostic %#v", got)
	}
	if len(ctx.memberTypeLookupPath) != 0 {
		t.Fatal("panic escaped path into caller")
	}
	runtime.KeepAlive(a)
	runtime.KeepAlive(b)
	runtime.KeepAlive(file)
}

func TestNamePath168HierarchyWorkBaseline(t *testing.T) {
	var source strings.Builder
	source.WriteString("class H0 {}\n")
	for i := 1; i <= 8; i++ {
		fmt.Fprintf(&source, "class H%d extends H%d {}\n", i, i-1)
	}
	helper := setupParseHelper(t, source.String())
	owner := helper.File.Symbols.FindClassScope("H8")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = owner
	work := func() {
		for i := 0; i < 8; i++ {
			if memberTypeInHierarchy(owner, "Absent", ctx) != nil {
				t.Fatal("negative hierarchy result")
			}
		}
	}
	namePath168Measure(t, "hierarchy-depth8x8", work)
	if len(ctx.memberTypeLookupPath) != 0 {
		t.Fatal("work escaped caller path")
	}
	runtime.KeepAlive(helper)
}
