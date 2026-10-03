package transpiler

import (
	"fmt"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestConversionOwnership_CompleteGraphLookupsDoNotScanUnrelatedFiles(t *testing.T) {
	t.Run("registered-and-nested-source-identities", func(t *testing.T) {
		helper := setupParseHelper(t, `package p;class Outer<T>{class Inner<U>{}}`)
		other := parsing.SourceFile{Name: "Other.java", Source: []byte(`package q;class Outer<T>{class Inner<U>{}}`)}
		if err := other.ParseAST(); err != nil {
			t.Fatal(err)
		}
		other.ParseSymbols()
		symbol.AddSymbolsToPackage(other.Symbols)
		ResolveFiles([]parsing.SourceFile{helper.File, other})
		ctx := helper.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		left, right := helper.File.Symbols.FindClassScope("Outer"), other.Symbols.FindClassScope("Outer")
		if left.TypeParameters[0].Declaration == right.TypeParameters[0].Declaration {
			t.Fatal("same-spelled classes lost distinct binders")
		}
		for _, row := range []struct {
			scope *symbol.ClassScope
			file  *symbol.FileScope
		}{{left, helper.File.Symbols}, {left.Subclasses[0], helper.File.Symbols}, {right, other.Symbols}, {right.Subclasses[0], other.Symbols}} {
			got := classScopeCtx(row.scope, ctx)
			if got.currentFile != row.file || got.currentClass != row.scope || got.currentClass.TypeParameters[0].Declaration != row.scope.TypeParameters[0].Declaration {
				t.Fatal("declaring file or class-owned parameter identity changed")
			}
		}
	})
	t.Run("fresh-graph-with-reused-context", func(t *testing.T) {
		old := setupParseHelper(t, `class Outer<T>{class Inner<U>{}}`)
		ctx := old.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		oldScope := old.File.Symbols.FindClassScope("Outer").Subclasses[0]
		if classScopeCtx(oldScope, ctx).currentFile != old.File.Symbols {
			t.Fatal("old graph's source owner disappeared")
		}
		fresh := setupParseHelper(t, `class Outer<T>{class Inner<U>{}}`)
		freshCtx := fresh.Ctx.Clone()
		freshCtx.callableSubclasses = ctx.callableSubclasses
		freshScope := fresh.File.Symbols.FindClassScope("Outer").Subclasses[0]
		if freshScope == oldScope || freshScope.TypeParameters[0].Declaration == oldScope.TypeParameters[0].Declaration || classScopeCtx(freshScope, freshCtx).currentFile != fresh.File.Symbols || classScopeCtx(oldScope, freshCtx).currentFile != fresh.File.Symbols {
			t.Fatal("new graph retained old file/declaration identities")
		}
	})
	t.Run("recreated-and-later-synthetic-owner-fallback", func(t *testing.T) {
		helper := setupParseHelper(t, `class Outer<T>{class Inner<U>{}}`)
		other := parsing.SourceFile{Name: "Other.java", Source: []byte(`package q;class Other{}`)}
		if err := other.ParseAST(); err != nil {
			t.Fatal(err)
		}
		other.ParseSymbols()
		symbol.AddSymbolsToPackage(other.Symbols)
		ResolveFiles([]parsing.SourceFile{helper.File, other})
		ctx := helper.Ctx.Clone()
		ctx.currentFile = other.Symbols
		ctx.currentClass = other.Symbols.BaseClass
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		outer := helper.File.Symbols.FindClassScope("Outer")
		_ = classScopeCtx(outer, ctx)
		synthetic := &symbol.ClassScope{Class: &symbol.Definition{Name: "Inner"}, Enclosing: outer}
		if synthetic == outer.Subclasses[0] || classScopeCtx(synthetic, ctx).currentFile != helper.File.Symbols {
			t.Fatal("recreated declaration lost enclosing-owner fallback")
		}
		// Later lowering may register a synthetic scope after the complete source
		// index was built. A missing entry must retain ordinary ownership search.
		synthetic.Enclosing = nil
		outer.Subclasses = append(outer.Subclasses, synthetic)
		if classScopeCtx(synthetic, ctx).currentFile != helper.File.Symbols {
			t.Fatal("later synthetic declaration lost source ownership")
		}
	})
	t.Run("repeated-ownership-query-budget", func(t *testing.T) {
		measure := func(files int) int64 {
			helper := setupParseHelper(t, `package p;class Target<T>{}`)
			active := parsing.SourceFile{Name: "Active.java", Source: []byte(`package q;class Active{}`)}
			if err := active.ParseAST(); err != nil {
				t.Fatal(err)
			}
			active.ParseSymbols()
			symbol.AddSymbolsToPackage(active.Symbols)
			complete := []parsing.SourceFile{helper.File, active}
			for index := 0; index < files; index++ {
				file := parsing.SourceFile{Name: fmt.Sprintf("Unrelated%d.java", index), Source: []byte(fmt.Sprintf(`package p;class Unrelated%d{}`, index))}
				if err := file.ParseAST(); err != nil {
					t.Fatal(err)
				}
				file.ParseSymbols()
				symbol.AddSymbolsToPackage(file.Symbols)
				complete = append(complete, file)
			}
			ResolveFiles(complete)
			if len(allSourceClassScopes()) != files+2 {
				t.Fatal("complete legal source graph was not retained")
			}
			target := helper.File.Symbols.FindClassScope("Target")
			ctx := helper.Ctx.Clone()
			ctx.currentFile = active.Symbols
			ctx.currentClass = active.Symbols.BaseClass
			ctx.callableSubclasses = &callableSubclassSourceInventory{}
			var result Ctx
			benchmark := testing.Benchmark(func(b *testing.B) {
				for index := 0; index < b.N; index++ {
					result = classScopeCtx(target, ctx)
				}
			})
			if result.currentFile != helper.File.Symbols || result.currentClass != target || result.currentClass.TypeParameters[0].Declaration != target.TypeParameters[0].Declaration {
				t.Fatal("measured lookup changed source owner or binder")
			}
			return benchmark.NsPerOp()
		}
		one := measure(0)
		dense := measure(512)
		t.Logf("same declaration ownership query: two-file ns/op=%d complete514-file ns/op=%d ratio=%.2f", one, dense, float64(dense)/float64(one))
		if one == 0 || dense > 6*one {
			t.Fatalf("CONVERSION_FILE_OWNERSHIP_REPEATED_GRAPH_SCAN: %d ns/op >6*same-query two-file baseline %d", dense, one)
		}
	})
}
