package transpiler

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func overrideBridgeTestMethod(t *testing.T, scope *symbol.ClassScope, name string) *symbol.Definition {
	t.Helper()
	if scope == nil {
		t.Fatal("missing class scope")
	}
	for _, method := range scope.Methods {
		if method != nil && !method.Constructor && method.OriginalName == name {
			return method
		}
	}
	t.Fatalf("missing %s.%s method", scope.Class.OriginalName, name)
	return nil
}

func overrideBridgeTestScope(t *testing.T, helper *ParseHelper, name string) *symbol.ClassScope {
	t.Helper()
	scope := helper.File.Symbols.FindClassScope(name)
	if scope == nil {
		t.Fatalf("missing %s class scope", name)
	}
	return scope
}

func TestDirectOwnerOverrideBridgePlanner_SpecializedOverrideSplitsExactAndErasedDescriptors(t *testing.T) {
	helper := setupParseHelper(t, `
public class OverrideBridgePlanProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered {
        public int number() { return 1; }
    }
    static final class Second implements Numbered {
        public int number() { return 2; }
    }
    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T exchange(T next) {
            T previous = value;
            value = next;
            return previous;
        }
    }
    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }
        @Override First exchange(First next) { return super.exchange(next); }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	method := overrideBridgeTestMethod(t, base, "exchange")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(base, method, helper.Ctx)
	if !ok || plan == nil {
		t.Fatal("specialized override family was not planned atomically")
	}
	if !plan.requiresErasedView {
		t.Fatal("concrete Base<First> specialization did not require an erased receiver view")
	}
	if got := stripJavaQualifier(plan.erasedParameters[0]); got != "Numbered" {
		t.Fatalf("erased parameter = %q, want Numbered", got)
	}
	if got := stripJavaQualifier(plan.erasedResult); got != "Numbered" {
		t.Fatalf("erased result = %q, want Numbered", got)
	}
	if len(plan.overrides) != 1 {
		t.Fatalf("bridge overrides = %d, want 1", len(plan.overrides))
	}
	bridge := plan.overrides[0]
	if bridge.owner == nil || bridge.owner.Class.OriginalName != "Specialized" {
		t.Fatalf("bridge owner = %#v, want Specialized", bridge.owner)
	}
	if len(bridge.parameters) != 1 || !bridge.parameters[0].requiresCast ||
		stripJavaQualifier(bridge.parameters[0].overrideJavaType) != "First" {
		t.Fatalf("bridge parameter plan = %#v, want Numbered -> First checkcast", bridge.parameters)
	}
	if !bridge.result.requiresWidening || stripJavaQualifier(bridge.result.overrideJavaType) != "First" {
		t.Fatalf("bridge result plan = %#v, want First -> Numbered widening", bridge.result)
	}
	erasedName := directOwnerOverrideBridgeErasedExecutionName(plan)
	exactName := directOwnerOverrideBridgeExactExecutionName(bridge)
	if erasedName == "" || exactName == "" || erasedName == exactName {
		t.Fatalf("bridge selectors must be non-empty and distinct: erased=%q exact=%q", erasedName, exactName)
	}
}

func TestDirectOwnerOverrideBridgePlanner_InheritedConcreteSpecializationNeedsNoSyntheticOverride(t *testing.T) {
	helper := setupParseHelper(t, `
public class InheritedBridgePlanProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered {
        public int number() { return 1; }
    }
    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T exchange(T next) { T old = value; value = next; return old; }
    }
    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "exchange"),
		helper.Ctx,
	)
	if !ok || plan == nil || !plan.requiresErasedView {
		t.Fatal("inherited concrete specialization did not plan erased physical storage")
	}
	if len(plan.overrides) != 0 {
		t.Fatalf("inherited method received %d unnecessary synthetic override bridges", len(plan.overrides))
	}
}

