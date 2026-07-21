package transpiler

import (
	"strings"
	"testing"
)

func TestEnumIntegration_GeneratesEnumHelpers(t *testing.T) {
	src := `
package enums.helpers;
public enum State {
    ON,
    OFF;
    public String label() { return name() + ":" + ordinal(); }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func StateValueOf(name string) *State") {
		t.Fatalf("expected generated valueOf helper, got:\n%s", out)
	}
	if !strings.Contains(flat, "State) Name() string") {
		t.Fatalf("expected name() accessor to be generated, got:\n%s", out)
	}
	if !strings.Contains(flat, "State) String() string") {
		t.Fatalf("expected fmt.Stringer bridge for Java enum text conversion, got:\n%s", out)
	}
	if !strings.Contains(flat, "== nil { return \"null\" }") {
		t.Fatalf("expected nil enum text conversion to match Java null, got:\n%s", out)
	}
	if !strings.Contains(flat, "State) Ordinal() int32") {
		t.Fatalf("expected ordinal() accessor to be generated, got:\n%s", out)
	}
	if !strings.Contains(flat, "State) CompareTo(other *State) int32") {
		t.Fatalf("expected compareTo helper to be generated, got:\n%s", out)
	}
}

func TestEnumIntegration_StringConversionUsesEnumName(t *testing.T) {
	src := `
public enum Day { MON, FRI }
public class EnumText {
    public static void printAll() {
        Day selected = Day.valueOf("FRI");
        System.out.println(selected);
        System.out.println("day=" + selected);
        System.out.println(String.valueOf(selected));
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "func (dy *Day) String() string") {
		t.Fatalf("expected Day to implement fmt.Stringer, got:\n%s", out)
	}
	if !strings.Contains(flat, "return dy.enumName") {
		t.Fatalf("expected enum String method to return its Java name, got:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.StringValueOf(selected)") {
		t.Fatalf("expected String.valueOf(enum) to use Java conversion semantics, got:\n%s", out)
	}
}

func TestEnumIntegration_StringConversionHonorsToStringOverride(t *testing.T) {
	src := `
public enum CustomLabel {
    ALPHA;
    public String toString() { return "custom-" + name(); }
}
public class CustomEnumText {
    public static String run() {
        CustomLabel value = CustomLabel.ALPHA;
        return value + "|" + String.valueOf(value) + "|" + value.toString();
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "func (cl *CustomLabel) String() string") || !strings.Contains(flat, "return cl.ToString()") {
		t.Fatalf("expected fmt.Stringer bridge to delegate to enum toString override, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestCustomEnumTextRuntime(t *testing.T) {
    if got := Run(); got != "custom-ALPHA|custom-ALPHA|custom-ALPHA" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestEnumIntegration_FloatingOutputUsesJavaFormatting(t *testing.T) {
	src := `
public class FloatingText {
    public static void print(double d, float f) {
        System.out.println(d);
        System.out.println("double=" + d);
        System.out.println(f);
        System.out.println(String.valueOf(d));
        System.out.println(Double.toString(d));
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if strings.Count(flat, "stdjava.StringValueOf(d)") < 3 {
		t.Fatalf("expected println, concatenation, and String.valueOf to use Java double formatting, got:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.StringValueOf(f)") {
		t.Fatalf("expected println(float) to use Java float formatting, got:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.DoubleToString(d)") {
		t.Fatalf("expected Double.toString to use Java double formatting, got:\n%s", out)
	}
}

func TestEnumIntegration_ValueOfInvocationRewritesToGeneratedHelper(t *testing.T) {
	src := `
package enums.valueof;
public class UsesValueOf {
    public State parse(String in) { return State.valueOf(in); }
}
public enum State { ON, OFF }
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "return StateValueOf(in)") {
		t.Fatalf("expected valueOf invocation to call generated helper, got:\n%s", out)
	}
}

func TestEnumIntegration_EmbedsInterfacesAndOverrides(t *testing.T) {
	src := `
package enums.overrides;
public interface Flag { boolean isOn(); }
public enum Switch implements Flag {
    ON { public boolean isOn() { return true; } },
    OFF;

    public boolean isOn() { return false; }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type Switch struct { enumName string enumOrdinal int32 Flag }") {
		t.Fatalf("expected enum to embed implemented interfaces, got:\n%s", out)
	}
	if !strings.Contains(flat, "Switch) IsOn() bool") {
		t.Fatalf("expected interface method wrapper on enum, got:\n%s", out)
	}
	if !strings.Contains(flat, "switch sw.enumName") && !strings.Contains(flat, "switch sh.enumName") {
		t.Fatalf("expected wrapper to dispatch based on enum constant name, got:\n%s", out)
	}
	if !strings.Contains(flat, "_Switch_ON_IsOn(") {
		t.Fatalf("expected constant-specific override to be invoked, got:\n%s", out)
	}
	if !strings.Contains(flat, "_Switch_IsOn_default(") {
		t.Fatalf("expected default implementation to be invoked for non-overrides, got:\n%s", out)
	}
}

func TestEnumIntegration_AbstractMethodWrapperPanics(t *testing.T) {
	src := `
package enums.abstracts;
public enum Operation {
    PLUS { public int apply(int x, int y) { return x + y; } },
    MINUS { public int apply(int x, int y) { return x - y; } },
    IDENTITY;
    public abstract int apply(int x, int y);
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "Operation) Apply(x int32, y int32) int32") {
		t.Fatalf("expected abstract method wrapper on enum, got:\n%s", out)
	}
	if !strings.Contains(flat, "_Operation_PLUS_Apply(") || !strings.Contains(flat, "_Operation_MINUS_Apply(") {
		t.Fatalf("expected constant-specific implementations for apply, got:\n%s", out)
	}
	if !strings.Contains(flat, "abstract enum method not implemented") {
		t.Fatalf("expected default branch to panic for missing implementations, got:\n%s", out)
	}
}
