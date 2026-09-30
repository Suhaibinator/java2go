package transpiler

import "testing"

// A generic override preserves the ancestor's erased descriptor. Only the
// concrete Leaf specialization needs a cast-before-body bridge. Calls through
// every source view must still dispatch to Leaf, while a raw call on Middle is
// allowed to pollute its Base storage. A valid raw Leaf argument reaches all
// three bodies and observes a delayed result cast after Base has mutated; an
// invalid argument is rejected by Leaf's bridge before any source body runs.
func TestDirectOwnerOverrideBridgeRuntime_GenericIntermediateRemainsErased(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class OverrideBridgeIntermediateRuntimeProbe {
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

    static String trace = "";
    static int baseBodies;
    static int middleBodies;
    static int leafBodies;

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T next) {
            trace += "B";
            baseBodies++;
            T previous = value;
            value = next;
            return previous;
        }
    }

    static class Middle<X extends Numbered> extends Base<X> {
        Middle(X value) { super(value); }

        @Override
        X exchange(X next) {
            trace += "M";
            middleBodies++;
            return super.exchange(next);
        }
    }

    static final class Leaf extends Middle<First> {
        Leaf(First value) { super(value); }

        @Override
        First exchange(First next) {
            trace += "L";
            leafBodies++;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Leaf leaf = new Leaf(new First(1));
        Base<First> asBase = leaf;
        Middle<First> asMiddle = leaf;
        Leaf asLeaf = leaf;

        First first = asBase.exchange(new First(2));
        First second = asMiddle.exchange(new First(3));
        First third = asLeaf.exchange(new First(4));

        RawUnbound operation = Base::exchange;
        Middle<First> genericMiddle = new Middle<First>(new First(10));
        Numbered middlePrevious = operation.apply(genericMiddle, new Second(11));
        Base rawMiddle = genericMiddle;

        Base rawLeaf = leaf;
        rawLeaf.value = new Second(9);
        String delayedResult;
        try {
            operation.apply(leaf, new First(5));
            delayedResult = "missing";
        } catch (ClassCastException expected) {
            delayedResult = "ClassCastException";
        }

        String rejectedArgument;
        try {
            operation.apply(leaf, new Second(6));
            rejectedArgument = "missing";
        } catch (ClassCastException expected) {
            rejectedArgument = "ClassCastException";
        }

        return first.number() + "," + second.number() + "," + third.number() + ":" +
                middlePrevious.number() + ":" + ((Numbered) rawMiddle.value).number() + ":" +
                delayedResult + ":" + rejectedArgument + ":" + trace + ":" +
                baseBodies + ":" + middleBodies + ":" + leafBodies + ":" +
                ((Numbered) rawLeaf.value).number();
    }
}
`, "1,2,3:10:11:ClassCastException:ClassCastException:LMBLMBLMBMBLMB:5:5:4:5")
}
