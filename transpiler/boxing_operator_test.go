package transpiler

import "testing"

func TestBoxingOperatorsIdentityPromotionAndImmutableUpdates(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedOperatorProgram {
    public static String run() {
        Integer first = 128;
        Integer equal = 128;
        Integer alias = first;
        Integer before = first++;
        Integer after = ++first;
        first += 2;
        Byte narrow = (byte)127;
        Byte narrowOld = narrow++;
        Character character = (char)65535;
        ++character;
        Integer[] slots = new Integer[] { 5 };
        Integer slotAlias = slots[0];
        int index = 0;
        Integer slotOld = slots[index++]++;
        slots[0] += 2;
        Long distance = 65L;
        Long wide = -8L;
        Boolean yes = true;
        Boolean no = false;
        return (first == alias) + ":" + (before == alias) + ":" + (after == first)
            + ":" + (equal == alias) + ":" + (equal == 128) + ":" + first + ":" + alias
            + ":" + narrow + ":" + narrowOld + ":" + (int)character
            + ":" + slots[0] + ":" + (slotAlias == slotOld) + ":" + index
            + ":" + (wide >> distance) + ":" + (wide >>> distance)
            + ":" + (yes & no) + ":" + (yes | no) + ":" + (yes ^ no)
            + ":" + !no + ":" + (yes == true);
    }
}`)
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestBoxedOperators(t *testing.T) {
    const want = "false:true:false:false:true:132:128:-128:127:0:8:true:1:-4:9223372036854775804:false:true:true:true:true"
    if got := Run(); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}
`)
}

func TestBoxingOperatorsNullTimingAndCheckedCasts(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedOperatorNullProgram {
    private static int effects = 0;
    private static int mark() { effects++; return 3; }
    public static int seen() { return effects; }
    public static int nullArithmetic() { effects = 0; Integer n = null; return n + mark(); }
    public static int nullCompound() { effects = 0; Integer n = null; n += mark(); return n; }
    public static int nullArrayCompound() { effects = 0; Integer[] n = new Integer[1]; n[0] += mark(); return n[0]; }
    public static boolean nullBoolean() { Boolean no = false; Boolean absent = null; return no & absent; }
    public static boolean shortCircuit() { Boolean no = false; Boolean absent = null; return no && absent; }
    public static Integer nullReferenceCast() { Object value = null; return (Integer)value; }
    public static int nullPrimitiveCast() { Object value = null; return (int)value; }
    public static Integer wrongReferenceCast() { Object value = 3L; return (Integer)value; }
    public static long widening() { Integer value = 23; return (long)value; }
    public static int saturating() { Double value = 1.0e100; return (int)value; }
    public static Double negativeDoubleZero() { return -0.0; }
    public static Float negativeFloatZero() { return -0.0F; }
    public static double infinity() { Double value = 1.0; return value / 0.0; }
    public static float remainder() { Float value = 5.5F; value %= 2.0F; return value; }
    public static double remainderExpression() { Double value = 5.5; return value % 2.0; }
    public static String conditional(Boolean choose, Integer left, Integer right) {
        Integer selected = choose ? left : right;
        return (selected == left) + ":" + selected;
    }
}`)
	runGoTestInTempModule(t, out, `
package main
import (
    "math"
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)
func TestBoxedNullBehavior(t *testing.T) {
    nulls := []func(){func(){NullArithmetic()},func(){NullCompound()},func(){NullArrayCompound()},func(){NullBoolean()},func(){NullPrimitiveCast()}}
    for i, call := range nulls {
        func() {
            defer func() {
                if _, ok := recover().(stdjava.NullPointerException); !ok { t.Errorf("case %d: expected Java null failure", i) }
            }()
            call()
            t.Errorf("case %d: expected panic", i)
        }()
        if Seen() != 0 { t.Fatalf("case %d: RHS evaluated before left null failure", i) }
    }
    if ShortCircuit() { t.Fatal("short circuit result") }
    if NullReferenceCast() != nil { t.Fatal("null cast did not preserve null") }
    func() {
        defer func() { if _, ok := recover().(stdjava.ClassCastException); !ok { t.Error("expected Java checked cast failure") } }()
        WrongReferenceCast()
    }()
    if Widening() != 23 || Saturating() != 2147483647 { t.Fatal("wrapper numeric casts") }
    if !math.Signbit(stdjava.UnboxDouble(NegativeDoubleZero())) || !math.Signbit(float64(stdjava.UnboxFloat(NegativeFloatZero()))) { t.Fatal("floating unary minus lost signed zero") }
    if !math.IsInf(Infinity(), 1) || Remainder() != 1.5 || RemainderExpression() != 1.5 { t.Fatal("floating division/remainder") }
    value := stdjava.NewInteger(999)
    if got := Conditional(stdjava.BoxBoolean(true), value, nil); got != "true:999" { t.Fatal(got) }
    if got := Conditional(stdjava.BoxBoolean(false), value, nil); got != "false:null" { t.Fatal(got) }
}
`)
}

func TestBoxingOperatorsUnboxFinalWrapperTypeParameterBounds(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoundedBoxedOperatorProgram {
    public static <T extends Integer> String integer(T value) {
        int total = 2;
        total += value;
        int selected = true ? value : 8;
        return (value + 1) + ":" + (-value) + ":" + (value == 5) + ":" + (5 != value)
            + ":" + total + ":" + selected + ":" + (long)value;
    }
    public static <T extends Long> String shift(T value, Integer distance) {
        return (value >> distance) + ":" + (value >>> distance) + ":" + (value << distance);
    }
    public static <T extends Boolean> boolean bool(T value) {
        boolean result = true;
        result &= value;
        return !value | (value == true) & (true ? value : false) & result;
    }
    public static <T extends Double> double floating(T value) { return -value % 2.0; }
    public static <T extends Number> boolean same(T left, T right) { return left == right; }
    public static <T extends Integer, U extends T> int chained(U value) { return value * 2; }
}`)
	runGoTestInTempModule(t, out, `
package main
import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)
func TestBoundedBoxedOperators(t *testing.T) {
    if got := Integer(stdjava.BoxInteger(5)); got != "6:-5:true:false:7:5:5" { t.Fatal(got) }
    if got := Shift(stdjava.BoxLong(-8), stdjava.BoxInteger(65)); got != "-4:9223372036854775804:-16" { t.Fatal(got) }
    if !Bool(stdjava.BoxBoolean(true)) || !Bool(stdjava.BoxBoolean(false)) { t.Fatal("bounded boolean") }
    if got := Floating(stdjava.BoxDouble(5.5)); got != -1.5 { t.Fatal(got) }
    value := stdjava.NewInteger(5)
    if !Same(value, value) || Same(value, stdjava.NewInteger(5)) { t.Fatal("Number-bound reference identity") }
    if got := Chained[*stdjava.Integer, *stdjava.Integer](value); got != 10 { t.Fatal(got) }
    defer func() { if _, ok := recover().(stdjava.NullPointerException); !ok { t.Error("bounded null must unbox with Java null failure") } }()
    Integer[*stdjava.Integer](nil)
}
`)
}

