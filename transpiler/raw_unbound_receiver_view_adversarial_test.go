package transpiler

import "testing"

// Two member classes with the same Java simple name must be resolved through
// their complete owner path. Falling back from LeftOuter.Node or
// RightOuter.Node to the first unqualified Node silently selects the wrong
// erased receiver descriptor and, for these distinct bounds, the wrong method.
func TestRawUnboundReceiverViewAdversarial_QualifiedNestedOwnersRemainDistinct(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
interface RawUnboundQualificationLeftBound { int leftNumber(); }
interface RawUnboundQualificationRightBound { int rightNumber(); }

final class RawUnboundQualificationLeftItem implements RawUnboundQualificationLeftBound {
    final int value;
    RawUnboundQualificationLeftItem(int value) { this.value = value; }
    public int leftNumber() { return value; }
}

final class RawUnboundQualificationRightItem implements RawUnboundQualificationRightBound {
    final int value;
    RawUnboundQualificationRightItem(int value) { this.value = value; }
    public int rightNumber() { return value; }
}

class RawUnboundQualificationLeftOuter<T extends RawUnboundQualificationLeftBound> {
    T outer;
    RawUnboundQualificationLeftOuter(T outer) { this.outer = outer; }

    class Node<U extends RawUnboundQualificationLeftBound> {
        U inner;
        Node(U inner) { this.inner = inner; }

        int mutate(T nextOuter, U nextInner) {
            outer = nextOuter;
            inner = nextInner;
            return outer.leftNumber() * 10 + inner.leftNumber();
        }
    }
}

class RawUnboundQualificationRightOuter<T extends RawUnboundQualificationRightBound> {
    T outer;
    RawUnboundQualificationRightOuter(T outer) { this.outer = outer; }

    class Node<U extends RawUnboundQualificationRightBound> {
        U inner;
        Node(U inner) { this.inner = inner; }

        int mutate(T nextOuter, U nextInner) {
            outer = nextOuter;
            inner = nextInner;
            return outer.rightNumber() * 10 + inner.rightNumber();
        }
    }
}

public class RawUnboundQualifiedNestedIdentityProbe {

    interface LeftOperation {
        int apply(RawUnboundQualificationLeftOuter.Node receiver,
                  RawUnboundQualificationLeftBound nextOuter,
                  RawUnboundQualificationLeftBound nextInner);
    }

    interface RightOperation {
        int apply(RawUnboundQualificationRightOuter.Node receiver,
                  RawUnboundQualificationRightBound nextOuter,
                  RawUnboundQualificationRightBound nextInner);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        RawUnboundQualificationLeftOuter<RawUnboundQualificationLeftItem> leftOuter =
                new RawUnboundQualificationLeftOuter<RawUnboundQualificationLeftItem>(
                        new RawUnboundQualificationLeftItem(1));
        RawUnboundQualificationLeftOuter<RawUnboundQualificationLeftItem>.Node<RawUnboundQualificationLeftItem> leftNode =
                leftOuter.new Node<RawUnboundQualificationLeftItem>(
                        new RawUnboundQualificationLeftItem(2));
        RawUnboundQualificationRightOuter<RawUnboundQualificationRightItem> rightOuter =
                new RawUnboundQualificationRightOuter<RawUnboundQualificationRightItem>(
                        new RawUnboundQualificationRightItem(5));
        RawUnboundQualificationRightOuter<RawUnboundQualificationRightItem>.Node<RawUnboundQualificationRightItem> rightNode =
                rightOuter.new Node<RawUnboundQualificationRightItem>(
                        new RawUnboundQualificationRightItem(6));

        LeftOperation left = RawUnboundQualificationLeftOuter.Node::mutate;
        RightOperation right = RawUnboundQualificationRightOuter.Node::mutate;
        int leftResult = left.apply(leftNode,
                new RawUnboundQualificationLeftItem(3),
                new RawUnboundQualificationLeftItem(4));
        int rightResult = right.apply(rightNode,
                new RawUnboundQualificationRightItem(7),
                new RawUnboundQualificationRightItem(8));
        return leftResult + ":" + rightResult;
    }
}
`, "34:78")
}

// A raw receiver slot has one erased JVM descriptor. The same method-reference
// object can therefore receive incompatible Go generic instantiations, pollute
// both objects, and defer each concrete failure until a parameterized read.
func TestRawUnboundReceiverViewAdversarial_OneReferenceAcceptsInvariantInstantiations(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawUnboundInvariantInstancesProbe {
    interface Numbered { int number(); }

    static final class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static final class Second implements Numbered {
        final int value;
        Second(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Box<T extends Numbered> {
        T value;
        Box(T value) { this.value = value; }

        T swap(T next) {
            T previous = value;
            value = next;
            return previous;
        }
    }

    interface RawOperation {
        Numbered apply(Box receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box<First> first = new Box<First>(new First(1));
        Box<Second> second = new Box<Second>(new Second(3));
        RawOperation operation = Box::swap;

        Numbered firstPrevious = operation.apply(first, new Second(2));
        Numbered secondPrevious = operation.apply(second, new First(4));
        Box firstRaw = first;
        Box secondRaw = second;

        String firstTypedRead;
        try {
            First polluted = first.value;
            firstTypedRead = "missing:" + polluted.number();
        } catch (ClassCastException expected) {
            firstTypedRead = "ClassCastException";
        }
        String secondTypedRead;
        try {
            Second polluted = second.value;
            secondTypedRead = "missing:" + polluted.number();
        } catch (ClassCastException expected) {
            secondTypedRead = "ClassCastException";
        }

        return firstPrevious.number() + ":" + secondPrevious.number() + ":" +
                ((Numbered) firstRaw.value).number() + ":" +
                ((Numbered) secondRaw.value).number() + ":" +
                firstTypedRead + ":" + secondTypedRead;
    }
}
`, "1:3:2:4:ClassCastException:ClassCastException")
}

