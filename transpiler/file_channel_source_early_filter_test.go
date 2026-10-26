package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func earlyFilterCostFixture(tb testing.TB) (Ctx, *symbol.Definition, *sitter.Node) {
	tb.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	tb.Cleanup(func() { symbol.GlobalScope = previous })
	java := "class Ordinary0 {}\n"
	for i := 1; i <= 8; i++ {
		java += fmt.Sprintf("class Ordinary%d extends Ordinary%d {}\n", i, i-1)
	}
	java += `class Cost extends Ordinary8 { int unrelated(){return 0;} int invoke(){return unrelated();} }`
	f := parsing.SourceFile{Name: "Cost.java", Source: []byte(java)}
	if err := f.ParseAST(); err != nil {
		tb.Fatal(err)
	}
	file := f.ParseSymbols()
	symbol.AddSymbolsToPackage(file)
	ctx := Ctx{currentFile: file, currentClass: file.BaseClass}
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Cost")
	if ctx.currentClass == nil {
		tb.Fatal("actual Cost declaration missing")
	}
	ctx.className = ctx.currentClass.Class.Name
	method := ctx.currentClass.FindMethod().ByOriginalName("invoke")[0]
	ctx.localScope = method
	var call *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			call = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(method.DeclarationNode)
	if call == nil {
		tb.Fatal("actual invocation missing")
	}
	return ctx, ctx.currentClass.FindMethod().ByOriginalName("unrelated")[0], call
}

func BenchmarkFileChannelEarlyFilterUnrelatedMethod(b *testing.B) {
	ctx, method, _ := earlyFilterCostFixture(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if fileChannelSourceProtocolSlot(method, ctx.currentClass, ctx) != "" {
			b.Fatal("unrelated method admitted")
		}
	}
}
func BenchmarkFileChannelEarlyFilterWrongDescriptor(b *testing.B) {
	ctx, actual, _ := earlyFilterCostFixture(b)
	method := *actual
	method.OriginalName = "write"
	method.OriginalType = "boolean"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if fileChannelSourceProtocolSlot(&method, ctx.currentClass, ctx) != "" {
			b.Fatal("wrong descriptor admitted")
		}
	}
}
func BenchmarkFileChannelEarlyFilterUnrelatedLifecycle(b *testing.B) {
	ctx, _, call := earlyFilterCostFixture(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if fileChannelSourceInheritedLifecycle(call, ctx, ctx.currentFile.Source) {
			b.Fatal("unrelated method admitted")
		}
	}
}

