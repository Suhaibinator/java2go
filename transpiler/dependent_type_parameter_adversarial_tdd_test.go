package transpiler

import "testing"

// Java infers T through the nested Box<T> argument and then uses T extends B
// to retain the precise inferred return type. The explicit invocation also
// freezes the wider interface view of the same dependent relation.
func TestDependentTypeParameterAdversarial_NestedArgumentInference(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentNestedArgumentProbe {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int n) {
            number = n;
        }

        public int value() {
            return number;
        }
    }

    static class Box<X> {
        X value;

        Box(X value) {
            this.value = value;
        }
    }

    static <B extends Root, T extends B> B first(Box<T> box) {
        return box.value;
    }

    public static String run() {
        Impl precise = first(new Box<Impl>(new Impl(1)));
        Root wide = DependentNestedArgumentProbe.<Root, Impl>first(new Box<Impl>(new Impl(2)));
        return precise.value() + ":" + wide.value();
    }
}
`, "1:2")
}

// A dependent widening nested inside a constructed generic result is legal in
// Java: T extends B permits the T value to initialize Box<B>.
func TestDependentTypeParameterAdversarial_NestedReturnConstruction(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentNestedReturnProbe {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int n) {
            number = n;
        }

        public int value() {
            return number;
        }
    }

    static class Box<X> {
        X value;

        Box(X value) {
            this.value = value;
        }
    }

    static <B extends Root, T extends B> Box<B> wrap(T value) {
        return new Box<B>(value);
    }

    public static String run() {
        Box<Root> boxed = DependentNestedReturnProbe.<Root, Impl>wrap(new Impl(3));
        return "" + boxed.value.value();
    }
}
`, "3")
}

// A transitive dependent chain over concrete class bounds must preserve both
// Java call-site views: the implicit call returns Child, while explicit type
// arguments request the Middle superclass view.
func TestDependentTypeParameterAdversarial_ConcreteExplicitAndImplicitViews(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentConcreteViewsProbe {
    static class Base {
        int number;

        Base(int n) {
            number = n;
        }
    }

    static class Middle extends Base {
        Middle(int n) {
            super(n);
        }

        int middleOnly() {
            return number + 20;
        }
    }

    static class Child extends Middle {
        Child(int n) {
            super(n);
        }

        int childOnly() {
            return number + 10;
        }
    }

    static <A extends Base, B extends A, T extends B> B widen(T value) {
        return value;
    }

    public static String run() {
        Child precise = widen(new Child(4));
        Middle middle = DependentConcreteViewsProbe.<Base, Middle, Child>widen(new Child(6));
        return precise.childOnly() + ":" + middle.middleOnly();
    }
}
`, "14:26")
}
