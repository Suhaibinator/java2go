package parity.rawinnerunbound;

public class RawInnerUnboundReferenceProbe {
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

    static final class OtherInnerItem implements InnerBound {
        final int value;
        OtherInnerItem(int value) { this.value = value; }
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

    interface RawUnbound {
        int apply(Outer.Inner receiver, OuterBound nextOuter, InnerBound nextInner);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<OuterItem> firstOuter = new Outer<OuterItem>(new OuterItem(1));
        Outer<OuterItem>.Inner<InnerItem> first =
                firstOuter.new Inner<InnerItem>(new InnerItem(2));

        Outer<OtherOuterItem> secondOuter =
                new Outer<OtherOuterItem>(new OtherOuterItem(8));
        Outer<OtherOuterItem>.Inner<OtherInnerItem> second =
                secondOuter.new Inner<OtherInnerItem>(new OtherInnerItem(9));

        RawUnbound operation = Outer.Inner::mutate;
        int firstResult = operation.apply(
                first, new OtherOuterItem(6), new OtherInnerItem(7));
        int secondResult = operation.apply(
                second, new OuterItem(3), new InnerItem(4));

        Outer firstRawOuter = firstOuter;
        Outer.Inner firstRaw = first;
        String firstTypedRead;
        try {
            OuterItem polluted = firstOuter.outer;
            firstTypedRead = "late:" + polluted.outerNumber();
        } catch (ClassCastException expected) {
            firstTypedRead = "ClassCastException";
        }
        String secondTypedRead;
        try {
            OtherOuterItem polluted = secondOuter.outer;
            secondTypedRead = "late:" + polluted.outerNumber();
        } catch (ClassCastException expected) {
            secondTypedRead = "ClassCastException";
        }

        return firstResult + ":" + secondResult + ":" +
                ((OuterBound) firstRawOuter.outer).outerNumber() + ":" +
                ((InnerBound) firstRaw.inner).innerNumber() + ":" +
                firstTypedRead + ":" + secondTypedRead;
    }

    public static void main(String[] args) {
        System.out.println(run());
    }
}
