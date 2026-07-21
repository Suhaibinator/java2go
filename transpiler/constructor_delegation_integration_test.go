package transpiler

import (
	"strings"
	"testing"
)

func TestConstructorDelegationMultiHopInitializesOnceAndKeepsMostDerivedDispatch(t *testing.T) {
	src := `
class DelegationBase {
    String baseTrace;

    DelegationBase() {
        baseTrace = "base:" + label();
    }

    String label() { return "base"; }
}

class DelegationChain extends DelegationBase {
    static int initializations;
    String trace = initialize();

    DelegationChain() {
        this((short) 2);
        trace += ":zero";
        return;
    }

    DelegationChain(short value) {
        this(value, true);
        trace += ":short";
    }

    DelegationChain(long value, boolean marker) {
        super();
        trace += ":long:" + label();
    }

    static String initialize() {
        initializations++;
        return "field";
    }

    String label() { return "chain"; }
}

class DelegationLeaf extends DelegationChain {
    DelegationLeaf() { super(); }
    String label() { return "leaf"; }

    String result() {
        return baseTrace + "|" + trace + "|" + initializations;
    }
}

public class ConstructorDelegationProgram {
    public static String run() {
        return new DelegationLeaf().result();
    }
}
`

	out := renderGoFileFromJava(t, src)
	if got := strings.Count(out, "new(delegationChain)"); got != 1 {
		t.Fatalf("DelegationChain allocations = %d, want exactly one terminal allocation:\n%s", got, out)
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestConstructorDelegationRuntime(t *testing.T) {
	const want = "base:leaf|field:long:leaf:short:zero|1"
	if got := Run(); got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

func TestConstructorDelegationResolvesThisAndSuperOverloads(t *testing.T) {
	src := `
class ConstructorChoice {
    String selected;

    ConstructorChoice() { this(null); }
    ConstructorChoice(Object value) { selected = "object"; }
    ConstructorChoice(String value) { selected = "string"; }
}

class ParentChoice {
    String selected;

    ParentChoice(float value) { selected = "float"; }
    ParentChoice(long value) { selected = "long"; }
    ParentChoice(Object value) { selected = "object"; }
    ParentChoice(String value) { selected = "string"; }
}

class WideningChild extends ParentChoice {
    WideningChild(short value) { super(value); }
}

class NullChild extends ParentChoice {
    NullChild() { super(null); }
}

public class ConstructorOverloadProgram {
    public static String run() {
        ConstructorChoice own = new ConstructorChoice();
        WideningChild widened = new WideningChild((short) 3);
        NullChild nullable = new NullChild();
        return own.selected + "|" + widened.selected + "|" + nullable.selected;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestConstructorOverloadRuntime(t *testing.T) {
	const want = "string|long|string"
	if got := Run(); got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

func TestConstructorDelegationPreservesThrowAndReturnCompletion(t *testing.T) {
	src := `
class AbruptChain {
    static int steps;

    AbruptChain() {
        this(true);
        steps += 100;
        return;
    }

    AbruptChain(String marker) {
        this(false);
        steps += 1000;
        return;
    }

    AbruptChain(boolean fail) {
        this(1);
        steps += 10;
        if (fail) {
            throw new IllegalStateException("stop");
        }
    }

    AbruptChain(int marker) {
        steps += 1;
    }

    static String exercise() {
        steps = 0;
        new AbruptChain("ok");
        int returned = steps;

        steps = 0;
        try {
            new AbruptChain();
            return "miss";
        } catch (IllegalStateException expected) {
            return returned + "|caught:" + steps;
        }
    }
}

public class ConstructorAbruptProgram {
    public static String run() { return AbruptChain.exercise(); }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestConstructorAbruptRuntime(t *testing.T) {
	const want = "1011|caught:11"
	if got := Run(); got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

func TestConstructorDelegationThreadsGenericAndEnclosingArguments(t *testing.T) {
	src := `
public class GenericInnerDelegation<T> {
    T seed;

    GenericInnerDelegation(T seed) { this(seed, 1); }
    GenericInnerDelegation(T seed, long marker) { this.seed = seed; }

    class Inner<U> {
        String trace = "field";
        U value;

        Inner(U value) {
            this(value, 1);
            trace += ":delegate";
        }

        Inner(U value, long marker) {
            this.value = value;
            trace += ":target";
        }

        String render() {
            return seed + ":" + trace + ":" + value;
        }
    }

    String exercise(T input) {
        Inner<T> value = new Inner<T>(input);
        return value.render();
    }

    public static String run() {
        return new GenericInnerDelegation<String>("outer").exercise("inner");
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestGenericInnerConstructorDelegationRuntime(t *testing.T) {
	const want = "outer:field:target:delegate:inner"
	if got := Run(); got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}
