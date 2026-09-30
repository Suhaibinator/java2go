package transpiler

import "testing"

// Java stores a field whose declared type is a class type parameter at that
// parameter's erasure. Heap pollution through a raw alias is therefore visible
// through every parameterized alias of the same object, but javac inserts a
// narrowing cast according to the expression's use. Assignment/conversion to
// Object or the first bound needs no cast, while selecting even a bound method
// directly from `typed.value` operates on the substituted First expression and
// therefore does cast. A bound call inside Outer still operates on erased T.
//
// The counters also pin cast placement relative to side effects: selecting the
// field receiver happens before a failing field-read cast, the generic method
// body completes before a failing result cast, and neither Numbered.number nor
// First.firstOnly is entered when its direct field receiver cast fails.
func TestRawGenericFieldCastTimingAdversarial_ErasedViewsDelayConcreteCast(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericFieldCastTimingProbe {
    static int routeCalls;
    static int receiverSelections;
    static int boundReadCalls;
    static int numberCalls;
    static int firstOnlyCalls;

    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            numberCalls++;
            return value;
        }

        int firstOnly() {
            firstOnlyCalls++;
            return value + 100;
        }
    }

    static class Second implements Numbered {
        final int value;

        Second(int value) {
            this.value = value;
        }

        public int number() {
            numberCalls++;
            return value;
        }
    }

    static class Outer<T extends Numbered> {
        T value;

        Outer(T value) {
            this.value = value;
        }

        T routeThroughLocal() {
            routeCalls++;
            T local = value;
            return local;
        }

        int readThroughBound() {
            boundReadCalls++;
            return value.number();
        }
    }

    static Outer<First> select(Outer<First> value) {
        receiverSelections++;
        return value;
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        routeCalls = 0;
        receiverSelections = 0;
        boundReadCalls = 0;
        numberCalls = 0;
        firstOnlyCalls = 0;

        Outer<First> typed = new Outer<First>(new First(1));
        Outer raw = typed;
        raw.value = new Second(2);

        int ownerBoundCall = typed.readThroughBound();
        Numbered numberedView = typed.value;
        Object objectView = typed.value;
        boolean nonNull = typed.value != null;

        String directBoundMember;
        try {
            directBoundMember = "unexpected:" + select(typed).value.number();
        } catch (ClassCastException expected) {
            directBoundMember = "ClassCastException";
        }

        String exactField;
        try {
            First first = select(typed).value;
            exactField = "unexpected:" + first.number();
        } catch (ClassCastException expected) {
            exactField = "ClassCastException";
        }

        String exactFieldCall;
        try {
            exactFieldCall = "unexpected:" + select(typed).value.firstOnly();
        } catch (ClassCastException expected) {
            exactFieldCall = "ClassCastException";
        }

        Numbered rawRoute = raw.routeThroughLocal();
        Numbered boundRoute = typed.routeThroughLocal();
        Object objectRoute = typed.routeThroughLocal();

        String exactRoute;
        try {
            First first = typed.routeThroughLocal();
            exactRoute = "unexpected:" + first.firstOnly();
        } catch (ClassCastException expected) {
            exactRoute = "ClassCastException";
        }

        return ownerBoundCall
                + ":" + directBoundMember
                + ":" + numberedView.number()
                + ":" + (objectView == numberedView)
                + ":" + nonNull
                + ":" + exactField
                + ":" + exactFieldCall
                + ":" + rawRoute.number()
                + ":" + boundRoute.number()
                + ":" + (objectRoute == rawRoute)
                + ":" + exactRoute
                + ":" + routeCalls
                + ":" + boundReadCalls
                + ":" + receiverSelections
                + ":" + numberCalls
                + ":" + firstOnlyCalls;
    }
}
`, "2:ClassCastException:2:true:true:ClassCastException:ClassCastException:2:2:true:ClassCastException:4:1:3:4:0")
}
