package transpiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func finalBridgeCostFixture(tb testing.TB, padding int) (Ctx, *symbol.ClassScope, *symbol.Definition) {
	tb.Helper()
	previous := symbol.GlobalScope
	previousIndex, previousImports := activeResolutionFiles, activeImportSourceInventory
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	tb.Cleanup(func() {
		symbol.GlobalScope = previous
		activeResolutionFiles, activeImportSourceInventory = previousIndex, previousImports
	})
	var text strings.Builder
	text.WriteString("final class FinalBridgeCostOwner { FinalBridgeCostOwner act() { return this; } } class FinalPaddingRoot {}\n")
	for index := 0; index < padding; index++ {
		fmt.Fprintf(&text, "class FinalPadding%d extends FinalPaddingRoot { int value; int get() { return value; } }\n", index)
	}
	file := parsing.SourceFile{Name: "FinalBridgeCostOwner.java", Source: []byte(text.String())}
	if err := file.ParseAST(); err != nil {
		tb.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	ResolveFile(file)
	owner := file.Symbols.FindClassScope("FinalBridgeCostOwner")
	if owner == nil || !declarationHasModifier(owner.Class.DeclarationNode, "final") {
		tb.Fatal("missing explicit final source declaration")
	}
	var method *symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate.OriginalName == "act" {
			method = candidate
		}
	}
	if method == nil {
		tb.Fatal("missing reference-return target method")
	}
	ctx := Ctx{currentFile: file.Symbols, currentClass: owner, callableSubclasses: &callableSubclassSourceInventory{}}
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	activeImportSourceInventory = resolvedSourceInventory(ctx)
	return ctx, owner, method
}

func TestFinalBridgeNegativeAllocationBudget(t *testing.T) {
	measure := func(padding int) float64 {
		ctx, owner, method := finalBridgeCostFixture(t, padding)
		return testing.AllocsPerRun(3, func() {
			if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, ctx); admitted || plan != nil {
				t.Fatal("final source declaration acquired an own descendant bridge family")
			}
		})
	}
	sparse, padded := measure(32), measure(544)
	t.Logf("allocations/query: sparse32=%.0f padded544=%.0f", sparse, padded)
	budget := sparse
	if budget < 1 {
		budget = 1
	}
	if padded > 6*budget {
		t.Fatalf("unrelated declarations scale final own-family work: %.0f > 6*%.0f allocations/query", padded, budget)
	}
}

func BenchmarkFinalBridgeNegativePlanner(b *testing.B) {
	for _, padding := range []int{32, 544} {
		b.Run(fmt.Sprintf("Unrelated-%d", padding), func(b *testing.B) {
			ctx, owner, method := finalBridgeCostFixture(b, padding)
			b.ReportAllocs()
			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, ctx); admitted || plan != nil {
					b.Fatal("impossible final own-family bridge")
				}
			}
			b.StopTimer()
		})
	}
}

func TestFinalBridgeOwnFinalDeclarationsStayUnplanned(t *testing.T) {
	for _, source := range []string{
		`final class FinalOwn { FinalOwn act() { return this; } }`,
		`final class FinalOwn<T> { T act(T value) { return value; } }`,
		`class Outer<T> { final class FinalOwn { T act(T value) { return value; } } }`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			owner := overrideBridgeTestScope(t, helper, "FinalOwn")
			if !declarationHasModifier(owner.Class.DeclarationNode, "final") {
				t.Fatal("control is not explicitly final")
			}
			if plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, overrideBridgeTestMethod(t, owner, "act"), helper.Ctx); admitted || plan != nil {
				t.Fatalf("own final family = %#v/%t", plan, admitted)
			}
		})
	}
}

func TestFinalBridgePreservesAncestorCovariance(t *testing.T) {
	helper := setupParseHelper(t, `class CovariantParent { CovariantParent copy() { return this; } } final class CovariantChild extends CovariantParent { @Override CovariantChild copy() { return this; } }`)
	parent := overrideBridgeTestScope(t, helper, "CovariantParent")
	child := overrideBridgeTestScope(t, helper, "CovariantChild")
	plan, admitted := planDirectOwnerCallableOverrideBridgeFamily(parent, overrideBridgeTestMethod(t, parent, "copy"), helper.Ctx)
	if !admitted || plan == nil || len(plan.overrides) != 1 || !plan.overrides[0].result.requiresWidening {
		t.Fatalf("final child lost ancestor covariant bridge: %#v/%t", plan, admitted)
	}
	selection, selected := directOwnerSpecializedOverrideBridgeForMethod(child, overrideBridgeTestMethod(t, child, "copy"), helper.Ctx)
	if !selected || selection.family == nil || selection.family.owner != parent || !selection.bridge.result.requiresWidening {
		t.Fatalf("final child lost ancestor-target selection: %#v/%t", selection, selected)
	}
}

