package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestRawGenericStaticParameterAndInheritedInnerTypeRuntime(t *testing.T) {
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previousGlobal })

	src := `
public class RawGenericProgram<T> {
    private Node head;

    public class Node {
        T value;
        Node next;
        Node(T value) { this.value = value; }
    }

    RawGenericProgram(T value) {
        this.head = new Node(value);
    }

    T get() {
        if (this.head != null) {
            return this.head.value;
        }
        return null;
    }

    String render() {
        String result = "value=";
        result += this.get();
        return result;
    }

    static void assertValue(RawGenericProgram list, int expected) {
        if ((Integer) list.get() != expected) {
            throw new AssertionError("wrong value");
        }
    }

    public static String run() {
        RawGenericProgram<Integer> list = new RawGenericProgram<>(7);
        assertValue(list, 7);
        return list.render();
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, expected := range []string{
		"head *RawGenericProgramNode[T]",
		"next *RawGenericProgramNode[T]",
		"return *new(T)",
		"func assertValue[ListT any](list *RawGenericProgram[ListT], expected int32)",
		"any(list.get()).(int32)",
		"stdjava.StringValueOf(rhs)",
	} {
		if !strings.Contains(flat, expected) {
			t.Fatalf("expected generated generic program to contain %q, got:\n%s", expected, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestRawGenericRuntime(t *testing.T) {
    if got := Run(); got != "value=7" {
        t.Fatalf("Run() = %q, want value=7", got)
    }
}
`)
}
