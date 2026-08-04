package transpiler

import (
	"strings"
	"testing"
)

// A bound method reference keeps the SAM's parameterized source signature even
// when the target method's physical descriptor uses the class parameter's
// erasure. The generated adapter must therefore accept First, forward it as
// Numbered, and project the erased result back to First.
func TestDirectOwnerCallableErasure_BoundMethodReferenceAdaptsPhysicalABI(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableMethodReferenceProgram {
    interface Numbered {
        int number();
    }

    interface Replacer {
        First replace(First next);
    }

    static class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static class Box<T extends Numbered> {
        T value;

        Box(T value) {
            this.value = value;
        }

        T replace(T next) {
            T previous = value;
            value = next;
            return previous;
        }
    }

    public static String run() {
        Box<First> box = new Box<First>(new First(1));
        Replacer replacer = box::replace;
        First previous = replacer.replace(new First(2));
        return previous.number() + ":" + box.value.number();
    }
}
`, "1:2")
}

// A concrete specialization has a narrower source override and therefore
// requires javac's synthetic bridge. Until explicit bridges are modeled, the
// uniform-descriptor planner must leave this family on the established ABI.
// The full raw runtime call remains future bridge TDD because that established
// ABI cannot yet accept the erased argument at all.
func TestDirectOwnerCallableErasure_SpecializedOverrideRemainsBridgeGated(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class SpecializedOverrideBridgeProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Second implements Numbered {
        public int number() { return 2; }
    }

    static class Base<T extends Numbered> {
        T exchange(T left, T right) {
            return right;
        }
    }

    static class Specialized extends Base<First> {
        @Override
        First exchange(First left, First right) {
            return right;
        }
    }

    public static String run() {
        return "shape";
    }
}
`))

	for _, want := range []string{
		"exchange(left T, right T) T",
		"exchange(left *SpecializedOverrideBridgeProbefirst, right *SpecializedOverrideBridgeProbefirst) *SpecializedOverrideBridgeProbefirst",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("bridge-required family did not retain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "exchange(left SpecializedOverrideBridgeProbenumbered, right SpecializedOverrideBridgeProbenumbered) SpecializedOverrideBridgeProbenumbered") {
		t.Fatalf("bridge-required family was incorrectly collapsed to its erasure:\n%s", out)
	}
}

// Source-view argument conversion remains separate from the erased physical
// method descriptor. A polluted result used as a typed argument runs first,
// then fails its First projection before the target method body starts.
func TestDirectOwnerCallableErasure_TypedArgumentProjectsBeforeTargetBody(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class TypedArgumentProjectionProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Second implements Numbered {
        public int number() { return 2; }
    }

    static int effects;
    static int bodyCalls;

    static class Box<T extends Numbered> {
        T value;

        Box(T value) {
            this.value = value;
        }

        T current() {
            effects = effects * 10 + 3;
            return value;
        }
    }

    static class Sink<T extends Numbered> {
        void accept(T value) {
            bodyCalls++;
            effects = effects * 10 + 7;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        effects = 0;
        bodyCalls = 0;
        Box<First> box = new Box<First>(new First());
        Box raw = box;
        raw.value = new Second();
        Sink<First> typed = new Sink<First>();
        String outcome;
        try {
            typed.accept(box.current());
            outcome = "unexpected";
        } catch (ClassCastException expected) {
            outcome = "ClassCastException";
        }
        return effects + ":" + bodyCalls + ":" + outcome;
    }
}
`, "3:0:ClassCastException")
}

// Generic override bodies operate on the erased descriptor too. Returning an
// already-polluted Second from super.exchange must stay Numbered inside
// Derived<X>; the cast belongs only at an external concrete First use.
func TestDirectOwnerCallableErasure_GenericOverrideDoesNotNarrowInsideBody(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class GenericOverrideErasedBodyProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Second implements Numbered {
        final int value;
        Second(int value) { this.value = value; }
        public int number() { return value; }
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

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }

        @Override
        X exchange(X next) {
            return super.exchange(next);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Base<First> typed = new Derived<First>(new First(1));
        Base raw = typed;
        Numbered first = (Numbered) raw.exchange(new Second(2));
        Numbered second = (Numbered) raw.exchange(new First(3));
        return first.number() + ":" + second.number() + ":" + ((Numbered) raw.value).number();
    }
}
`, "1:2:3")
}

