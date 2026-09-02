package transpiler

import (
	"strings"
	"testing"
)

func TestClassStringConversion_GeneratesFmtStringerBridge(t *testing.T) {
	src := `
public class ClassStringText {
    static class P {
        int value;
        P(int value) { this.value = value; }
        public String toString() { return "P" + value; }
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func (cp *ClassStringTextp) String() string") {
		t.Fatalf("expected generated class to implement fmt.Stringer, got:\n%s", out)
	}
	if !strings.Contains(flat, "return cp.ToStringJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected String bridge to delegate to Java toString, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (cp *ClassStringTextp) StringJava2goExecution(") {
		t.Fatalf("expected execution-aware String bridge, got:\n%s", out)
	}
}

func TestClassStringConversion_RuntimePathsAndExecution(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;

public class ClassStringRuntime {
	interface Marker {}

    static class P {
        int value;
        P(int value) { this.value = value; }
        public String toString() { return "P" + value; }
    }

    static class Base {
        public String toString() { return "base"; }
    }

    static class Child extends Base {
        public String toString() { return "child"; }
    }

    static class Locked {
        public synchronized String toString() {
            synchronized (this) {
                return "locked";
            }
        }
    }

	record Label(int value) {
		public String toString() { return "label-" + value; }
	}

    public static String run() {
		class Local {
			public String toString() { return "local"; }
		}

        P value = new P(3);
        List<P> values = new ArrayList<P>();
        values.add(value);
        values.add(new P(4));

        Base erased = new Child();
        Locked locked = new Locked();
        String lockedText;
        synchronized (locked) {
            lockedText = String.valueOf(locked);
        }
		Marker anonymous = new Marker() {
			public String toString() { return "anonymous"; }
		};
		Local local = new Local();

		return value + "|" + String.valueOf(value) + "|" + values + "|" + erased + "|" + lockedText
				+ "|" + anonymous + "|" + local + "|" + new Label(7);
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "stdjava.StringValueOfExecution(__java2goExecution, locked)") {
		t.Fatalf("expected concrete generated class conversion to preserve the execution token, got:\n%s", out)
	}
	if !strings.Contains(flat, "return ce.Java2goClassStringRuntimebaseSelf.ToStringJava2goExecution(__java2goExecution)") {
		t.Fatalf("expected base String bridge to retain Java virtual dispatch, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestClassStringRuntime(t *testing.T) {
	const want = "P3|P3|[P3, P4]|child|locked|anonymous|local|label-7"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}
