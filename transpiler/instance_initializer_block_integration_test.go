package transpiler

import "testing"

// Field initializers and instance-initializer blocks form one textual sequence.
// A constructor that delegates with this(...) must run that sequence only in
// the terminal constructor, after the implicit super() and before either
// constructor body.
func TestInstanceInitializerBlock_TextualOrderAndThisDelegation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InstanceInitializerProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    int first = mark(2, 10);
    { mark(3, 0); first += 1; }
    int second = mark(4, first + 5);
    { mark(5, 0); second += 1; }

    InstanceInitializerProgram(int value) {
        this(value, mark(1, 3));
        mark(7, 0);
    }

    InstanceInitializerProgram(int left, int right) {
        mark(6, 0);
        second += left + right;
    }

    public static String run() {
        effects = 0;
        InstanceInitializerProgram value = new InstanceInitializerProgram(4);
        return effects + ":" + value.first + ":" + value.second;
    }
}
`, "1234567:11:24")
}

// Explicit super arguments are evaluated first. The superclass then completes
// its own field/block phase and delegated constructor bodies before the
// subclass field/block phase begins.
func TestInstanceInitializerBlock_ExplicitSuperAndInheritanceOrder(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
class InstanceInitializerBase {
    int base = InstanceInitializerInheritanceProgram.mark(2, 20);
    { InstanceInitializerInheritanceProgram.mark(3, 0); base += 1; }

    InstanceInitializerBase() {
        InstanceInitializerInheritanceProgram.mark(4, 0);
    }

    InstanceInitializerBase(int ignored) {
        this();
        InstanceInitializerInheritanceProgram.mark(5, 0);
    }
}

public class InstanceInitializerInheritanceProgram extends InstanceInitializerBase {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    int child = mark(6, 30);
    { mark(7, 0); child += base; }

    InstanceInitializerInheritanceProgram() {
        super(mark(1, 0));
        mark(8, 0);
    }

    public static String run() {
        effects = 0;
        InstanceInitializerInheritanceProgram value =
            new InstanceInitializerInheritanceProgram();
        return effects + ":" + value.base + ":" + value.child;
    }
}
`, "12345678:21:51")
}

// An implicit default constructor still performs implicit super() first, then
// the source-ordered field/block phase for each class in the hierarchy.
func TestInstanceInitializerBlock_ImplicitConstructorsRunEveryPhase(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
class InstanceInitializerImplicitBase {
    int base = InstanceInitializerImplicitProgram.mark(1, 4);
    { InstanceInitializerImplicitProgram.mark(2, 0); base += 1; }
}

public class InstanceInitializerImplicitProgram extends InstanceInitializerImplicitBase {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    int child = mark(3, 6);
    { mark(4, 0); child += base; }

    public static String run() {
        effects = 0;
        InstanceInitializerImplicitProgram value =
            new InstanceInitializerImplicitProgram();
        return effects + ":" + value.base + ":" + value.child;
    }
}
`, "1234:5:11")
}

// Abrupt completion of an initializer block prevents every later initializer
// and the constructor body from executing, and the partially initialized
// object is never returned to the caller.
func TestInstanceInitializerBlock_ExceptionStopsRemainingInitialization(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InstanceInitializerExceptionProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    int first = mark(1, 1);
    {
        mark(2, 0);
        if (first == 1) {
            throw new IllegalStateException("boom");
        }
    }
    int never = mark(3, 3);
    { mark(4, 0); }

    InstanceInitializerExceptionProgram() {
        mark(5, 0);
    }

    public static String run() {
        effects = 0;
        int caught = 0;
        try {
            new InstanceInitializerExceptionProgram();
        } catch (IllegalStateException expected) {
            caught = 1;
        }
        return effects + ":" + caught;
    }
}
`, "12:1")
}

// Each initializer block has a fresh lexical scope. Reusing a local name in a
// later block is valid Java and must not turn into a duplicate Go declaration.
func TestInstanceInitializerBlock_MultipleBlocksHaveSeparateLexicalScopes(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InstanceInitializerScopeProgram {
    int value;

    {
        int local = 4;
        value = value + local;
    }

    {
        int local = 7;
        value = value * 10 + local;
    }

    public static String run() {
        return "" + new InstanceInitializerScopeProgram().value;
    }
}
`, "47")
}

// A non-static inner object's enclosing reference is installed before its
// field/block phase, so both explicit Outer.this access and unqualified outer
// method lookup are available during initialization.
func TestInstanceInitializerBlock_InnerClassSeesEnclosingInstance(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class InstanceInitializerOuterProgram {
    static int effects;
    int outer;

    InstanceInitializerOuterProgram(int outer) {
        this.outer = outer;
    }

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    int bump() {
        return outer + 1;
    }

    class Inner {
        int value = mark(1, InstanceInitializerOuterProgram.this.outer);
        { mark(2, 0); value = value + bump(); }

        Inner() {
            mark(3, 0);
        }
    }

    public static String run() {
        InstanceInitializerOuterProgram owner =
            new InstanceInitializerOuterProgram(7);
        effects = 0;
        Inner value = owner.new Inner();
        return effects + ":" + value.value;
    }
}
`, "123:15")
}