// Raw and exact bound references share the physical Numbered descriptor. The
// raw SAM accepts heterogeneous implementations; the exact SAM casts only the
// completed invocation result, after virtual Derived -> Base dispatch ran.
func TestDirectOwnerCallableErasure_BoundReferencesPreserveRawAndExactViews(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableBoundReferenceProbe {
    interface Numbered { int number(); }

    interface Binary<A extends Numbered> {
        A apply(A left, A right);
    }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Second implements Numbered {
        final int value;
        Second(int value) { this.value = value; }
        public int number() { return value; }
    }

    static int bodies;

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T left, T right) {
            bodies++;
            T previous = value;
            value = right;
            return previous;
        }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }

        @Override
        X exchange(X left, X right) {
            bodies++;
            return super.exchange(left, right);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        bodies = 0;
        Base<First> exact = new Derived<First>(new First(1));
        Base raw = exact;

        Binary<Numbered> rawReference = raw::exchange;
        Numbered previous = rawReference.apply(new First(8), new Second(2));

        Binary<First> exactReference = exact::exchange;
        String outcome;
        try {
            exactReference.apply(new First(9), new First(3));
            outcome = "unexpected";
        } catch (ClassCastException expected) {
            outcome = "ClassCastException";
        }
        return previous.number() + ":" + ((Numbered) raw.value).number() + ":" + bodies + ":" + outcome;
    }
}
`, "1:3:4:ClassCastException")
}

func TestDirectOwnerCallableErasure_GenericBodyRelayKeepsErasedView(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableGenericBodyRelayProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Second implements Numbered {
        public int number() { return 2; }
    }

    static class Box<T extends Numbered> {
        T value;
        Box(T value) { this.value = value; }

        T current() { return value; }

        T put(T next) {
            T previous = value;
            value = next;
            return previous;
        }

        T relay() { return current(); }

        T forward() { return put(current()); }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<First> typed = new Box<First>(new First());
        Box raw = typed;
        raw.value = new Second();
        Numbered relay = typed.relay();
        Numbered forward = typed.forward();
        return relay.number() + ":" + forward.number();
    }
}
`, "2:2")
}

func TestDirectOwnerCallableErasure_InheritedExactResultProjectsOwnerArguments(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableInheritedResultProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
        int firstOnly() { return value + 100; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T current() { return value; }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    public static String run() {
        Derived<First> derived = new Derived<First>(new First(7));
        First assigned = derived.current();
        return assigned.firstOnly() + ":" + derived.current().firstOnly();
    }
}
`, "107:107")
}

// One bridge-requiring sibling makes the owner parameter's complete physical
// representation indivisible. Migrating safe/value while leaving bridge on T
// would make the Base dispatch interface impossible for Specialized to satisfy.
func TestDirectOwnerCallableErasure_SiblingBridgeGatesWholeOwnerParameter(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class ErasedCallableSiblingBridgeGateProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T safe(T next) {
            T previous = value;
            value = next;
            return previous;
        }

        T bridge(T next) { return next; }
    }

    static class Specialized extends Base<First> {
        Specialized(First value) { super(value); }

        @Override
        First bridge(First next) { return next; }
    }

    public static String run() { return "shape"; }
}
`))

	for _, want := range []string{
		"value T",
		"safe(next T) T",
		"bridge(next T) T",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("bridge-gated sibling did not retain %q:\n%s", want, out)
		}
	}
}

// A descendant parameter that is positionally the same slot as Base<T> is
// part of the same representation plan. An unsupported nested Box<X> use in
// Derived therefore prevents Base's field/result from migrating in isolation.
func TestDirectOwnerCallableErasure_MappedDescendantNestedUseGatesInheritedStorage(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class ErasedCallableMappedShapeGateProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Box<V> {
        V value;
        Box(V value) { this.value = value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T current() { return value; }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
        X risky(Box<X> source) { return source.value; }
    }

    public static String run() { return "shape"; }
}
`))

	for _, want := range []string{"value T", "current() T"} {
		if !strings.Contains(out, want) {
			t.Fatalf("mapped unsupported descendant did not retain %q:\n%s", want, out)
		}
	}
}

// Anonymous subclasses are hoisted after ordinary symbol planning. An
// anonymous subclass of Derived<First> is still transitively a subclass of
// Base<T>, so its potential specialized override must gate Base's ABI.
func TestDirectOwnerCallableErasure_TransitiveAnonymousSubclassGatesOwnerABI(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class ErasedCallableAnonymousDescendantGateProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T current() { return value; }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    public static String run() {
        Base<First> value = new Derived<First>(new First()) {
            @Override First current() { return new First(); }
        };
        return "shape";
    }
}
`))

	for _, want := range []string{"value T", "current() T"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transitive anonymous subclass did not retain %q:\n%s", want, out)
		}
	}
}

