package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenericInnerRawParameterConstructionUsesErasedCallableView(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawInnerParameterProbe {
    interface Numbered { int number(); }
    interface Mutator { int apply(Item nextOuter, Item nextInner); }

    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }

        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) { this.inner = inner; }

            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }

    static class Derived<T extends Numbered> extends Outer<T> {
        Derived(T outer) { super(outer); }

        int construct(T nextOuter, Item initialInner, Item nextInner) {
            Outer<T>.Inner<Item> value = this.new Inner<Item>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    static int mutateRaw(Outer rawOuter) {
        Outer.Inner rawInner = rawOuter.new Inner(new Item(2));
        return rawInner.mutate(new Item(3), new Item(4));
    }

    public static String run() {
        Outer rawOuter = new Outer(new Item(1));
        int raw = mutateRaw(rawOuter);
        Derived<Item> derived = new Derived<Item>(new Item(1));
        int explicit = derived.construct(new Item(3), new Item(2), new Item(4));
        Outer<Item> exactOuter = new Outer<Item>(new Item(1));
        Outer<Item>.Inner<Item> exactInner = exactOuter.new Inner<Item>(new Item(2));
        Mutator mutator = exactInner::mutate;
        int referenced = mutator.apply(new Item(5), new Item(6));
        return raw + ":" + explicit + ":" + referenced;
    }
}
`, "34:34:56")
}

func TestGenericInnerRawConstructionKeepsDistinctCarriedAndOwnErasures(t *testing.T) {
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

        class Inner<U extends InnerBound> {
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

func TestGenericInnerCarriedTypeExceptionRejectsUnsafeArguments(t *testing.T) {
	tests := []struct {
		name      string
		localType string
	}{
		{name: "nested carried argument", localType: "Outer<Wrapper<X>>.Inner<Item>"},
		{name: "mixed carried and own arguments", localType: "Outer<X>.Inner<X>"},
		{name: "own argument only", localType: "Outer<Item>.Inner<X>"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := fmt.Sprintf(`
public class UnsafeCarriedInnerProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        public int number() { return 1; }
    }
    static final class Wrapper<V> implements Numbered {
        public int number() { return 2; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) { return 1; }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int unsafe(X next) {
            %s value = null;
            return next.number();
        }
    }
    public static String run() { return "shape"; }
}
`, test.localType)
			out := normalizeSpaces(renderGoFileFromJava(t, source))
			if !strings.Contains(out, "unsafe(next X) int32") {
				t.Fatalf("unsafe carried/own member argument migrated callable ABI:\n%s", out)
			}
		})
	}
}

func TestGenericInnerCarriedTypeExceptionRejectsConstructorOuterFormal(t *testing.T) {
	source := `
public class CarriedInnerConstructorFormalProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            U inner;
            Inner(T snapshot, U inner) {
                outer = snapshot;
                this.inner = inner;
            }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int construct(X nextOuter, Item initialInner, Item nextInner) {
            Outer<X>.Inner<Item> value = this.new Inner<Item>(nextOuter, initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }
    public static String run() {
        Derived<Item> value = new Derived<Item>(new Item(1));
        return String.valueOf(value.construct(new Item(3), new Item(2), new Item(4)));
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	if !strings.Contains(out, "construct(nextOuter X,") {
		t.Fatalf("carried constructor formal did not gate callable ABI:\n%s", out)
	}
	assertGeneratedLocalConstructorResult(t, source, "34")
}

func TestGenericInnerCarriedTypeExceptionRejectsSuperclassConstructorConsumption(t *testing.T) {
	source := `
public class CarriedInnerSuperclassConstructorProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Holder<V extends Numbered> {
        V snapshot;
        Holder(V snapshot) { this.snapshot = snapshot; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> extends Holder<T> {
            U inner;
            Inner(U inner) {
                super(outer);
                this.inner = inner;
            }
            int mutate(T nextOuter, U nextInner) {
                snapshot = nextOuter;
                inner = nextInner;
                return snapshot.number() * 10 + inner.number();
            }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int construct(X nextOuter, Item initialInner, Item nextInner) {
            Outer<X>.Inner<Item> value = this.new Inner<Item>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }
    public static String run() {
        Derived<Item> value = new Derived<Item>(new Item(1));
        return String.valueOf(value.construct(new Item(3), new Item(2), new Item(4)));
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	for _, want := range []string{"mutate(nextOuter T, nextInner U) int32", "construct(nextOuter X,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("superclass constructor consumption did not retain %q:\n%s", want, out)
		}
	}
	assertGeneratedLocalConstructorResult(t, source, "34")
}

func TestGenericInnerCarriedTypeExceptionRejectsFieldInitializerConsumption(t *testing.T) {
	source := `
public class CarriedInnerFieldInitializerProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static final class Holder<V extends Numbered> {
        final V snapshot;
        Holder(V snapshot) { this.snapshot = snapshot; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            Object hidden = new Holder<T>(outer);
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int construct(X nextOuter, Item initialInner, Item nextInner) {
            Outer<X>.Inner<Item> value = this.new Inner<Item>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }
    public static String run() {
        Derived<Item> value = new Derived<Item>(new Item(1));
        return String.valueOf(value.construct(new Item(3), new Item(2), new Item(4)));
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	for _, want := range []string{"mutate(nextOuter T, nextInner U) int32", "construct(nextOuter X,"} {
		if !strings.Contains(out, want) {
			t.Fatalf("field initializer consumption did not retain %q:\n%s", want, out)
		}
	}
	assertGeneratedLocalConstructorResult(t, source, "34")
}

func TestGenericInnerCarriedTypeExceptionAllowsDirectErasedFieldInitializer(t *testing.T) {
	source := `
public class DirectCarriedFieldInitializerProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            T copy = outer;
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return copy.number() * 100 + outer.number() * 10 + inner.number();
            }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int construct(X nextOuter, Item initialInner, Item nextInner) {
            Outer<X>.Inner<Item> value = this.new Inner<Item>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }
    public static String run() {
        Derived<Item> value = new Derived<Item>(new Item(1));
        return String.valueOf(value.construct(new Item(3), new Item(2), new Item(4)));
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	if strings.Contains(out, "mutate(nextOuter T, nextInner U) int32") {
		t.Fatalf("direct field initializer unnecessarily gated callable ABI:\n%s", out)
	}
	assertGeneratedLocalConstructorResult(t, source, "134")
}

func TestGenericInnerCarriedTypeExceptionIgnoresStaticInitializerShadow(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class StaticInitializerShadowProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }

        static {
            class Local<T> {
                T value;
            }
            Local<String> local = new Local<String>();
            local.value = "shadow";
        }

        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    static class Derived<X extends Numbered> extends Outer<X> {
        Derived(X value) { super(value); }
        int construct(X nextOuter, Item initialInner, Item nextInner) {
            Outer<X>.Inner<Item> value = this.new Inner<Item>(initialInner);
            return value.mutate(nextOuter, nextInner);
        }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    static int mutateRaw(Outer rawOuter) {
        Outer.Inner rawInner = rawOuter.new Inner(new Item(2));
        return rawInner.mutate(new Item(3), new Item(4));
    }
    public static String run() {
        Outer rawOuter = new Outer(new Item(1));
        Derived<Item> derived = new Derived<Item>(new Item(1));
        return mutateRaw(rawOuter) + ":" +
                derived.construct(new Item(3), new Item(2), new Item(4));
    }
}
`, "34:34")
}

func TestGenericInnerSpecializedOverrideRemainsBridgeGated(t *testing.T) {
	source := `
public class SpecializedInnerOverrideProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) { this.inner = inner; }
            int mutate(T nextOuter, U nextInner) {
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    static final class ExactOuter extends Outer<Item> {
        ExactOuter(Item value) { super(value); }
        class ExactInner extends Inner<Item> {
            ExactInner(Item value) { super(value); }
            @Override int mutate(Item nextOuter, Item nextInner) {
                return nextOuter.number() * 10 + nextInner.number();
            }
        }
    }
    public static String run() {
        ExactOuter outer = new ExactOuter(new Item(1));
        ExactOuter.ExactInner inner = outer.new ExactInner(new Item(2));
        return String.valueOf(inner.mutate(new Item(3), new Item(4)));
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, source))
	if !strings.Contains(out, "mutate(nextOuter T, nextInner U) int32") {
		t.Fatalf("specialized inner override incorrectly migrated bridge ABI:\n%s", out)
	}
}