func TestBoxingOperatorsSequenceEarlierPrimitiveReads(t *testing.T) {
	out := renderGoFileFromJava(t, `
class OrderedPrimitiveHolder { int number; }
public class BinaryReadOrderingProgram {
    public static int addition() { int value = 10; return value + value++; }
    public static boolean equality() { int value = 10; return value == value++; }
    public static int shift() { int value = 2; return value << (value = 1); }
    public static int unsigned() { int value = -8; return value >>> (value = 1); }
    public static boolean bool() { boolean value = true; return value ^ (value = false); }
    public static String text() { String value = "first"; return value + (value = "second"); }
    public static boolean constant() { int value = 0; return Integer.MAX_VALUE == (value = 2147483647); }
    public static int castFailureOrder() {
        OrderedPrimitiveHolder holder = null;
        Object wrong = 1L;
        try { return holder.number + (int)wrong; }
        catch (NullPointerException expected) { return 1; }
        catch (ClassCastException expected) { return 2; }
    }
    public static int arrayFailureOrder() {
        OrderedPrimitiveHolder holder = null;
        int[] empty = new int[0];
        try { return holder.number + empty[0]; }
        catch (NullPointerException expected) { return 1; }
        catch (ArrayIndexOutOfBoundsException expected) { return 2; }
    }
}`)
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestBinaryReads(t *testing.T) {
    if Addition() != 20 || !Equality() || Shift() != 4 || Unsigned() != 2147483644 || !Bool() || Text() != "firstsecond" || !Constant() {
        t.Fatalf("binary values: %d %v %d %d %v %s", Addition(), Equality(), Shift(), Unsigned(), Bool(), Text())
    }
    if CastFailureOrder() != 1 || ArrayFailureOrder() != 1 { t.Fatal("right failure happened before left null field read") }
}
`)
}
