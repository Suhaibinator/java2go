package transpiler

import "testing"

func TestGenericInnerRawBoundMethodReferencePreservesHiddenInstantiation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawInnerMethodReferenceProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    interface ExactOp { int apply(Item outer, Item inner); }
    interface RawOp { int apply(Numbered outer, Numbered inner); }
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
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<Item> typedOuter = new Outer<Item>(new Item(1));
        Outer<Item>.Inner<Item> typedInner = typedOuter.new Inner<Item>(new Item(2));
        ExactOp exact = typedInner::mutate;
        int exactResult = exact.apply(new Item(3), new Item(4));

        Outer rawOuter = typedOuter;
        Outer.Inner rawInner = rawOuter.new Inner(new Item(5));
        RawOp raw = rawInner::mutate;
        int rawResult = raw.apply(new Item(6), new Item(7));
        return exactResult + ":" + rawResult;
    }
}
`, "34:67")
}

func TestGenericInnerRawBoundMethodReferenceStagesReceiverAndNullCheckAtCreation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawInnerMethodReferenceTimingProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    interface RawOp { int apply(Numbered outer, Numbered inner); }
    static int receiverEvaluations;
    static String trace = "";
    static Item argument(String marker, int value) {
        trace += marker;
        return new Item(value);
    }
    static class Outer<T extends Numbered> {
        T outer;
        Outer(T outer) { this.outer = outer; }
        class Inner<U extends Numbered> {
            U inner;
            Inner(U inner) {
                receiverEvaluations++;
                trace += "R";
                this.inner = inner;
            }
            int mutate(T nextOuter, U nextInner) {
                trace += "M";
                outer = nextOuter;
                inner = nextInner;
                return outer.number() * 10 + inner.number();
            }
        }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<Item> first = new Outer<Item>(new Item(1));
        Outer firstRawOuter = first;
        RawOp bound = (firstRawOuter.new Inner(new Item(2)))::mutate;

        Outer<Item> second = new Outer<Item>(new Item(8));

        int result = bound.apply(argument("A", 6), argument("B", 7));
        String nullTiming;
        Outer.Inner nullInner = null;
        try {
            RawOp rejected = nullInner::mutate;
            nullTiming = rejected == null ? "impossible" : "late";
        } catch (NullPointerException expected) {
            nullTiming = "creation";
        }
        String castTiming;
        Object wrongReceiver = new Item(10);
        try {
            RawOp rejected = ((Outer.Inner) wrongReceiver)::mutate;
            castTiming = rejected == null ? "impossible" : "late";
        } catch (ClassCastException expected) {
            castTiming = "creation";
        }
        return result + ":" + first.outer.number() + ":" +
                second.outer.number() + ":" + receiverEvaluations + ":" +
                nullTiming + ":" + castTiming + ":" + trace;
    }
}
`, "67:6:8:1:creation:creation:RABM")
}

func TestGenericInnerRawBoundMethodReferenceKeepsErasedArgumentsAndDelayedCasts(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawInnerMethodReferencePollutionProbe {
    interface Numbered { int number(); }
    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }
    static final class OtherItem implements Numbered {
        final int value;
        OtherItem(int value) { this.value = value; }
        public int number() { return value; }
    }
    interface RawOp { int apply(Numbered outer, Numbered inner); }
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
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<Item> typedOuter = new Outer<Item>(new Item(1));
        Outer<Item>.Inner<Item> typedInner = typedOuter.new Inner<Item>(new Item(2));
        Outer.Inner rawInner = typedInner;
        RawOp raw = rawInner::mutate;
        int result = raw.apply(new OtherItem(6), new OtherItem(7));

        Outer rawOuter = typedOuter;
        int observedOuter = ((Numbered) rawOuter.outer).number();
        int observedInner = ((Numbered) rawInner.inner).number();
        String outerCast;
        try {
            Item pollutedOuter = typedOuter.outer;
            outerCast = "late:" + pollutedOuter.number();
        } catch (ClassCastException expected) {
            outerCast = "ClassCastException";
        }
        String innerCast;
        try {
            Item pollutedInner = typedInner.inner;
            innerCast = "late:" + pollutedInner.number();
        } catch (ClassCastException expected) {
            innerCast = "ClassCastException";
        }
        return result + ":" + observedOuter + ":" + observedInner + ":" +
                outerCast + ":" + innerCast;
    }
}
`, "67:6:7:ClassCastException:ClassCastException")
}