func TestDirectOwnerOverrideBridgePlanner_PreservesCastOrderAndIgnoresOverload(t *testing.T) {
	helper := setupParseHelper(t, `
public class OrderedBridgePlanProbe {
    interface Numbered { int number(); }
    static class First implements Numbered { public int number() { return 1; } }
    static class Second implements Numbered { public int number() { return 2; } }
    static class Base<T extends Numbered> {
        T exchange(T left, int marker, T right) { return right; }
    }
    static final class Specialized extends Base<First> {
        @Override First exchange(First left, int marker, First right) { return right; }
        First exchange(Second left, int marker, First right) { return right; }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "exchange"),
		helper.Ctx,
	)
	if !ok || plan == nil || len(plan.overrides) != 1 {
		t.Fatalf("ordered bridge plan = %#v, ok=%t", plan, ok)
	}
	got := make([]bool, len(plan.overrides[0].parameters))
	for index, parameter := range plan.overrides[0].parameters {
		got[index] = parameter.requiresCast
	}
	if want := []bool{true, false, true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("parameter checkcast positions = %v, want %v", got, want)
	}
}

func TestDirectOwnerOverrideBridgePlanner_GenericIntermediateUsesOneErasedDescriptor(t *testing.T) {
	helper := setupParseHelper(t, `
public class GenericIntermediateBridgePlanProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered { public int number() { return 1; } }
    static class Base<T extends Numbered> {
        T exchange(T next) { return next; }
    }
    static class Middle<X extends Numbered> extends Base<X> {
        @Override X exchange(X next) { return super.exchange(next); }
    }
    static final class Leaf extends Middle<First> {
        @Override First exchange(First next) { return super.exchange(next); }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "exchange"),
		helper.Ctx,
	)
	if !ok || plan == nil {
		t.Fatal("generic intermediate family was not planned")
	}
	if len(plan.overrides) != 1 || plan.overrides[0].owner.Class.OriginalName != "Leaf" {
		t.Fatalf("bridges = %#v, want only concrete Leaf bridge", plan.overrides)
	}
}

func TestDirectOwnerOverrideBridgePlanner_CovariantResultAloneRequiresBridge(t *testing.T) {
	helper := setupParseHelper(t, `
public class CovariantResultBridgePlanProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered { public int number() { return 1; } }
    static class Base<T extends Numbered> {
        T choose() { return null; }
    }
    static final class Specialized extends Base<Numbered> {
        @Override First choose() { return new First(); }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "choose"),
		helper.Ctx,
	)
	if !ok || plan == nil || len(plan.overrides) != 1 {
		t.Fatalf("covariant-result bridge plan = %#v, ok=%t", plan, ok)
	}
	bridge := plan.overrides[0]
	if len(bridge.parameters) != 0 {
		t.Fatalf("zero-argument covariant bridge received parameter plan %#v", bridge.parameters)
	}
	if !bridge.result.requiresWidening {
		t.Fatal("First covariant result did not retain its Numbered bridge widening")
	}
}

func TestDirectOwnerOverrideBridgePlanner_RejectsAtomicSiblingNestedUse(t *testing.T) {
	helper := setupParseHelper(t, `
public class AtomicBridgeRejectionProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered { public int number() { return 1; } }
    static class Holder<V> { V value; }
    static class Base<T extends Numbered> {
        T exchange(T next) { return next; }
        T unsafe(Holder<T> nested) { return nested.value; }
    }
    static final class Specialized extends Base<First> {
        @Override First exchange(First next) { return next; }
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	if plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "exchange"),
		helper.Ctx,
	); ok || plan != nil {
		t.Fatalf("partial bridge plan escaped despite unsupported sibling: %#v", plan)
	}
}

func TestDirectOwnerOverrideBridgePlanner_RejectsUnsupportedAbstractDescendantAtomically(t *testing.T) {
	helper := setupParseHelper(t, `
public class AbstractBridgeRejectionProbe {
    interface Numbered { int number(); }
    static final class First implements Numbered { public int number() { return 1; } }
    static class Base<T extends Numbered> {
        T exchange(T next) { return next; }
    }
    static abstract class Specialized extends Base<First> {
        Specialized(First value) {}
        @Override abstract First exchange(First next);
    }
}
`)
	base := overrideBridgeTestScope(t, helper, "Base")
	if plan, ok := planDirectOwnerCallableOverrideBridgeFamily(
		base,
		overrideBridgeTestMethod(t, base, "exchange"),
		helper.Ctx,
	); ok || plan != nil {
		t.Fatalf("unsupported abstract descendant received a partial bridge plan: %#v", plan)
	}
}