func TestFinalBridgeModifierProofIgnoresMetadataAndText(t *testing.T) {
	for _, prefix := range []string{"/* final */", `@Marker(" final ")`} {
		t.Run(prefix, func(t *testing.T) {
			helper := setupParseHelper(t, `@interface Marker { String value(); } `+prefix+` class MutableParent { MutableParent copy() { return this; } } final class FinalChild extends MutableParent { @Override FinalChild copy() { return this; } }`)
			parent := overrideBridgeTestScope(t, helper, "MutableParent")
			if declarationHasModifier(parent.Class.DeclarationNode, "final") {
				t.Fatal("text inside annotation/comment became a modifier token")
			}
			// Metadata alone is not a declaration certificate. The actual source
			// remains a legal nonfinal class with a covariant final descendant.
			parent.Class.IsFinal = true
			plan, admitted := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(parent, overrideBridgeTestMethod(t, parent, "copy"), helper.Ctx)
			if !admitted || plan == nil || len(plan.overrides) != 1 || !plan.overrides[0].result.requiresWidening {
				t.Fatalf("source covariance lost to metadata/text finalness: %#v/%t", plan, admitted)
			}
		})
	}
}

func TestFinalBridgeGenericFinalRuntimeCarriage(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericFieldErasureProbe {
    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }

        int firstOnly() {
            return value + 100;
        }
    }

    static class Second implements Numbered {
        final int value;

        Second(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static final class Outer<T extends Numbered> {
        T value;

        Outer(T value) {
            this.value = value;
        }

        int readThroughFirstBound() {
            return value.number();
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<First> typed = new Outer<First>(new First(1));
        Outer raw = typed;

        raw.value = new Second(2);
        Numbered rawObserved = (Numbered) raw.value;
        int boundOnly = typed.readThroughFirstBound();

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.value.firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }

        return rawObserved.number() + ":" + boundOnly + ":" + delayedFailure;
    }
}
`, "2:2:ClassCastException")
}

func TestFinalBridgeEnclosingFinalRuntimeCarriage(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DistinctInnerErasureProbe {
    interface OuterBound { int outerNumber(); }
    interface InnerBound { int innerNumber(); }

    static final class OuterItem implements OuterBound {
        final int value;
        OuterItem(int value) { this.value = value; }
        public int outerNumber() { return value; }
    }

    static final class OtherOuterItem implements OuterBound {
        final int value;
        OtherOuterItem(int value) { this.value = value; }
        public int outerNumber() { return value; }
    }

    static final class InnerItem implements InnerBound {
        final int value;
        InnerItem(int value) { this.value = value; }
        public int innerNumber() { return value; }
    }

    static class Outer<T extends OuterBound> {
        T outer;
        Outer(T outer) { this.outer = outer; }

        final class Inner<U extends InnerBound> {
            U inner;
            Inner(U inner) { this.inner = inner; }

            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.outerNumber() * 10 + inner.innerNumber();
            }
        }
    }

    static class Derived<X extends OuterBound> extends Outer<X> {
        Derived(X outer) { super(outer); }

        int construct(X nextOuter, InnerItem initialInner, InnerItem nextInner) {
            Outer<X>.Inner<InnerItem> value = this.new Inner<InnerItem>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    static int mutateRaw(Outer rawOuter) {
        Outer.Inner rawInner = rawOuter.new Inner(new InnerItem(2));
        return rawInner.mutate(new OtherOuterItem(3), new InnerItem(4));
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<OuterItem> typedOuter = new Outer<OuterItem>(new OuterItem(1));
        Outer rawOuter = typedOuter;
        int raw = mutateRaw(rawOuter);
        int observed = ((OuterBound) rawOuter.outer).outerNumber();
        String delayed;
        try {
            OuterItem polluted = typedOuter.outer;
            delayed = "unexpected:" + polluted.outerNumber();
        } catch (ClassCastException expected) {
            delayed = "ClassCastException";
        }
        Derived<OuterItem> derived = new Derived<OuterItem>(new OuterItem(5));
        int exact = derived.construct(new OuterItem(7), new InnerItem(6), new InnerItem(8));
        return raw + ":" + exact + ":" + observed + ":" + delayed;
    }
}
`, "34:78:3:ClassCastException")
}
