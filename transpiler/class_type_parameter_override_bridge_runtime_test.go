package transpiler

import "testing"

// Base::exchange has one erased descriptor. A plain Base accepts raw pollution,
// while a Specialized receiver dynamically selects javac's synthetic bridge.
// Its failing Numbered -> First cast must happen before either source body runs.
func TestDirectOwnerOverrideBridgeRuntime_RawUnboundDispatchAndArgumentCastTiming(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawUnboundBridgeRuntimeProbe {
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

    static int baseBodies;
    static int specializedBodies;

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T next) {
            baseBodies++;
            T previous = value;
            value = next;
            return previous;
        }
    }

    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }

        @Override
        First exchange(First next) {
            specializedBodies++;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        RawUnbound operation = Base::exchange;

        Base<First> plain = new Base<First>(new First(1));
        Numbered plainResult = operation.apply(plain, new Second(2));

        Specialized specialized = new Specialized(new First(3));
        Numbered specializedResult = operation.apply(specialized, new First(4));

        String rejected;
        try {
            operation.apply(specialized, new Second(5));
            rejected = "missing";
        } catch (ClassCastException expected) {
            rejected = "ClassCastException";
        }

        return plainResult.number() + ":" + specializedResult.number() + ":" +
                baseBodies + ":" + specializedBodies + ":" + rejected;
    }
}
`, "1:3:2:1:ClassCastException")
}

// Invocation arguments are evaluated before Java discovers that an unbound
// instance receiver is null. Once the raw entry has both values, however, its
// receiver guard must win over the specialized bridge's erased-argument cast.
func TestDirectOwnerOverrideBridgeRuntime_NullReceiverPrecedesBridgeCast(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class OverrideBridgeNullReceiverProbe {
    interface Numbered { int number(); }

    static final class First implements Numbered {
        public int number() { return 1; }
    }

    static final class Second implements Numbered {
        public int number() { return 2; }
    }

    static int argumentEffects;
    static int baseBodies;
    static int specializedBodies;

    static Numbered wrongArgument() {
        argumentEffects++;
        return new Second();
    }

    static class Base<T extends Numbered> {
        T exchange(T next) {
            baseBodies++;
            return next;
        }
    }

    static final class Specialized extends Base<First> {
        @Override
        First exchange(First next) {
            specializedBodies++;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        RawUnbound operation = Base::exchange;
        Base receiver = null;

        String outcome;
        try {
            operation.apply(receiver, wrongArgument());
            outcome = "missing";
        } catch (NullPointerException expected) {
            outcome = "NullPointerException";
        } catch (ClassCastException wrongPrecedence) {
            outcome = "ClassCastException";
        }

        return outcome + ":" + argumentEffects + ":" +
                baseBodies + ":" + specializedBodies;
    }
}
`, "NullPointerException:1:0:0")
}

// A valid bridge argument reaches the exact override. If the Base subobject was
// already polluted, super.exchange performs its effects and mutation before
// the source-level First result projection fails. Moving that cast into the
// erased entry would incorrectly suppress both bodies.
func TestDirectOwnerOverrideBridgeRuntime_ResultCastOccursAfterBodyEffects(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class OverrideBridgeResultTimingProbe {
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

    static int effects;
    static int baseBodies;
    static int specializedBodies;

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T next) {
            effects = effects * 10 + 2;
            baseBodies++;
            T previous = value;
            value = next;
            return previous;
        }
    }

    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }

        @Override
        First exchange(First next) {
            effects = effects * 10 + 1;
            specializedBodies++;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Specialized specialized = new Specialized(new First(1));
        Base raw = specialized;
        raw.value = new Second(9);
        RawUnbound operation = Base::exchange;

        String outcome;
        try {
            operation.apply(specialized, new First(2));
            outcome = "missing";
        } catch (ClassCastException expected) {
            outcome = "ClassCastException";
        }

        return outcome + ":" + effects + ":" + baseBodies + ":" +
                specializedBodies + ":" + ((Numbered) raw.value).number();
    }
}
`, "ClassCastException:12:1:1:2")
}

// The structural receiver entry, synthetic bridge, exact override, and super
// implementation must all reuse the caller's *Execution. The outer monitor and
// nested synchronized methods are reentrant on the JVM; a fresh generated token
// deadlocks. Run in a goroutine so this regression fails promptly.
func TestDirectOwnerOverrideBridgeRuntime_PreservesExecutionThroughExactAndSuper(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class OverrideBridgeExecutionProbe {
    interface Numbered { int number(); }

    static final class First implements Numbered {
        final int value;
        First(int value) { this.value = value; }
        public int number() { return value; }
    }

    static String trace = "";

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        synchronized T exchange(T next) {
            synchronized (this) {
                trace += "B";
                T previous = value;
                value = next;
                return previous;
            }
        }
    }

    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }

        @Override
        synchronized First exchange(First next) {
            synchronized (this) {
                trace += "S";
                return super.exchange(next);
            }
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Specialized specialized = new Specialized(new First(1));
        RawUnbound operation = Base::exchange;
        Numbered previous;
        synchronized (specialized) {
            previous = operation.apply(specialized, new First(2));
        }
        Base raw = specialized;
        return trace + ":" + previous.number() + ":" +
                ((Numbered) raw.value).number();
    }
}
`)

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"
)

func TestOverrideBridgeExecution(t *testing.T) {
    result := make(chan string, 1)
    go func() {
        result <- Run()
    }()
    select {
    case got := <-result:
        if got != "SB:1:2" {
            t.Fatalf("Run() = %q, want SB:1:2", got)
        }
    case <-time.After(3 * time.Second):
        t.Fatal("raw override bridge lost the caller execution token and deadlocked")
    }
}
`)
}

// A bridge-enabled Base method returning T through try/finally must record the
// abrupt return in its erased physical result type. The finally block still
// runs after the mutation, and only the Specialized source-result projection
// may reject the polluted value.
func TestDirectOwnerOverrideBridgeRuntime_TryFinallyUsesErasedReturnChannel(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class OverrideBridgeTryFinallyProbe {
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

    static int effects;

    static class Base<T extends Numbered> {
        T value;
        Base(T value) { this.value = value; }

        T exchange(T next) {
            effects = effects * 10 + 1;
            try {
                T previous = value;
                value = next;
                return previous;
            } finally {
                effects = effects * 10 + 2;
            }
        }
    }

    static final class Specialized extends Base<First> {
        Specialized(First value) { super(value); }

        @Override
        First exchange(First next) {
            effects = effects * 10 + 3;
            return super.exchange(next);
        }
    }

    interface RawUnbound {
        Numbered apply(Base receiver, Numbered next);
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Specialized specialized = new Specialized(new First(1));
        Base raw = specialized;
        raw.value = new Second(9);
        RawUnbound operation = Base::exchange;

        String outcome;
        try {
            operation.apply(specialized, new First(2));
            outcome = "missing";
        } catch (ClassCastException expected) {
            outcome = "ClassCastException";
        }

        return outcome + ":" + effects + ":" +
                ((Numbered) raw.value).number();
    }
}
`, "ClassCastException:312:2")
}
