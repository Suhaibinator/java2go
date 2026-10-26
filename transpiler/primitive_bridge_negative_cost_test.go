package transpiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func primitiveBridgeCostFixture(tb testing.TB, padding int) (Ctx, *symbol.ClassScope, *symbol.Definition) {
	tb.Helper()
	previous := symbol.GlobalScope
	previousIndex, previousImports := activeResolutionFiles, activeImportSourceInventory
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	tb.Cleanup(func() {
		symbol.GlobalScope = previous
		activeResolutionFiles, activeImportSourceInventory = previousIndex, previousImports
	})
	var text strings.Builder
	text.WriteString("class BridgeCostOwner { void act() {} } class BridgeCostChild extends BridgeCostOwner { @Override void act() {} } class PaddingRoot {}\n")
	for index := 0; index < padding; index++ {
		fmt.Fprintf(&text, "class Padding%d extends PaddingRoot { int value; int get() { return value; } }\n", index)
	}
	file := parsing.SourceFile{Name: "BridgeCostOwner.java", Source: []byte(text.String())}
	if err := file.ParseAST(); err != nil {
		tb.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	ResolveFile(file)
	owner := file.Symbols.FindClassScope("BridgeCostOwner")
	if owner == nil {
		tb.Fatal("missing target declaration")
	}
	var method *symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate.OriginalName == "act" {
			method = candidate
		}
	}
	if method == nil {
		tb.Fatal("missing target method")
	}
	ctx := Ctx{currentFile: file.Symbols, currentClass: owner, callableSubclasses: &callableSubclassSourceInventory{}}
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	activeImportSourceInventory = resolvedSourceInventory(ctx)
	return ctx, owner, method
}

func TestPrimitiveBridgeNegativeAllocationBudget(t *testing.T) {
	measure := func(padding int) float64 {
		ctx, owner, method := primitiveBridgeCostFixture(t, padding)
		return testing.AllocsPerRun(3, func() {
			if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, ctx); admitted || plan != nil {
				t.Fatal("void method without declaration parameters acquired an erased bridge")
			}
		})
	}
	sparse, padded := measure(32), measure(544)
	t.Logf("allocations/query: sparse32=%.0f padded544=%.0f", sparse, padded)
	// An impossible descriptor family should not allocate work proportional to
	// unrelated declarations. Keep the existing source-query ratio-six budget.
	budget := sparse
	if budget < 1 {
		budget = 1
	}
	if padded > 6*budget {
		t.Fatalf("unrelated source declarations scale negative bridge work: %.0f > 6*%.0f allocations/query", padded, budget)
	}
}

func BenchmarkPrimitiveBridgeNegativePlanner(b *testing.B) {
	for _, padding := range []int{32, 544} {
		b.Run(fmt.Sprintf("Unrelated-%d", padding), func(b *testing.B) {
			ctx, owner, method := primitiveBridgeCostFixture(b, padding)
			b.ReportAllocs()
			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, ctx); admitted || plan != nil {
					b.Fatal("impossible erased bridge")
				}
			}
			b.StopTimer()
		})
	}
}

func TestPrimitiveBridgeNegativePlannerKeepsScalarResults(t *testing.T) {
	for _, tc := range []struct{ result, body string }{
		{"void", ""}, {"boolean", "return false;"}, {"byte", "return 1;"}, {"short", "return 1;"}, {"char", "return 'a';"},
		{"int", "return 1;"}, {"long", "return 1L;"}, {"float", "return 1F;"}, {"double", "return 1D;"},
	} {
		t.Run(tc.result, func(t *testing.T) {
			helper := setupParseHelper(t, "class PrimitiveOwner { "+tc.result+" act() {"+tc.body+"} } class PrimitiveChild extends PrimitiveOwner { "+tc.result+" act() {"+tc.body+"} }")
			owner := overrideBridgeTestScope(t, helper, "PrimitiveOwner")
			method := overrideBridgeTestMethod(t, owner, "act")
			if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, helper.Ctx); admitted || plan != nil {
				t.Fatalf("%s acquired an erased bridge: %#v", tc.result, plan)
			}
		})
	}
}

func TestPrimitiveBridgeNegativePlannerPreservesGenericParameterBridges(t *testing.T) {
	for _, result := range []string{"void", "int"} {
		t.Run(result, func(t *testing.T) {
			body := ""
			if result == "int" {
				body = "return 7;"
			}
			helper := setupParseHelper(t, `class ParameterBridgeProbe {
			interface Numbered { int number(); }
			static final class First implements Numbered { public int number() { return 1; } }
			static class Base<T extends Numbered> { `+result+` act(T value) { `+body+` } }
			static final class Child extends Base<First> { @Override `+result+` act(First value) { `+body+` } }
			}`)
			base := overrideBridgeTestScope(t, helper, "Base")
			baseMethod := overrideBridgeTestMethod(t, base, "act")
			plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(base, baseMethod, helper.Ctx)
			if !admitted || plan == nil || len(plan.overrides) != 1 || !plan.overrides[0].parameters[0].requiresCast {
				t.Fatalf("generic %s parameter bridge lost: %#v/%t", result, plan, admitted)
			}
			child := overrideBridgeTestScope(t, helper, "Child")
			if len(child.TypeParameters) != 0 {
				t.Fatal("control child unexpectedly generic")
			}
			selection, selected := directOwnerSpecializedOverrideBridgeForMethod(child, overrideBridgeTestMethod(t, child, "act"), helper.Ctx)
			if !selected || selection.family == nil || selection.family.owner != base || !selection.bridge.parameters[0].requiresCast {
				t.Fatalf("nongeneric child lost generic ancestor bridge: %#v/%t", selection, selected)
			}
		})
	}
}
