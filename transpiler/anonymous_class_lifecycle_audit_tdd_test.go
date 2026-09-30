package transpiler

import "testing"

// A superclass constructor can initialize one of its own fields before making
// a virtual call. Java dispatches that call to the anonymous override, and the
// override must observe the already-written inherited field through the same
// superclass subobject.
func TestAnonymousClassLifecycleAudit_SuperConstructorOverrideSeesInheritedState(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class LifecycleInheritedBase {
    int inherited;
    int observed;

    LifecycleInheritedBase(int value) {
        inherited = value;
        observed = probe();
    }

    int probe() {
        return -1;
    }
}

public class LifecycleInheritedProgram {
    public static int run() {
        var instance = new LifecycleInheritedBase(7) {
            int probe() {
                return inherited + 2;
            }
        };
        return instance.observed * 10 + instance.inherited;
    }
}
`, 97)
}

// Compiler-synthesized capture fields already contain their values when a
// superclass constructor dispatches into an anonymous override. In contrast,
// declared reference fields still have their Java default value, null.
func TestAnonymousClassLifecycleAudit_StringDefaultAndCaptureDuringSuper(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
abstract class LifecycleStringBase {
    int duringConstruction;

    LifecycleStringBase() {
        duringConstruction = inspect();
    }

    abstract int inspect();
}

public class LifecycleStringProgram {
    public static int run() {
        String captured = "ok";
        var instance = new LifecycleStringBase() {
            String declared;

            int inspect() {
                return (declared == null ? 10 : 90) + captured.length();
            }
        };

        int afterConstruction = instance.inspect();
        return instance.duringConstruction * 100
            + (instance.declared == null ? 10 : 90)
            + afterConstruction;
    }
}
`, 1222)
}

// Field initializers and instance-initializer blocks share one textual order
// after super construction. Keeping only the field declarations silently loses
// observable side effects from the intervening blocks.
func TestAnonymousClassLifecycleAudit_InterleavesFieldsAndInitializerBlocks(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class LifecycleInitializerBlockProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static int run() {
        effects = 0;
        var instance = new Object() {
            int first = mark(1, 1);
            { mark(2, 0); }
            int second = mark(3, 3);
            { mark(4, 0); }
        };

        return effects * 10 + instance.first + instance.second;
    }
}
`, 12344)
}

// An anonymous object is lexically enclosed by the surrounding instance.
// Unqualified outer fields and methods, an explicit Outer.this, and captured
// method locals must all remain available alongside the anonymous state.
func TestAnonymousClassLifecycleAudit_StatefulObjectRetainsEnclosingInstance(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class LifecycleOuterProgram {
    int base;

    LifecycleOuterProgram(int base) {
        this.base = base;
    }

    int bump() {
        return base + 1;
    }

    int evaluate(int seed) {
        var instance = new Object() {
            int state = seed;

            int read() {
                return base * 100
                    + bump() * 10
                    + LifecycleOuterProgram.this.base
                    + state;
            }
        };
        return instance.read();
    }

    public static int run() {
        return new LifecycleOuterProgram(3).evaluate(4);
    }
}
`, 347)
}

// A stateful anonymous interface implementation must inherit executable default
// methods, not a nil embedded interface placeholder. The default method must in
// turn dispatch its abstract call to the anonymous override.
func TestAnonymousClassLifecycleAudit_InterfaceDefaultMethodCarrier(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
interface LifecycleDefaultCarrier {
    int value();

    default int doubled() {
        return value() * 2;
    }
}

public class LifecycleDefaultProgram {
    public static int run() {
        LifecycleDefaultCarrier instance = new LifecycleDefaultCarrier() {
            int base = 6;

            public int value() {
                return base;
            }
        };

        return instance.value() * 10 + instance.doubled();
    }
}
`, 72)
}

// Even with no fields and exactly one SAM method, `this` denotes the anonymous
// object, not the lexically enclosing receiver. A closure adapter cannot model
// this body unless it also gives the anonymous object its own identity.
func TestAnonymousClassLifecycleAudit_SAMThisIsAnonymousReceiver(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
interface LifecycleIdentityProbe {
    int code();
}

public class LifecycleThisProgram implements LifecycleIdentityProbe {
    public int code() {
        return 9;
    }

    int evaluate() {
        LifecycleIdentityProbe probe = new LifecycleIdentityProbe() {
            public int code() {
                return (Object) this == (Object) LifecycleThisProgram.this ? 90 : 7;
            }
        };
        return probe.code();
    }

    public static int run() {
        return new LifecycleThisProgram().evaluate();
    }
}
`, 7)
}
