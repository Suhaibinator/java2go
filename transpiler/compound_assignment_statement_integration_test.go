package transpiler

import (
	"strings"
	"testing"
)

func TestSimpleLocalNumericCompoundAssignmentsUseStatementFastPath(t *testing.T) {
	src := `
public class CompoundAssignmentStatementProgram {
    public static String run() {
        int intValue = 2147483647;
        int intStep = 1;
        intValue += intStep;

        long longValue = 9223372036854775807L;
        long longStep = 1L;
        longValue += longStep;

        short shortValue = 32767;
        short shortStep = 1;
        shortValue += shortStep;

        byte byteValue = 127;
        byte byteStep = 1;
        byteValue += byteStep;

        char charValue = (char) 65535;
        char charStep = (char) 1;
        charValue += charStep;

        float floatValue = 1.5F;
        float floatStep = 0.25F;
        floatValue += floatStep;

        double doubleValue = 2.5D;
        double doubleStep = 1.25D;
        doubleValue += doubleStep;

        byte narrowedByte = 120;
        narrowedByte += 10;
        short narrowedShort = 32760;
        narrowedShort += 10;
        char narrowedChar = (char) 65535;
        narrowedChar += 1;

        int shifted = 1;
        int shiftDistance = 32;
        shifted <<= shiftDistance;

        int loopSum = 0;
        int loopStep = 2;
        for (int index = 0; index < 6; index += loopStep) {
            loopSum += index;
        }

        return intValue + ":" + longValue + ":" + shortValue + ":" + byteValue
                + ":" + (int) charValue + ":" + floatValue + ":" + doubleValue
                + ":" + narrowedByte + ":" + narrowedShort + ":" + (int) narrowedChar
                + ":" + shifted + ":" + loopSum;
    }
}
`

	out := renderGoFileFromJava(t, src)
	for _, direct := range []string{
		"intValue += intStep",
		"longValue += longStep",
		"shortValue += shortStep",
		"byteValue += byteStep",
		"floatValue += floatStep",
		"doubleValue += doubleStep",
		"index += loopStep",
		"loopSum += index",
	} {
		if !strings.Contains(out, direct) {
			t.Fatalf("expected direct statement %q, got:\n%s", direct, out)
		}
	}
	for _, narrowed := range []string{
		"charValue = rune(uint16(",
		"narrowedByte = int8(",
		"narrowedShort = int16(",
		"narrowedChar = rune(uint16(",
		"shifted = int32(",
	} {
		if !strings.Contains(out, narrowed) {
			t.Fatalf("expected direct typed assignment containing %q, got:\n%s", narrowed, out)
		}
	}
	if strings.Contains(out, "func(dst *") {
		t.Fatalf("safe local numeric statement assignments must not use assignment-value IIFEs, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestCompoundAssignmentStatementRuntime(t *testing.T) {
    const want = "-2147483648:-9223372036854775808:-32768:-128:0:1.75:3.75:-126:-32766:0:1:6"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}

func TestCompoundAssignmentStatementFastPathFallsBackForObservableCases(t *testing.T) {
	src := `
public class CompoundAssignmentFallbackProgram {
    private int field = 1;
    private static int calls;

    private static int next() {
        calls++;
        return 3;
    }

    private static int boxedAdd(Integer boxed) {
        boxed += 2;
        return boxed;
    }

    public static void nullableAdd() {
        Integer nullable = null;
        nullable += 1;
    }

    public static String run() {
        int staged = 1;
        staged += (staged = 5);

        int sideEffect = 2;
        sideEffect += next();

        int[] values = new int[] { 10, 20 };
        int index = 0;
        values[index++] += next();

        String text = "value=";
        text += 7;

        CompoundAssignmentFallbackProgram holder = new CompoundAssignmentFallbackProgram();
        holder.field += 2;

        int expressionBase = 8;
        int expressionResult = (expressionBase += 1);

        return staged + ":" + sideEffect + ":" + values[0] + ":" + index + ":" + calls
                + ":" + text + ":" + boxedAdd(4) + ":" + holder.field
                + ":" + expressionBase + ":" + expressionResult;
    }
}
`

	out := renderGoFileFromJava(t, src)
	for _, forbidden := range []string{
		"staged +=",
		"sideEffect +=",
		"values[int(index)] +=",
		"text +=",
		"boxed +=",
		"holder.field +=",
		"nullable +=",
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("observable compound assignment %q must retain the fallback, got:\n%s", forbidden, out)
		}
	}
	if count := strings.Count(out, "func(dst *"); count < 8 {
		t.Fatalf("expected assignment-value fallbacks for effectful/complex/value-position cases, got %d:\n%s", count, out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestCompoundAssignmentFallbackRuntime(t *testing.T) {
    const want = "6:5:13:1:2:value=7:6:3:9:9"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}

func TestNullableCompoundAssignmentPanics(t *testing.T) {
    defer func() {
        if recover() == nil {
            t.Fatal("NullableAdd() did not panic")
        }
    }()
    NullableAdd()
}
`)
}