func TestFileChannelEarlyFilterDescriptorContracts(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{"Root.java": `abstract class Root extends java.nio.channels.FileChannel {
 public int read(java.nio.ByteBuffer b){return 0;} public int read(java.nio.ByteBuffer b,long p){return 0;}
 public int write(java.nio.ByteBuffer b){return 0;} public int write(java.nio.ByteBuffer b,long p){return 0;}
 protected void implCloseChannel(){} int unrelated(){return 0;}
 }`}, "Root.java")
	want := map[string]map[int]string{"read": {1: fileChannelSourceReadSlot, 2: fileChannelSourceReadAtSlot}, "write": {1: fileChannelSourceWriteSlot, 2: fileChannelSourceWriteAtSlot}, "implCloseChannel": {0: fileChannelSourceCloseSlot}, "unrelated": {0: ""}}
	for name, arities := range want {
		for _, m := range ctx.currentClass.FindMethod().ByOriginalName(name) {
			if got := fileChannelSourceProtocolSlot(m, ctx.currentClass, ctx); got != arities[len(m.Parameters)] {
				t.Errorf("%s/%d=%q", name, len(m.Parameters), got)
			}
		}
	}
	base := ctx.currentClass.FindMethod().ByOriginalName("write")[1]
	for name, change := range map[string]func(*symbol.Definition){
		"static": func(m *symbol.Definition) { m.IsStatic = true }, "private": func(m *symbol.Definition) { m.IsPrivate = true }, "constructor": func(m *symbol.Definition) { m.Constructor = true },
		"wrong-return": func(m *symbol.Definition) { m.OriginalType = "boolean" }, "wrong-name": func(m *symbol.Definition) { m.OriginalName = "ordinary" },
		"arity0": func(m *symbol.Definition) { m.Parameters = nil }, "arity3": func(m *symbol.Definition) {
			m.Parameters = append(append([]*symbol.Definition(nil), m.Parameters...), m.Parameters[0])
		},
		"nil-first": func(m *symbol.Definition) {
			m.Parameters = append([]*symbol.Definition(nil), m.Parameters...)
			m.Parameters[0] = nil
		},
		"varargs": func(m *symbol.Definition) { m.IsVariadic = true },
	} {
		t.Run(name, func(t *testing.T) {
			m := *base
			change(&m)
			if got := fileChannelSourceProtocolSlot(&m, ctx.currentClass, ctx); got != "" {
				t.Fatalf("invalid descriptor got%q", got)
			}
		})
	}
}
func TestFileChannelEarlyFilterNilDetachedCycleGuards(t *testing.T) {
	ctx, method, call := earlyFilterCostFixture(t)
	if fileChannelSourceProtocolSlot(nil, ctx.currentClass, ctx) != "" || fileChannelSourceProtocolSlot(method, nil, ctx) != "" {
		t.Fatal("nil input admitted")
	}
	malformed := *method
	malformed.OriginalName = "write"
	malformed.OriginalType = "int"
	malformed.Parameters = []*symbol.Definition{{OriginalType: "java.nio.ByteBuffer"}, nil}
	for name, scope := range map[string]*symbol.ClassScope{"nil": nil, "ordinary": ctx.currentClass, "detached": {Superclass: "java.nio.channels.FileChannel"}} {
		t.Run(name, func(t *testing.T) {
			if fileChannelSourceProtocolSlot(&malformed, scope, ctx) != "" {
				t.Fatal("unproved owner admitted")
			}
		})
	}
	cycle := ordinaryDemandTestContext(t, map[string]string{"Cycle.java": `class Cycle extends Other {int unrelated(){return 0;}} class Other extends Cycle {}`}, "Cycle.java")
	if fileChannelSourceProtocolSlot(method, cycle.currentClass, cycle) != "" {
		t.Fatal("cyclic ordinary owner admitted")
	}
	if fileChannelSourceInheritedLifecycle(nil, ctx, ctx.currentFile.Source) || fileChannelSourceInheritedLifecycle(call, Ctx{}, ctx.currentFile.Source) {
		t.Fatal("nil lifecycle context admitted")
	}
}

// The allowance covers node-kind inspection, not a source-ancestry walk for a
// method name that cannot select a FileChannel protocol declaration.
func TestFileChannelEarlyFilterNegativeWorkBudget(t *testing.T) {
	ctx, method, call := earlyFilterCostFixture(t)
	slots := testing.AllocsPerRun(100, func() {
		if fileChannelSourceProtocolSlot(method, ctx.currentClass, ctx) != "" {
			t.Fatal("unrelated method admitted")
		}
	})
	lifecycle := testing.AllocsPerRun(100, func() {
		if fileChannelSourceInheritedLifecycle(call, ctx, ctx.currentFile.Source) {
			t.Fatal("unrelated lifecycle admitted")
		}
	})
	if slots > 1 || lifecycle > 2 {
		t.Fatalf("unrelated candidates must not walk ancestry: method %.0f allocations (max1), lifecycle %.0f (max2)", slots, lifecycle)
	}
}

func TestFileChannelEarlyFilterMalformedCanonicalNilLegacyGuard(t *testing.T) {
	// This synthetic descriptor has no valid Java declaration. Preserve the old
	// canonical-owner failure while protecting rejected owners from new panics.
	ctx := ordinaryDemandTestContext(t, map[string]string{"Root.java": `abstract class Root extends java.nio.channels.FileChannel {public int write(java.nio.ByteBuffer b,long p){return 0;}}`}, "Root.java")
	method := *ctx.currentClass.FindMethod().ByOriginalName("write")[0]
	method.Parameters = append([]*symbol.Definition(nil), method.Parameters...)
	method.Parameters[1] = nil
	didPanic := false
	func() {
		defer func() { didPanic = recover() != nil }()
		_ = fileChannelSourceProtocolSlot(&method, ctx.currentClass, ctx)
	}()
	if !didPanic {
		t.Fatal("malformed canonical descriptor changed legacy failure mode")
	}
}
