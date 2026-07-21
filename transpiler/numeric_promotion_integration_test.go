package transpiler

import (
	"strings"
	"testing"
)

func TestBinaryNumericPromotion_MixedPrimitiveOperandsCompileAndRun(t *testing.T) {
	src := `
public class NumericPromotionProgram {
    public static long combine(int value, int[] values) {
        return value + 1L + values[0] * 31L;
    }

    public static long narrow(byte b, short s, char c, int i, long l) {
        return b + s + c + i + l;
    }

    public static double blend(float f, int i, long l, double d) {
        return f + i + l + d;
    }

    public static boolean less(int i, long l) {
        return i < l;
    }

    public static boolean equal(int i, long l) {
        return i == l;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, expected := range []string{
		"int64(value)",
		"int64(values[0])*int64(31)",
		"float32(i)",
		"float32(l)",
		"float64(",
		"int64(i) < l",
		"int64(i) == l",
	} {
		if !strings.Contains(flat, expected) {
			t.Fatalf("expected Java binary numeric promotion containing %q, got:\n%s", expected, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestNumericPromotionBehavior(t *testing.T) {
    if got := Combine(7, []int32{5}); got != 163 {
        t.Fatalf("Combine() = %d, want 163", got)
    }
    if got := Narrow(byte(1), int16(2), rune(3), int32(4), int64(5)); got != 15 {
        t.Fatalf("Narrow() = %d, want 15", got)
    }
    if got := Blend(float32(1.5), int32(2), int64(3), float64(4.25)); got != 10.75 {
        t.Fatalf("Blend() = %v, want 10.75", got)
    }
    if !Less(4, 5) || Less(6, 5) {
        t.Fatal("Less() did not preserve int/long comparison semantics")
    }
    if !Equal(5, 5) || Equal(4, 5) {
        t.Fatal("Equal() did not preserve int/long equality semantics")
    }
}
`)
}
