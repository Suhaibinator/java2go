package transpiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func declaredTypeAllocationFixture(tb testing.TB, padding int) (Ctx, *symbol.FileScope) {
	tb.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	tb.Cleanup(func() { symbol.GlobalScope = previous })
	var text strings.Builder
	text.WriteString("package p;\n")
	for index := 0; index < padding; index++ {
		fmt.Fprintf(&text, "class Padding%d {}\n", index)
	}
	text.WriteString("class RootOwner {static class Inner {static class Leaf {}}}\n")
	inputs := []parsing.SourceFile{
		{Name: "p/Roots.java", Source: []byte(text.String())},
		{Name: "q/RootOwner.java", Source: []byte("package q;class RootOwner {static class Foreign {}}")},
	}
	for index := range inputs {
		file := &inputs[index]
		if err := file.ParseAST(); err != nil {
			tb.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
	}
	for _, file := range inputs {
		ResolveFile(file)
	}
	owner := inputs[0].Symbols.FindClassScope("RootOwner")
	if owner == nil || owner.Class == nil {
		tb.Fatal("missing actual source owner")
	}
	return Ctx{currentFile: inputs[0].Symbols, currentClass: owner}, inputs[0].Symbols
}

// The frozen original helper supplies the explicitly requested differential
// control. Independent expected source declarations below also guard semantics.
func legacyDeclaredFileTypeAllocationControl(name string, file *symbol.FileScope, ctx Ctx) *symbol.ClassScope {
	if file == nil {
		return nil
	}
	parts := strings.Split(name, ".")
	for _, top := range file.TopLevelClasses {
		if top.Class != nil && top.Class.OriginalName == parts[0] {
			return directMemberTypePath(top, parts[1:], ctx)
		}
	}
	return nil
}

func TestDeclaredFileTypeAllocationBudget(t *testing.T) {
	for _, padding := range []int{32, 544} {
		t.Run(fmt.Sprint(padding), func(t *testing.T) {
			ctx, file := declaredTypeAllocationFixture(t, padding)
			for _, name := range []string{"RootOwner", "Absent", "RootOwner.Inner.Leaf", "RootOwner.Missing"} {
				want := legacyDeclaredFileTypeAllocationControl(name, file, ctx)
				allocations := testing.AllocsPerRun(20, func() {
					if got := declaredFileType(name, file, ctx); got != want {
						t.Fatal("source declaration identity changed")
					}
				})
				t.Logf("source-files=2 padding=%d name=%s allocations/query=%.0f", padding, name, allocations)
				if allocations != 0 {
					t.Errorf("DECLARED_FILE_NAME_SLICE_ALLOCATION padding=%d name=%s got=%.0f want=0", padding, name, allocations)
				}
			}
		})
	}
}

func BenchmarkDeclaredFileTypeNameWalk(b *testing.B) {
	for _, padding := range []int{32, 544} {
		for _, name := range []string{"RootOwner", "Absent", "RootOwner.Inner.Leaf", "RootOwner.Missing"} {
			b.Run(fmt.Sprintf("Owners-%d/%s", padding, name), func(b *testing.B) {
				ctx, file := declaredTypeAllocationFixture(b, padding)
				want := legacyDeclaredFileTypeAllocationControl(name, file, ctx)
				b.ReportAllocs()
				b.ResetTimer()
				for index := 0; index < b.N; index++ {
					if got := declaredFileType(name, file, ctx); got != want {
						b.Fatal("source declaration identity changed")
					}
				}
				b.StopTimer()
			})
		}
	}
}

func TestDeclaredFileTypeSegmentsAndOwnerIdentity(t *testing.T) {
	ctx, file := declaredTypeAllocationFixture(t, 2)
	owner := ctx.currentClass
	inner := owner.Subclasses[0]
	leaf := inner.Subclasses[0]
	for _, control := range []struct {
		name string
		want *symbol.ClassScope
	}{
		{"RootOwner", owner}, {"RootOwner.Inner", inner}, {"RootOwner.Inner.Leaf", leaf},
		{"Absent", nil}, {"RootOwner.Foreign", nil}, {"q.RootOwner.Foreign", nil},
		{"", nil}, {".", nil}, {"..", nil}, {".RootOwner", nil}, {"RootOwner.", nil},
		{"RootOwner..Inner", nil}, {"RootOwner.Inner..Leaf", nil}, {"RootOwner..", nil},
	} {
		t.Run(control.name, func(t *testing.T) {
			old, got := legacyDeclaredFileTypeAllocationControl(control.name, file, ctx), declaredFileType(control.name, file, ctx)
			if old != control.want || got != control.want {
				t.Fatalf("%q old=%p got=%p want actual declaration %p", control.name, old, got, control.want)
			}
		})
	}
	if declaredFileType("RootOwner", nil, ctx) != nil {
		t.Fatal("nil file selected source owner")
	}
	foreign := findQualifiedSourceClass("q.RootOwner")
	copyFile := *file
	copyFile.TopLevelClasses = []*symbol.ClassScope{{}, foreign, owner}
	// First matching owner is authoritative even when a later same-name owner has
	// the requested member. Do not turn this walk into a whole-graph search.
	if old, got := legacyDeclaredFileTypeAllocationControl("RootOwner.Inner", &copyFile, ctx), declaredFileType("RootOwner.Inner", &copyFile, ctx); old != nil || got != nil {
		t.Fatal("later source owner replaced first matching owner")
	}
}

func TestDeclaredFileTypeInheritedMembersAndFreshGraphs(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Parent.java": `package p;public class Parent {public static class PublicMember{} static class PackageMember{}}`,
		"q/Child.java":  `package q;public class Child extends p.Parent {public static class Own{}}`,
	})
	child := findQualifiedSourceClass("q.Child")
	ctx = classScopeCtx(child, ctx)
	file := ctx.currentFile
	for _, control := range []struct{ name, qualified string }{
		{"Child.Own", "q.Child.Own"}, {"Child.PublicMember", "p.Parent.PublicMember"}, {"Child.PackageMember", ""},
	} {
		want := findQualifiedSourceClass(control.qualified)
		if old, got := legacyDeclaredFileTypeAllocationControl(control.name, file, ctx), declaredFileType(control.name, file, ctx); old != want || got != want {
			t.Fatalf("%s old=%p got=%p want=%p", control.name, old, got, want)
		}
	}
	old := declaredFileType("Child.Own", file, ctx)
	fresh := setupParseHelper(t, `class Child {static class Own{}}`)
	got := declaredFileType("Child.Own", fresh.Ctx.currentFile, fresh.Ctx)
	if got == nil || got == old || got.Enclosing != fresh.Ctx.currentClass {
		t.Fatal("new source graph reused an old member declaration")
	}
}
