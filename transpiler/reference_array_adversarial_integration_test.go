package transpiler

import (
	"fmt"
	"testing"
)

// These tests are adversarial TDD targets for the reference-array ABI. Keep the
// Java programs and parity assertions intact: a fix must preserve Java runtime
// descriptors and evaluation order across every flow, not merely make the
// generated Go compile by falling back to unchecked native slices.

func TestReferenceArrayAdversarial_MultidimensionalRowAliasKeepsOneRepresentation(t *testing.T) {
	src := `
public class ReferenceArrayRows {
    static int trace;

    static class Cell {
        int value;
        Cell(int value) { this.value = value; }
    }

    static int mark(int value) {
        trace = trace * 10 + value;
        return value;
    }

    public static int run() {
        trace = 0;
        Cell[][] grid = new Cell[2][2];
        grid[mark(1)][mark(0)] = new Cell(mark(7));
        Cell[] row = grid[1];
        return trace * 1000 + row.length * 100 + row[0].value;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestMultidimensionalRowAliasParity(t *testing.T) {
    if got := Run(); got != 107207 {
        t.Fatalf("Run() = %d, want 107207 (left-to-right trace 107, row length 2, value 7)", got)
    }
}
`)
}

func TestReferenceArrayAdversarial_ObjectArrayViewRetainsDynamicComponent(t *testing.T) {
	src := `
public class ReferenceArrayObjectView {
    static int trace;

    static class Base {
        int value;
        Base(int value) { this.value = value; }
    }

    static class Child extends Base {
        Child(int value) { super(value); }
    }

    static int index() {
        trace = trace * 10 + 1;
        return 0;
    }

    static Base replacement() {
        trace = trace * 10 + 2;
        return new Base(9);
    }

    public static int run() {
        trace = 0;
        Child[] actual = new Child[] { new Child(4) };
        Object[] view = actual;
        try {
            view[index()] = replacement();
            trace = trace * 10 + 9;
        } catch (ArrayStoreException expected) {
            trace = trace * 10 + 3;
        }
        return trace * 100 + actual[0].value * 10 + view.length;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestObjectArrayViewParity(t *testing.T) {
    if got := Run(); got != 12341 {
        t.Fatalf("Run() = %d, want 12341 (trace 123, rejected store leaves value 4, length 1)", got)
    }
}
`)
}

func TestReferenceArrayAdversarial_CastsAndInstanceofUseJavaDescriptors(t *testing.T) {
	src := `
public class ReferenceArrayDescriptors {
    static class Base { }
    static class Child extends Base { }

    public static int run() {
        Object childArray = new Child[1];
        Object baseArray = new Base[1];
        int bits = 0;

        if (childArray instanceof Child[]) bits |= 1;
        if (childArray instanceof Base[]) bits |= 2;
        if (baseArray instanceof Base[]) bits |= 4;
        if (!(baseArray instanceof Child[])) bits |= 8;

        try {
            Child[] wrong = (Child[]) baseArray;
            wrong[0] = new Child();
            bits |= 512;
        } catch (ClassCastException expected) {
            bits |= 16;
        }

        Base[] baseView = (Base[]) baseArray;
        if (baseView[0] == null) bits |= 32;

        Base[] good = (Base[]) childArray;
        if (good.length == 1) bits |= 64;

        if (childArray instanceof Base[] typed) {
            if (typed.length == 1) bits |= 128;
        }
        if (baseArray instanceof Child[] wrongPattern) {
            bits |= wrongPattern.length == 1 ? 256 : 1024;
        }

        return bits;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestArrayDescriptorParity(t *testing.T) {
    if got := Run(); got != 255 {
        t.Fatalf("Run() descriptor bits = %d, want 255", got)
    }
}
`)
}

const referenceArraySyntheticImplementorSource = `
public class ReferenceArraySyntheticImplementors {
    interface Op {
        int apply(int value);
    }

    public static int lambdaValue() {
        int offset = 3;
        Op[] values = new Op[] { value -> value + offset };
        return values.length * 10 + values[0].apply(4);
    }

    public static int anonymousValue() {
        Op[] values = new Op[] {
            new Op() {
                public int apply(int value) { return value + 5; }
            }
        };
        return values.length * 10 + values[0].apply(4);
    }

    public static int localValue() {
        int offset = 6;
        class LocalOp implements Op {
            public int apply(int value) { return value + offset; }
        }
        Op[] values = new Op[] { new LocalOp() };
        return values.length * 10 + values[0].apply(4);
    }
}
`

func TestReferenceArrayAdversarial_InterfaceArraysAcceptSyntheticImplementors(t *testing.T) {
	tests := []struct {
		name     string
		function string
		want     int
	}{
		{name: "lambda", function: "LambdaValue", want: 17},
		{name: "anonymous", function: "AnonymousValue", want: 19},
		{name: "local_class", function: "LocalValue", want: 20},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, referenceArraySyntheticImplementorSource)
			runGoTestInTempModule(t, out, fmt.Sprintf(`
package main

import "testing"

func TestSyntheticInterfaceArrayParity(t *testing.T) {
    if got := %s(); got != %d {
        t.Fatalf("%s() = %%d, want %d", got)
    }
}
`, test.function, test.want, test.function, test.want))
		})
	}
}