// An unbound reference captures no receiver and performs no creation-time null
// check. At each invocation Java evaluates the receiver argument once, then all
// remaining arguments, and only then fails while dispatching through a typed
// null receiver. The target body must not run on that failing call.
func TestRawUnboundReceiverViewAdversarial_ReceiverAndArgumentsPrecedeTypedNilFailure(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawUnboundReceiverTimingProbe {
    interface Numbered { int number(); }

    static final class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static String trace = "";

    static class Box<T extends Numbered> {
        T value;
        Box(T value) { this.value = value; }

        T swap(T next) {
            trace += "M";
            T previous = value;
            value = next;
            return previous;
        }
    }

    interface RawOperation {
        Numbered apply(Box receiver, Numbered next);
    }

    static Box<First> receiver(Box<First> value) {
        trace += "R";
        return value;
    }

    static Numbered argument(String marker, int value) {
        trace += marker;
        return new First(value);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        trace = "C";
        RawOperation operation = Box::swap;
        String creationTrace = trace;
        trace = "";

        Box<First> box = new Box<First>(new First(1));
        Numbered previous = operation.apply(receiver(box), argument("A", 2));

        String nullOutcome;
        try {
            operation.apply(receiver(null), argument("B", 3));
            nullOutcome = "missing";
        } catch (NullPointerException expected) {
            nullOutcome = "NullPointerException";
        }

        return creationTrace + ":" + previous.number() + ":" +
                box.value.number() + ":" + nullOutcome + ":" + trace;
    }
}
`, "C:1:2:NullPointerException:RAMRB")
}

// A type-qualified reference is classified as static before Java considers an
// unbound instance target. Here Box is the static method's explicit parameter,
// so rewriting the SAM slot to the method-only erased receiver view would
// corrupt an otherwise ordinary raw argument.
func TestRawUnboundReceiverViewAdversarial_StaticReferenceKeepsExplicitRawParameter(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawUnboundStaticClassificationProbe {
    interface Numbered { int number(); }

    static final class Item implements Numbered {
        final int value;
        Item(int value) { this.value = value; }
        public int number() { return value; }
    }

    static class Box<T extends Numbered> {
        T value;
        Box(T value) { this.value = value; }

        T swap(T next) {
            T previous = value;
            value = next;
            return previous;
        }

        static Numbered id(Box receiver) {
            return (Numbered) receiver.value;
        }
    }

    interface RawOperation {
        Numbered apply(Box receiver);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Box box = new Box(new Item(7));
        RawOperation operation = Box::id;
        return String.valueOf(operation.apply(box).number());
    }
}
`, "7")
}
