package transpiler

import (
	"fmt"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestCampaignPackageOwnershipFacts(t *testing.T) {
	withCleanDiagnostics(t)
	previous := activeResolutionFiles
	t.Cleanup(func() { activeResolutionFiles = previous })
	t.Run("same-spelled-nested-declarations-and-late-registration", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Outer.java": `package p;public class Outer<T>{public class Inner<U>{}}`,
			"q/Outer.java": `package q;public class Outer<T>{public class Inner<U>{}}`,
		})
		_ = ctx
		left, right := findQualifiedSourceClass("p.Outer"), findQualifiedSourceClass("q.Outer")
		activeResolutionFiles = newResolutionFileIndex()
		for _, row := range []struct {
			scope *symbol.ClassScope
			pkg   string
		}{{left, "p"}, {left.Subclasses[0], "p"}, {right, "q"}, {right.Subclasses[0], "q"}} {
			if got := findJavaPackageForClassScope(row.scope); got != row.pkg {
				t.Fatalf("declaration package=%q, want %q", got, row.pkg)
			}
		}
		if left.TypeParameters[0].Declaration == right.TypeParameters[0].Declaration {
			t.Fatal("same-spelled owners lost binder identity")
		}
		late := &symbol.ClassScope{Class: &symbol.Definition{Name: "Late", OriginalName: "Late"}, Enclosing: left}
		if got := findJavaPackageForClassScope(late); got != "" {
			t.Fatalf("unregistered scope invented package %q", got)
		}
		left.Subclasses = append(left.Subclasses, late)
		if got := findJavaPackageForClassScope(late); got != "p" {
			t.Fatalf("late registered scope lost package: %q", got)
		}
		orphan := &symbol.ClassScope{Class: &symbol.Definition{Name: "Outer", OriginalName: "Outer"}}
		if got := findJavaPackageForClassScope(orphan); got != "" {
			t.Fatalf("name collision invented ownership %q", got)
		}
		if got := findJavaPackageForClassScope(nil); got != "" {
			t.Fatalf("nil scope invented package %q", got)
		}
	})
	t.Run("old-graph-is-not-authority", func(t *testing.T) {
		old := setupParseHelper(t, `package old;class Outer{class Inner{}}`)
		oldScope := old.File.Symbols.BaseClass
		oldIndex := newResolutionFileIndex()
		fresh := setupParseHelper(t, `package fresh;class Outer{class Inner{}}`)
		activeResolutionFiles = oldIndex
		if got := findJavaPackageForClassScope(fresh.File.Symbols.BaseClass); got != "fresh" {
			t.Fatalf("fresh owner package=%q", got)
		}
		if got := findJavaPackageForClassScope(oldScope); got != "" {
			t.Fatalf("old graph retained package %q", got)
		}
	})
	t.Run("conversion-restores-index-on-success-and-strict-error", func(t *testing.T) {
		helper := setupParseHelper(t, `package p;class Outer{static int answer(){return 37;}}`)
		sentinel := newResolutionFileIndex()
		activeResolutionFiles = sentinel
		ctx := helper.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		if _, err := convertFileNode(helper.File, ctx); err != nil {
			t.Fatal(err)
		}
		if activeResolutionFiles != sentinel {
			t.Fatal("successful conversion leaked active ownership")
		}
		bad := setupParseHelper(t, `package q;class Broken{static int fail(){record UnsupportedLocal(int value){}return 1;}}`)
		activeResolutionFiles = sentinel
		diagnostics.mu.Lock()
		previousStrict := diagnostics.strict
		diagnostics.mu.Unlock()
		setStrictMode(true)
		defer setStrictMode(previousStrict)
		if _, err := convertFileNode(bad.File, bad.Ctx); err == nil {
			t.Fatal("unsupported local record did not retain strict failure")
		}
		if activeResolutionFiles != sentinel {
			t.Fatal("strict failure leaked active ownership")
		}
	})
}

func campaignPackageOwnershipBenchmarkGraph(b *testing.B, padding int) (*symbol.ClassScope, *resolutionFileIndex) {
	b.Helper()
	previousGraph, previousIndex := symbol.GlobalScope, activeResolutionFiles
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	b.Cleanup(func() { symbol.GlobalScope = previousGraph; activeResolutionFiles = previousIndex })
	for index := 0; index <= padding; index++ {
		name := fmt.Sprintf("Noise%d", index)
		if index == padding {
			name = "Target"
		}
		file := parsing.SourceFile{Name: name + ".java", Source: []byte("package p;class " + name + "{class Inner{}}")}
		if err := file.ParseAST(); err != nil {
			b.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
	}
	target := findQualifiedSourceClass("p.Target").Subclasses[0]
	index := newResolutionFileIndex()
	if len(index.files) != 2*(padding+1) {
		b.Fatal("complete benchmark graph lost source declarations")
	}
	return target, index
}

func BenchmarkCampaignPackageOwnership(b *testing.B) {
	for _, padding := range []int{0, 512, 1212} {
		b.Run(fmt.Sprintf("Registered-%d", padding+1), func(b *testing.B) {
			target, index := campaignPackageOwnershipBenchmarkGraph(b, padding)
			activeResolutionFiles = index
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if got := findJavaPackageForClassScope(target); got != "p" {
					b.Fatalf("ownership query=%q", got)
				}
			}
		})
	}
}

func TestCampaignPackageOwnershipQueryBudget(t *testing.T) {
	measure := func(padding int) int64 {
		result := testing.Benchmark(func(b *testing.B) {
			target, index := campaignPackageOwnershipBenchmarkGraph(b, padding)
			activeResolutionFiles = index
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if got := findJavaPackageForClassScope(target); got != "p" {
					b.Fatalf("ownership query=%q", got)
				}
			}
		})
		return result.NsPerOp()
	}
	sparse, dense := measure(31), measure(543)
	t.Logf("same registered nested declaration: same32files=%d ns/op, complete544files=%d ns/op, ratio=%.2f", sparse, dense, float64(dense)/float64(sparse))
	if sparse == 0 || dense > 6*sparse {
		t.Fatalf("PACKAGE_OWNERSHIP_REPEATED_GRAPH_SCAN: %d ns/op >6*same-query 32-file baseline %d", dense, sparse)
	}
}
