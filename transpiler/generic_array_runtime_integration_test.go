package transpiler

import (
	"strings"
	"testing"
)

const genericArrayRuntimeJavaSource = `
public class GenericArrayRuntime {
    static class Base {
        int value;
        Base(int value) { this.value = value; }
    }

    static class Child extends Base {
        Child(int value) { super(value); }
    }

    static <T> T first(T[] values) {
        return values[0];
    }

    static <T> T write(T[] values, int index, T value) {
        return values[index] = value;
    }

    static <T> T forwardFirst(T[] values) {
        return first(values);
    }

    static <T> T forwardWrite(T[] values, T value) {
        return write(values, 0, value);
    }

    static <T extends Base> T boundedFirst(T[] values) {
        return values[0];
    }

    public static String run() {
        String[] words = new String[] { "alpha", "beta" };
        String firstWord = forwardFirst(words);
        String assignedWord = forwardWrite(words, "gamma");

        Child[] actual = new Child[] { new Child(7) };
        Base[] view = actual;
        Base before = forwardFirst(view);
        Child bounded = boundedFirst(actual);

        int rejected = 0;
        try {
            forwardWrite(view, new Base(9));
        } catch (ArrayStoreException expected) {
            rejected = 1;
        }

        Base assignedChild = forwardWrite(view, new Child(4));
        return firstWord + ":" + assignedWord + ":" + words[0]
            + ":" + before.value + ":" + bounded.value + ":" + rejected
            + ":" + assignedChild.value + ":" + actual[0].value;
    }
}
`

func TestGenericReferenceArrayRuntimeUsesErasedStaticDescriptors(t *testing.T) {
	out := renderGoFileFromJava(t, genericArrayRuntimeJavaSource)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, `stdjava.TypeID("T")`) {
		t.Fatalf("generic reference-array lowering emitted a bogus nominal descriptor for T:\n%s", out)
	}
	for _, fragment := range []string{
		`stdjava.ReferenceArrayGet[T](values, 0, stdjava.ObjectTypeID)`,
		`stdjava.ReferenceArrayAssign[T](values, index`,
		`stdjava.ReferenceArrayGet[T](values, 0, stdjava.TypeID("GenericArrayRuntime$Base"))`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("expected erased generic-array fragment %q:\n%s", fragment, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestGenericArrayRuntimeParity(t *testing.T) {
    if got := Run(); got != "alpha:gamma:gamma:7:7:1:4:4" {
        t.Fatalf("Run() = %q, want exact generic read/write/covariant-store parity", got)
    }
}
`)
}

const objectArraySourceObjectJavaSource = `
public class ObjectArraySourceObject {
    static class Foo {
        int value;
        Foo(int value) { this.value = value; }
    }

    public static int run() {
        Object[] values = new Object[1];
        values[0] = new Foo(42);
        return ((Foo) values[0]).value;
    }
}
`

func TestObjectArrayAcceptsSourceClassWithoutMatchingClassArrayDeclaration(t *testing.T) {
	out := renderGoFileFromJava(t, objectArraySourceObjectJavaSource)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`stdjava.NewReferenceArrayOf[any](1, stdjava.ObjectTypeID)`,
		`stdjava.RegisterJavaType(stdjava.TypeID("ObjectArraySourceObject$Foo")`,
		`*ObjectArraySourceObjectfoo) JavaDynamicTypeID() stdjava.TypeID`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("Object[] did not instrument its possible source-class value; missing %q:\n%s", fragment, out)
		}
	}
	if strings.Contains(flat, `*stdjava.ObjectInfo`) || strings.Contains(flat, `stdjava.NewGeneratedObjectInfo`) {
		t.Fatalf("leaf Foo should not allocate concrete-hierarchy metadata:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestObjectArraySourceObjectParity(t *testing.T) {
    if got := Run(); got != 42 {
        t.Fatalf("Run() = %d, want 42", got)
    }
}
	`)
}

const genericArrayRankInferenceJavaSource = `
public class GenericArrayRankInference {
    static <T> T[][] echo(T[][] values) {
        return values;
    }

    static <T> T[][] forward(T[][] values) {
        return echo(values);
    }

    static <T> T peel(T[][] values) {
        return values[0][0];
    }

    public static String run() {
        String[][] matrix = new String[][] { { "ranked" } };
        String[][] result = forward(matrix);
        return peel(result) + ":" + result[0][0];
    }
}
`

func TestGenericReferenceArrayInferenceRetainsArrayRank(t *testing.T) {
	out := renderGoFileFromJava(t, genericArrayRankInferenceJavaSource)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, `stdjava.TypeID("T")`) {
		t.Fatalf("ranked generic reference-array lowering emitted TypeID(T):\n%s", out)
	}
	for _, fragment := range []string{
		`echoJava2goExecution[T](__java2goExecution, values)`,
		`forwardJava2goExecution[string](__java2goExecution, matrix)`,
		`peelJava2goExecution[string](__java2goExecution, result)`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("expected ranked generic-array inference fragment %q:\n%s", fragment, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestGenericArrayRankParity(t *testing.T) {
    if got := Run(); got != "ranked:ranked" {
        t.Fatalf("Run() = %q, want ranked:ranked", got)
    }
}
`)
}
