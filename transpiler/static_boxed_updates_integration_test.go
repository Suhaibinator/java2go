package transpiler

import "testing"

func TestStaticBoxedUpdatesPreserveAliasesAndExpressionResults(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class StaticBoxedUpdatesProgram {
    static Integer value = 1000;
    static int calls;
    public static Integer current() { return value; }
    public static Integer postIncrement() { return value++; }
    public static Integer preIncrement() { return ++value; }
    public static Integer postDecrement() { return value--; }
    public static Integer preDecrement() { return --value; }
    public static Integer compound() { return value += 4; }
    static int replace() { value = 2000; return 2; }
    public static Integer compoundReplacingRhs() { return value += replace(); }
    static StaticBoxedUpdatesProgram qualifier() { calls++; return null; }
    public static Integer qualifiedPostIncrement() { return qualifier().value++; }
    public static int count() { return calls; }
}
`)
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func TestImmutableStaticUpdates(t *testing.T) {
    original := Current()
    if result := PostIncrement(); result != original || stdjava.UnboxInteger(original) != 1000 {
        t.Fatal("post-increment must return the old, unmodified object")
    }
    if Current() == original || stdjava.UnboxInteger(Current()) != 1001 {
        t.Fatal("post-increment must replace static storage")
    }
    if result := PreIncrement(); result != Current() || stdjava.UnboxInteger(result) != 1002 {
        t.Fatal("pre-increment must return the stored replacement")
    }
    before := Current()
    if result := PostDecrement(); result != before || stdjava.UnboxInteger(before) != 1002 {
        t.Fatal("post-decrement must return the old, unmodified object")
    }
    if result := PreDecrement(); result != Current() || stdjava.UnboxInteger(result) != 1000 {
        t.Fatal("pre-decrement must return the stored replacement")
    }
    before = Current()
    if result := Compound(); result != Current() || result == before || stdjava.UnboxInteger(result) != 1004 {
        t.Fatal("compound assignment must return a replacement object")
    }
    if stdjava.UnboxInteger(before) != 1000 {
        t.Fatal("compound assignment changed an existing alias")
    }
    if result := CompoundReplacingRhs(); stdjava.UnboxInteger(result) != 1006 {
        t.Fatal("compound assignment must capture its old value before evaluating the RHS")
    }
    before = Current()
    if result := QualifiedPostIncrement(); result != before || Count() != 1 || stdjava.UnboxInteger(Current()) != 1007 {
        t.Fatal("a null static-field qualifier must be evaluated exactly once")
    }
}
`)
}

func TestStaticBoxedCompoundUnboxingTiming(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class StaticBoxedNullTimingProgram {
    static Integer value;
    static Boolean flag = false;
    static int calls;
    static int rhs() { calls++; return 4; }
    static Boolean nullFlag() { calls++; return null; }
    public static Integer compound() { return value += rhs(); }
    public static Integer increment() { return value++; }
    public static Boolean booleanCompound() { return flag &= nullFlag(); }
    public static int count() { return calls; }
    public static Integer current() { return value; }
    public static Boolean currentFlag() { return flag; }
}
`)
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func requireNullPointer(t *testing.T, action func()) {
    t.Helper()
    defer func() {
        failure, ok := recover().(stdjava.Throwable)
        if !ok || failure.ThrowableTypeName() != "NullPointerException" {
            t.Fatalf("expected NullPointerException, got %v", failure)
        }
    }()
    action()
}

func TestStaticNullUnboxing(t *testing.T) {
    requireNullPointer(t, func() { Compound() })
    if Count() != 0 || Current() != nil {
        t.Fatal("the old null wrapper must fail before evaluating the RHS")
    }
    requireNullPointer(t, func() { Increment() })
    if Current() != nil {
        t.Fatal("a failed update must leave the static field null")
    }
    before := CurrentFlag()
    requireNullPointer(t, func() { BooleanCompound() })
    if Count() != 1 || CurrentFlag() != before {
        t.Fatal("boolean &= must evaluate and unbox its RHS even when the old value is false")
    }
}
`)
}

func TestStaticBoxedUpdatesRetainPrimitiveNarrowing(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class StaticBoxedNarrowingProgram {
    static Byte small = (byte) 127;
    static Short medium = (short) 32767;
    static Character character = (char) 65535;
    static Long wide = 9223372036854775807L;
    static Float fraction = 1.5F;
    static Double precise = 2.5D;
    public static Byte nextByte() { return ++small; }
    public static Short nextShort() { return ++medium; }
    public static Character nextCharacter() { return ++character; }
    public static Long nextLong() { return ++wide; }
    public static Float nextFloat() { return fraction += 0.25F; }
    public static Double nextDouble() { return precise += 0.25D; }
}
`)
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func TestStaticPrimitiveBoundaries(t *testing.T) {
    if got := stdjava.UnboxByte(NextByte()); got != -128 { t.Fatalf("byte = %d", got) }
    if got := stdjava.UnboxShort(NextShort()); got != -32768 { t.Fatalf("short = %d", got) }
    if got := stdjava.UnboxCharacter(NextCharacter()); got != 0 { t.Fatalf("char = %d", got) }
    if got := stdjava.UnboxLong(NextLong()); got != -9223372036854775808 { t.Fatalf("long = %d", got) }
    if got := stdjava.UnboxFloat(NextFloat()); got != 1.75 { t.Fatalf("float = %g", got) }
    if got := stdjava.UnboxDouble(NextDouble()); got != 2.75 { t.Fatalf("double = %g", got) }
}
`)
}
