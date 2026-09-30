package transpiler

import (
	"strings"
	"testing"
)

func TestGeneratedReferenceArrayCovarianceChecksStoresAndReifiesAllArrayDescriptors(t *testing.T) {
	src := `
public class ReferenceArrayProgram {
    private static String trace = "";
	private static String primitiveTrace = "";

    static class Base {
        public int ObjectInfo = 1;
        int value;

        Base(int value) {
            trace = trace + "b";
            this.value = value;
        }

        public int javaObjectInfo() { return 2; }
        public int java2goReferenceDynamicType() { return 3; }
        public int java2goReferenceView() { return 4; }

        int reservedSum() {
            return ObjectInfo + javaObjectInfo()
                + java2goReferenceDynamicType() + java2goReferenceView();
        }
    }

    static class Child extends Base {
        Child(int value) {
            super(value);
            trace = trace + "c";
        }
    }

    static Base[] select(Base[] values) {
        trace = trace + "a";
        return values;
    }

    static int index() {
        trace = trace + "i";
        return 0;
    }

    static Base replacement() {
        trace = trace + "r";
        return new Base(9);
    }

    public static int run() {
        trace = "";
        Base[] view = new Child[] { new Child(4), new Child(5) };
        try {
            select(view)[index()] = replacement();
            trace = trace + "x";
        } catch (ArrayStoreException expected) {
            trace = trace + "e";
        }
        Base assigned = (view[1] = new Child(6));
        return view[0].value * 100 + view[1].value * 10
            + assigned.reservedSum() + view.length;
    }

    public static String traceValue() {
        return trace;
    }

    public static double numeric() {
        double[] values = new double[] { 2.0, 3.0 };
        values[0] = values[0] + 1.0;
        return values[0] + values.length;
    }

    public static int primitiveNested() {
        int[][] matrix = new int[2][2];
        int[] row = matrix[0];
        row[1] = 7;

        Object[] erased = matrix;
        Object first = erased[0];
        int score = first instanceof int[] ? 1 : 0;
        if (first instanceof long[]) score += 10;
        try {
            erased[0] = new long[] { 9L, 10L };
            score += 100;
        } catch (ArrayStoreException expected) {
            score += 2;
        }

        int[] recovered = (int[]) first;
        score += recovered[1] * 10;
        matrix[1] = new int[] { 3, 4 };
        return score + matrix[1][0] * 100;
    }

	static int primitiveRhs() {
		primitiveTrace = primitiveTrace + "r";
		return 8;
	}

	public static String primitiveSimpleAssignmentOrder() {
		primitiveTrace = "";
		int[] missing = null;
		try {
			missing[0] = primitiveRhs();
		} catch (NullPointerException expected) {
			primitiveTrace = primitiveTrace + "n";
		}
		int[] one = new int[1];
		try {
			one[2] = primitiveRhs();
		} catch (ArrayIndexOutOfBoundsException expected) {
			primitiveTrace = primitiveTrace + "b";
		}
		return primitiveTrace;
	}

	public static String primitiveCompoundAssignmentOrder() {
		primitiveTrace = "";
		int[] missing = null;
		try {
			missing[0] += primitiveRhs();
		} catch (NullPointerException expected) {
			primitiveTrace = primitiveTrace + "n";
		}
		int[] one = new int[1];
		try {
			one[2] += primitiveRhs();
		} catch (ArrayIndexOutOfBoundsException expected) {
			primitiveTrace = primitiveTrace + "b";
		}
		return primitiveTrace;
	}
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`view := stdjava.ReferenceArrayLiteralOf[*ReferenceArrayProgramchild](stdjava.TypeID("ReferenceArrayProgram$Child")`,
		`stdjava.ReferenceArrayGet[*ReferenceArrayProgrambase]`,
		`stdjava.ReferenceArrayAssign[*ReferenceArrayProgrambase]`,
		`stdjava.ReferenceArrayLength(view)`,
		`stdjava.RegisterJavaType(stdjava.TypeID("ReferenceArrayProgram$Child"), stdjava.TypeID("ReferenceArrayProgram$Base"))`,
		`*stdjava.ObjectInfo`,
		`Java2goReferenceDynamicType0`,
		`Java2goReferenceView0`,
		`JavaObjectInfo0`,
		`ObjectInfo0 int32`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("expected generated reference-array fragment %q:\n%s", fragment, out)
		}
	}

	numeric := normalizeSpaces(generatedFunctionText(out, "Numeric"))
	if !strings.Contains(numeric, `stdjava.PrimitiveArrayLiteral[float64](stdjava.PrimitiveDoubleTypeID, 2.0, 3.0)`) ||
		!strings.Contains(numeric, "values.Elements[0]") ||
		!strings.Contains(numeric, "stdjava.PrimitiveArrayLength(values)") {
		t.Fatalf("primitive array lost its descriptor-bearing optimized element lowering:\n%s", numeric)
	}
	if strings.Contains(numeric, "ReferenceArrayGet") ||
		strings.Contains(numeric, "ReferenceArrayAssign") ||
		strings.Contains(numeric, "ReferenceArrayLength") {
		t.Fatalf("primitive numeric path unexpectedly uses reified reference-array helpers:\n%s", numeric)
	}
	primitiveNested := normalizeSpaces(generatedFunctionText(out, "PrimitiveNested"))
	for _, fragment := range []string{
		`stdjava.NewMultiArrayOf[int32](stdjava.PrimitiveIntTypeID, 2`,
		`stdjava.ReferenceArrayGet[*stdjava.PrimitiveArray[int32]]`,
		`stdjava.JavaArrayInstanceOf(first, stdjava.ArrayTypeID(stdjava.PrimitiveIntTypeID))`,
		`stdjava.JavaArrayCast[*stdjava.PrimitiveArray[int32]]`,
	} {
		if !strings.Contains(primitiveNested, fragment) {
			t.Fatalf("expected primitive multidimensional descriptor fragment %q:\n%s", fragment, primitiveNested)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestReferenceArrayRuntimeParity(t *testing.T) {
    if got := Run(); got != 472 {
        t.Fatalf("Run() = %d, want 472", got)
    }
    if got := TraceValue(); got != "bcbcairbebc" {
        t.Fatalf("TraceValue() = %q, want exact Java evaluation/store trace", got)
    }
    if got := Numeric(); got != 5.0 {
        t.Fatalf("Numeric() = %v, want 5.0", got)
    }
    if got := PrimitiveNested(); got != 373 {
        t.Fatalf("PrimitiveNested() = %d, want 373", got)
    }
	if got := PrimitiveSimpleAssignmentOrder(); got != "rnrb" {
		t.Fatalf("PrimitiveSimpleAssignmentOrder() = %q, want rnrb", got)
	}
	if got := PrimitiveCompoundAssignmentOrder(); got != "nb" {
		t.Fatalf("PrimitiveCompoundAssignmentOrder() = %q, want nb", got)
	}
}
`)
}

func TestGeneratedLeafReferenceArrayIdentityAvoidsPerObjectMetadataAllocation(t *testing.T) {
	src := `
public class LeafReferenceArrayProgram {
    static class Leaf {
        int value;

        Leaf(int value) {
            this.value = value;
        }
    }

    public static int run() {
        Leaf[] actual = new Leaf[] { new Leaf(3), new Leaf(4) };
        Object[] erased = actual;
        erased[1] = new Leaf(9);
        return actual[0].value * 100 + actual[1].value * 10 + erased.length;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		`stdjava.RegisterJavaType(stdjava.TypeID("LeafReferenceArrayProgram$Leaf"), stdjava.ObjectTypeID)`,
		`JavaDynamicTypeID() stdjava.TypeID`,
		`stdjava.ReferenceArrayLiteralOf[*LeafReferenceArrayProgramleaf](stdjava.TypeID("LeafReferenceArrayProgram$Leaf")`,
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("expected lightweight leaf reference identity fragment %q:\n%s", fragment, out)
		}
	}
	for _, forbidden := range []string{
		`*stdjava.ObjectInfo`,
		`stdjava.NewGeneratedObjectInfo`,
		`Java2goReferenceDynamicType`,
		`Java2goReferenceView`,
		`Java2goWithSelf`,
	} {
		if strings.Contains(flat, forbidden) {
			t.Fatalf("leaf reference identity unexpectedly contains per-object hierarchy machinery %q:\n%s", forbidden, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestLeafReferenceArrayRuntime(t *testing.T) {
    if got := Run(); got != 392 {
        t.Fatalf("Run() = %d, want 392", got)
    }
}
`)
}
