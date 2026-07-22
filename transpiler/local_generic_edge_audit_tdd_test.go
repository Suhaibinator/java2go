package transpiler

import "testing"

// A member inner class is inherited by a subclass. An unqualified `new Inner()`
// in a Sub instance method uses that Sub object as the Inner's enclosing Outer
// instance, including when distinct Sub instances exercise the same call site.
func TestLocalGenericEdgeAudit_InheritedMemberInnerUsesSubclassReceiver(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InheritedMemberInnerProgram {
    static class Outer {
        int base;

        Outer(int base) {
            this.base = base;
        }

        class Inner {
            int value() {
                return Outer.this.base * 10 + 3;
            }
        }
    }

    static class Sub extends Outer {
        Sub(int base) {
            super(base);
        }

        int compute() {
            Inner inherited = new Inner();
            return inherited.value();
        }
    }

    public static String run() {
        return new Sub(7).compute() + ":" + new Sub(4).compute();
    }
}
`, "73:43")
}

// Java permits a class, one of its generic methods, and a method-local generic
// class to each declare a type parameter named T. The three declarations are
// distinct: the class field, captured method argument, and local field retain
// their own bounds even after the local class is hoisted for Go code generation.
func TestLocalGenericEdgeAudit_ShadowedTypeParametersKeepDeclarationProvenance(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
interface ShadowClassMark {
    int classCode();
}

interface ShadowMethodMark {
    int methodCode();
}

interface ShadowLocalMark {
    int localCode();
}

public class ShadowedTypeParameterProgram<T extends ShadowClassMark> {

    static class ClassValue implements ShadowClassMark {
        int code;

        ClassValue(int code) {
            this.code = code;
        }

        public int classCode() {
            return code;
        }
    }

    static class MethodValue implements ShadowMethodMark {
        int code;

        MethodValue(int code) {
            this.code = code;
        }

        public int methodCode() {
            return code;
        }
    }

    static class LocalValue implements ShadowLocalMark {
        int code;

        LocalValue(int code) {
            this.code = code;
        }

        public int localCode() {
            return code;
        }
    }

    T classValue;

    ShadowedTypeParameterProgram(T classValue) {
        this.classValue = classValue;
    }

    <T extends ShadowMethodMark> int score(T methodValue, int localCode) {
        class Local<T extends ShadowLocalMark> {
            T localValue;

            Local(T localValue) {
                this.localValue = localValue;
            }

            int read() {
                return classValue.classCode() * 100
                    + methodValue.methodCode() * 10
                    + localValue.localCode();
            }
        }

        return new Local<LocalValue>(new LocalValue(localCode)).read();
    }

    public static String run() {
        return new ShadowedTypeParameterProgram<ClassValue>(new ClassValue(2))
                .score(new MethodValue(3), 4)
            + ":"
            + new ShadowedTypeParameterProgram<ClassValue>(new ClassValue(5))
                .score(new MethodValue(6), 7);
    }
}
`, "234:567")
}

// T's upper bound is the other method type parameter B. Java therefore allows
// a T value to be returned as B and passed directly to a B parameter. A hoisted
// local class must preserve that dependent relation instead of treating B as a
// declaration-only Go constraint.
func TestLocalGenericEdgeAudit_DependentBoundSupportsReturnAndArgumentConversion(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DependentBoundConversionProgram {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int number) {
            this.number = number;
        }

        public int value() {
            return number;
        }
    }

    static <B extends Root, T extends B> int score(T concrete) {
        class Local {
            B widen(T value) {
                return value;
            }

            int accept(B value) {
                return value.value();
            }

            int read() {
                B widened = widen(concrete);
                return accept(concrete) * 10 + widened.value();
            }
        }

        return new Local().read();
    }

    public static String run() {
        return score(new Impl(7)) + ":" + score(new Impl(4));
    }
}
`, "77:44")
}

// Nested member classes supply only their own source-level type arguments: the
// outer T is carried by the enclosing instance. The same program also checks
// inherited-field substitution for a readable wildcard upper bound and Java's
// raw-type erasure, where U extends Root erases to Root rather than an unbound U.
func TestLocalGenericEdgeAudit_InheritedGenericFieldSubstitutionPaths(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InheritedGenericFieldProgram {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int number) {
            this.number = number;
        }

        public int value() {
            return number;
        }
    }

    static class Outer<T> {
        class Parent<U extends Root> {
            U held;

            Parent(U held) {
                this.held = held;
            }
        }

        class Child<V extends Root> extends Parent<V> {
            Child(V held) {
                super(held);
            }
        }

        int read(Child<Impl> child) {
            return child.held.value();
        }
    }

    static class GenericParent<U extends Root> {
        U held;

        GenericParent(U held) {
            this.held = held;
        }
    }

    static class GenericChild<U extends Root> extends GenericParent<U> {
        GenericChild(U held) {
            super(held);
        }
    }

    static int readWildcard(GenericChild<? extends Root> child) {
        return child.held.value();
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    static class RawChild extends GenericParent {
        RawChild(Root held) {
            super(held);
        }
    }

    static int readRaw(RawChild child) {
        return child.held.value();
    }

    public static String run() {
        Outer<String> outer = new Outer<String>();
        Outer<String>.Child<Impl> nested = outer.new Child<Impl>(new Impl(6));
        GenericChild<Impl> wildcard = new GenericChild<Impl>(new Impl(7));
        RawChild raw = new RawChild(new Impl(8));
        return outer.read(nested) + ":" + readWildcard(wildcard) + ":" + readRaw(raw);
    }
}
`, "6:7:8")
}

// Go code generation must retain every Java intersection-bound member, not
// only the first interface in the bound list. The call below deliberately uses
// a method declared solely by Secondary.
func TestLocalGenericEdgeAudit_IntersectionBoundResolvesSecondInterfaceMethod(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class IntersectionSecondBoundProgram {
    interface Primary {
        int primary();
    }

    interface Secondary {
        int secondary();
    }

    static class Both implements Primary, Secondary {
        int first;
        int second;

        Both(int first, int second) {
            this.first = first;
            this.second = second;
        }

        public int primary() {
            return first;
        }

        public int secondary() {
            return second;
        }
    }

    static <T extends Primary & Secondary> int score(T value) {
        class Local {
            int read() {
                return value.secondary();
            }
        }

        return new Local().read();
    }

    public static String run() {
        return score(new Both(2, 9)) + ":" + score(new Both(4, 7));
    }
}
`, "9:7")
}