func TestDirectOwnerCallableErasure_ExplicitLocalTernaryAndParameterReassignment(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableExplicitLocalProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Second implements Numbered {
        final int value;
        Second(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T current() { return value; }

        T relay() {
            T local;
            local = current();
            return local;
        }

        T swap(T left, T right) {
            left = right;
            return left;
        }

        T choose(boolean selected) {
            T local = selected ? value : null;
            return local;
        }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Base<First> typed = new Derived<First>(new First(1));
        Base raw = typed;
        raw.value = new Second(2);
        Numbered relayed = (Numbered) raw.relay();
        First swapped = typed.swap(new First(3), new First(4));
        Numbered missing = typed.choose(false);
        return relayed.number() + ":" + swapped.number();
    }
}
`, "2:4")
}

func TestDirectOwnerCallableErasure_ExplicitOwnerCastUsesErasure(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableOwnerCastProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Second implements Numbered {
        public int number() { return 2; }
    }

    static class Box<T extends Numbered> {
        T cast(Numbered value) { return (T) value; }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<First> typed = new Box<First>();
        Box raw = typed;
        Numbered result = (Numbered) raw.cast(new Second());
        return String.valueOf(result.number());
    }
}
`, "2")
}

func TestDirectOwnerCallableErasure_ExplicitOwnerCastOfNullSucceeds(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableOwnerNullCastProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        public int number() { return 1; }
    }

    static class Box<T extends Numbered> {
        T castNull() { return (T) null; }
    }

    public static String run() {
        Box<First> typed = new Box<First>();
        typed.castNull();
        return "ok";
    }
}
`, "ok")
}

func TestDirectOwnerCallableErasure_BodyTypeArgumentGatesWholePlan(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class ErasedCallableBodyTypeArgumentGateProbe {
    interface Numbered { int number(); }

    static class Holder<V> {
        V value;
        Holder(V value) { this.value = value; }
        V get() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
        T relay(T input) { return new Holder<T>(input).get(); }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    public static String run() { return "shape"; }
}
`))

	for _, want := range []string{"value T", "relay(input T) T"} {
		if !strings.Contains(out, want) {
			t.Fatalf("body type argument did not retain %q:\n%s", want, out)
		}
	}
}

func TestDirectOwnerCallableErasure_SiblingBodyOnlyTypeArgumentGatesWholePlan(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class ErasedCallableSiblingBodyTypeGateProbe {
    interface Numbered { int number(); }

    static class Holder<V> {
        V value;
        Holder(V value) { this.value = value; }
        V get() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T next) {
            T previous = value;
            value = next;
            return previous;
        }

        int helper() {
            return new Holder<T>(value).get().number();
        }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    public static String run() { return "shape"; }
}
`))

	for _, want := range []string{"value T", "exchange(next T) T"} {
		if !strings.Contains(out, want) {
			t.Fatalf("sibling body-only type argument did not retain %q:\n%s", want, out)
		}
	}
}

func TestDirectOwnerCallableErasure_ConstructorBodyTypeStorageGatesWholePlan(t *testing.T) {
	source := `
public class ErasedCallableConstructorBodyGateProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Holder<V> {
        V value;
        Holder(V value) { this.value = value; }
        V get() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;

        Base(T initial) {
            value = initial;
            T copy;
            copy = value;
            Holder<T> holder = new Holder<T>(value);
            value = holder.get();
        }

        T exchange(T next) {
            T previous = value;
            value = next;
            return previous;
        }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }
    }

    public static String run() {
        Derived<First> value = new Derived<First>(new First(1));
        return String.valueOf(value.exchange(new First(2)).number());
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	for _, want := range []string{"value T", "exchange(next T) T"} {
		if !strings.Contains(out, want) {
			t.Fatalf("constructor body storage did not retain %q:\n%s", want, out)
		}
	}
	assertGeneratedLocalConstructorResult(t, source, "1")
}

func TestDirectOwnerCallableErasure_DescendantOnlyMethodMigratesInheritedField(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class ErasedCallableDescendantOnlyProbe {
    interface Numbered { int number(); }

    static class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Second implements Numbered {
        final int value;
        Second(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) { super(value); }

        X exchange(X next) {
            X previous = value;
            value = next;
            return previous;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Derived<First> typed = new Derived<First>(new First(1));
        Derived raw = typed;
        raw.value = new Second(2);
        Numbered previous = (Numbered) raw.exchange(new First(3));
        return previous.number() + ":" + ((Numbered) raw.value).number();
    }
}
`, "2:3")
}
