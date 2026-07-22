package transpiler

import (
	"strings"
	"testing"
)

func TestPrimitiveDescriptorCodegenUsesPrecomputedConstants(t *testing.T) {
	src := `
public class PrimitiveDescriptorProgram {
    public static int run(int size) {
        boolean[] booleans = new boolean[size];
        byte[] bytes = new byte[size];
        short[] shorts = new short[size];
        char[] chars = new char[size];
        int[] ints = new int[size];
        long[] longs = new long[size];
        float[] floats = new float[size];
        double[] doubles = new double[size];
        return booleans.length + bytes.length + shorts.length + chars.length
            + ints.length + longs.length + floats.length + doubles.length;
    }
}
`

	out := renderGoFileFromJava(t, src)
	for _, constant := range []string{
		"stdjava.PrimitiveBooleanTypeID",
		"stdjava.PrimitiveByteTypeID",
		"stdjava.PrimitiveShortTypeID",
		"stdjava.PrimitiveCharTypeID",
		"stdjava.PrimitiveIntTypeID",
		"stdjava.PrimitiveLongTypeID",
		"stdjava.PrimitiveFloatTypeID",
		"stdjava.PrimitiveDoubleTypeID",
	} {
		if !strings.Contains(out, constant) {
			t.Fatalf("generated primitive arrays did not use %s:\n%s", constant, out)
		}
	}
	if strings.Contains(out, "stdjava.PrimitiveTypeID(") {
		t.Fatalf("generated primitive descriptor still performs a runtime name lookup:\n%s", out)
	}
}
