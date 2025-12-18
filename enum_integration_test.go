package main

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
	if !strings.Contains(flat, "State) Ordinal() int") {
		t.Fatalf("expected ordinal() accessor to be generated, got:\n%s", out)
	}
	if !strings.Contains(flat, "State) CompareTo(other *State) int") {
		t.Fatalf("expected compareTo helper to be generated, got:\n%s", out)
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

	if !strings.Contains(flat, "type Switch struct { Name string Ordinal int Flag }") {
		t.Fatalf("expected enum to embed implemented interfaces, got:\n%s", out)
	}
	if !strings.Contains(flat, "Switch) IsOn() bool") {
		t.Fatalf("expected interface method wrapper on enum, got:\n%s", out)
	}
	if !(strings.Contains(flat, "switch sw.Name") || strings.Contains(flat, "switch sh.Name")) {
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
