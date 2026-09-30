package transpiler

import (
	"strings"
	"testing"
)

func TestRawUnboundReceiverView_PreservesDynamicSelfDispatch(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawUnboundDynamicDispatchProbe {
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

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T swap(T next) {
            trace += "B";
            T previous = value;
            value = next;
            return previous;
        }
    }

    static class Child<X extends Numbered> extends Base<X> {
        Child(X value) { super(value); }

        @Override
        X swap(X next) {
            trace += "C";
            X previous = value;
            value = next;
            return previous;
        }
    }

    interface RawOperation {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Base<First> value = new Child<First>(new First(1));
        RawOperation operation = Base::swap;
        Numbered previous = operation.apply(value, new Second(2));
        Base raw = value;
        return trace + ":" + previous.number() + ":" +
                ((Numbered) raw.value).number();
	}
}
`, "C:1:2")
}

func TestRawUnboundReceiverView_DoesNotRewriteSourceImplementorSAM(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `
public class RawUnboundSourceImplementorProbe {
    interface Numbered { int number(); }

    static final class Item implements Numbered {
        public int number() { return 1; }
    }

    static class Box<T extends Numbered> {
        T value;
        T swap(T next) {
            T previous = value;
            value = next;
            return previous;
        }
    }

    interface RawOperation {
        Numbered apply(Box receiver, Numbered next);
    }

    static class SourceImplementation implements RawOperation {
        public Numbered apply(Box receiver, Numbered next) {
            return (Numbered) receiver.value;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        RawOperation reference = Box::swap;
        RawOperation lambda = (receiver, next) -> (Numbered) receiver.value;
        return "shape";
    }
}
`))
	if strings.Contains(out, "Apply(receiver Java2goRawReceiverFrom") ||
		strings.Contains(out, "ApplyJava2goExecution(__java2goExecution *stdjava.Execution, receiver Java2goRawReceiverFrom") {
		t.Fatalf("source implementor/lambda SAM was partially rewritten to the method-only receiver view:\n%s", out)
	}
}
