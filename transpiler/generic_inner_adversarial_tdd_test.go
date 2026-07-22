package transpiler

import "testing"

// A raw generic outer and its raw member class retain Java's erased, mutable
// field semantics after construction. In particular, both fields may change
// runtime type before the inner method reads them.
func TestGenericInnerAdversarial_RawOuterAndInnerPermitErasedMutation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawMutationProbe {
    static class Outer<T> {
        T outer;

        Outer(T outer) {
            this.outer = outer;
        }

        class Inner<U> {
            U inner;

            Inner(U inner) {
                this.inner = inner;
            }

            String render() {
                return outer + ":" + inner;
            }
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer rawOuter = new Outer("initial");
        rawOuter.outer = 42;
        Outer.Inner rawInner = rawOuter.new Inner("initial-inner");
        rawInner.inner = 7;
        return rawInner.render();
    }
}
`, "42:7")
}

// A member class inherited through a generic subclass must use the concrete
// superclass view of the qualifying instance while preserving its own U.
func TestGenericInnerAdversarial_InheritedExplicitCreationUsesSuperclassView(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InheritedExplicitProbe {
    static class Base<T> {
        final T outer;

        Base(T outer) {
            this.outer = outer;
        }

        class Inner<U> {
            final U inner;

            Inner(U inner) {
                this.inner = inner;
            }

            String render() {
                return outer + ":" + inner;
            }
        }
    }

    static class Sub<X> extends Base<X> {
        Sub(X outer) {
            super(outer);
        }
    }

    public static String run() {
        Sub<String> sub = new Sub<String>("left");
        Base<String>.Inner<Integer> value = sub.new Inner<Integer>(7);
        return value.render();
    }
}
`, "left:7")
}

// An unqualified inherited member-class creation inside a concrete subclass
// has an implicit Base<String> enclosing instance, not an erased Base<any>.
func TestGenericInnerAdversarial_InheritedImplicitCreationUsesConcreteSuperclass(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InheritedImplicitProbe {
    static class Base<T> {
        final T outer;

        Base(T outer) {
            this.outer = outer;
        }

        class Inner<U> {
            final U inner;

            Inner(U inner) {
                this.inner = inner;
            }

            String render() {
                return outer + ":" + inner;
            }
        }
    }

    static class ConcreteSub extends Base<String> {
        ConcreteSub(String outer) {
            super(outer);
        }

        String make(int number) {
            Inner<Integer> value = new Inner<Integer>(number);
            return value.render();
        }
    }

    public static String run() {
        return new ConcreteSub("implicit").make(11);
    }
}
`, "implicit:11")
}

// Green guard: all type arguments carried through two enclosing generic
// member classes remain independently ordered at the leaf constructor.
func TestGenericInnerAdversarial_TripleCarriedTypeArgumentsRemainOrdered(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class TripleCarriedProbe {
    static class Outer<A> {
        final A first;

        Outer(A first) {
            this.first = first;
        }

        class Middle<B> {
            final B second;

            Middle(B second) {
                this.second = second;
            }

            class Leaf<C> {
                final C third;

                Leaf(C third) {
                    this.third = third;
                }

                String render() {
                    return first + ":" + second + ":" + third;
                }
            }
        }
    }

    public static String run() {
        Outer<String> outer = new Outer<String>("a");
        Outer<String>.Middle<Integer> middle = outer.new Middle<Integer>(8);
        Outer<String>.Middle<Integer>.Leaf<Long> leaf = middle.new Leaf<Long>(9L);
        return leaf.render();
    }
}
`, "a:8:9")
}
