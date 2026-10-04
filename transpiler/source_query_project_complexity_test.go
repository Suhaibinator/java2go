package transpiler

import (
	"fmt"
	"go/ast"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// Repeated renders must not multiply work on unrelated source syntax. Both
// measurements lower the same 32 legal getter files; only unrelated complete-
// graph syntax differs. Source names and field/method/binder decisions are not
// cached by this test, and all source stays registered throughout each render.
func TestSourceQueryInventory_ProjectFilesDoNotMultiplyUnrelatedASTWork(t *testing.T) {
	const files = 32
	setup := func(t *testing.T, padded bool) ([]parsing.SourceFile, Ctx) {
		// Use the same standard setup/reset as existing compiler semantic tests.
		helper := setupParseHelper(t, `package p;class Root{}`)
		targets := make([]parsing.SourceFile, 0, files)
		for index := 0; index < files; index++ {
			file := parsing.SourceFile{Name: fmt.Sprintf("Cell%d.java", index), Source: []byte(fmt.Sprintf(`package p;class Cell%d<T> extends Root{public T get(){return null;}}`, index))}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			file.ParseSymbols()
			symbol.AddSymbolsToPackage(file.Symbols)
			targets = append(targets, file)
		}
		complete := append([]parsing.SourceFile{helper.File}, targets...)
		if padded {
			// Keep each legal Java method below its bytecode size limit while
			// increasing only the unrelated complete-graph source inventory.
			var padding strings.Builder
			padding.WriteString("package p;class Padding{")
			for index := 0; index < 8; index++ {
				fmt.Fprintf(&padding, "void noise%d(){int value=0;", index)
				padding.WriteString(strings.Repeat("value++;", 4096))
				padding.WriteString("}")
			}
			padding.WriteString("}")
			file := parsing.SourceFile{Name: "Padding.java", Source: []byte(padding.String())}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			file.ParseSymbols()
			symbol.AddSymbolsToPackage(file.Symbols)
			complete = append(complete, file)
		}
		ResolveFiles(complete)
		expected := files + 1
		if padded {
			expected++
		}
		if len(allSourceClassScopes()) != expected {
			t.Fatal("complete legal source graph was not retained")
		}
		return targets, helper.Ctx
	}
	render := func(t *testing.T, targets []parsing.SourceFile, base Ctx, check bool) {
		// Each complete-graph conversion batch owns fresh structural facts.
		// File contexts share them; every measured batch includes its first scan.
		base.callableSubclasses = &callableSubclassSourceInventory{}
		for _, file := range targets {
			scope := file.Symbols.TopLevelClasses[0]
			declaration := scope.TypeParameters[0].Declaration
			var getter *symbol.Definition
			for _, method := range scope.Methods {
				if method.OriginalName == "get" {
					getter = method
				}
			}
			if getter == nil || getter.OriginalType != "T" || getter.TypeParameterBindings["T"] != declaration {
				t.Fatal("getter lost class-owned declaration identity")
			}
			ctx := base.Clone()
			ctx.currentFile = file.Symbols
			ctx.currentClass = file.Symbols.BaseClass
			ctx.genericFamilies = &genericFamilyAnalysis{}
			node, err := convertFileNode(file, ctx)
			if err != nil {
				t.Fatal(err)
			}
			if check {
				found := 0
				ast.Inspect(node, func(node ast.Node) bool {
					if function, ok := node.(*ast.FuncDecl); ok && function.Recv != nil && function.Name.Name == getter.Name {
						found++
					}
					return true
				})
				if found != 1 || scope.TypeParameters[0].Declaration != declaration || getter.TypeParameterBindings["T"] != declaration {
					t.Fatal("render changed source getter/parameter identity")
				}
			}
		}
	}
	t.Run("registered-source-and-getter-identities", func(t *testing.T) {
		targets, ctx := setup(t, true)
		render(t, targets, ctx, true)
	})
	t.Run("whole-project-inventory-budget", func(t *testing.T) {
		targets, ctx := setup(t, true)
		dense := testing.AllocsPerRun(1, func() { render(t, targets, ctx, false) })
		baselineTargets, baselineCtx := setup(t, false)
		baseline := testing.AllocsPerRun(1, func() { render(t, baselineTargets, baselineCtx, false) })
		for index := range targets {
			if string(targets[index].Source) != string(baselineTargets[index].Source) {
				t.Fatal("target source bytes changed between complete graphs")
			}
		}
		t.Logf("same 32 legal getter files: unpadded allocations=%.0f padded allocations=%.0f ratio=%.2f", baseline, dense, dense/baseline)
		if baseline == 0 || dense > 6*baseline {
			t.Fatalf("SOURCE_QUERY_REPEATED_PROJECT_AST_INVENTORY: %.0f allocations >6*same-target baseline %.0f", dense, baseline)
		}
	})
}
