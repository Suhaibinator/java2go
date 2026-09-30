package transpiler

import "testing"

func TestGenericArrayInferenceCombinesAllLowerBounds(t *testing.T) {
	src := `
public class GenericArrayLubProgram {
    static class Base {
        int value;
        Base(int value) { this.value = value; }
    }

    static class Child extends Base {
        Child(int value) { super(value); }
    }

    static class Sibling extends Base {
        Sibling(int value) { super(value); }
    }

    static int effects;

    static Base markedBase(int value) {
        effects = effects * 10 + value;
        return new Base(value);
    }

    static <T> T write(T[] values, T value) {
        values[0] = value;
        return value;
    }

    static <T> T secondArrayFirst(T[] first, T[] second) {
        return second[0];
    }

    public static String run() {
        effects = 0;
        Child[] children = new Child[] { new Child(3) };
        int rejected = 0;
        try {
            Base ignored = write(children, markedBase(7));
        } catch (ArrayStoreException expected) {
            rejected = 1;
        }

        Sibling[] siblings = new Sibling[] { new Sibling(9) };
        Base selected = secondArrayFirst(children, siblings);
        return rejected + ":" + effects + ":" + selected.value
            + ":" + children[0].value;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestGenericArrayLubParity(t *testing.T) {
    if got := Run(); got != "1:7:9:3" {
        t.Fatalf("Run() = %q, want exact generic lower-bound/LUB parity", got)
    }
}
`)
}
