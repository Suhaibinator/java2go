package stdjava

import (
	"math"
	"sync"
	"testing"
)

func expectBoxedException(t *testing.T, name string, invoke func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		exception, ok := recovered.(Throwable)
		if !ok || exception.ThrowableTypeName() != name {
			t.Fatalf("panic = %T (%v), want %s", recovered, recovered, name)
		}
	}()
	invoke()
}

func TestBoxedCachesAndFreshConstructors(t *testing.T) {
	for _, value := range []bool{false, true} {
		if BoxBoolean(value) != BoxBoolean(value) || NewBoolean(value) == BoxBoolean(value) ||
			NewBoolean(value) == NewBoolean(value) {
			t.Fatalf("Boolean(%v) cache/constructor identity", value)
		}
	}
	for value := -128; value <= 127; value++ {
		if BoxByte(int8(value)) != BoxByte(int8(value)) ||
			BoxShort(int16(value)) != BoxShort(int16(value)) ||
			BoxInteger(int32(value)) != BoxInteger(int32(value)) ||
			BoxLong(int64(value)) != BoxLong(int64(value)) {
			t.Fatalf("cache did not retain identity for %d", value)
		}
		if NewByte(int8(value)) == BoxByte(int8(value)) ||
			NewShort(int16(value)) == BoxShort(int16(value)) ||
			NewInteger(int32(value)) == BoxInteger(int32(value)) ||
			NewLong(int64(value)) == BoxLong(int64(value)) {
			t.Fatalf("constructor reused cached object for %d", value)
		}
	}
	for value := rune(0); value < 128; value++ {
		if BoxCharacter(value) != BoxCharacter(value) || NewCharacter(value) == BoxCharacter(value) {
			t.Fatalf("Character(%d) cache/constructor identity", value)
		}
	}
	for _, value := range []int32{-129, 128, 4096} {
		if BoxShort(int16(value)) == BoxShort(int16(value)) ||
			BoxInteger(value) == BoxInteger(value) || BoxLong(int64(value)) == BoxLong(int64(value)) {
			t.Fatalf("unexpected cache beyond fixed range at %d", value)
		}
	}
	if BoxCharacter(128) == BoxCharacter(128) || BoxCharacter(65535) == BoxCharacter(65535) ||
		BoxFloat(1) == BoxFloat(1) || BoxDouble(1) == BoxDouble(1) {
		t.Fatal("uncached boxing reused an object")
	}
	original := BoxInteger(7)
	updated := BoxInteger(original.IntValue() + 1)
	if original.IntValue() != 7 || updated.IntValue() != 8 || original == updated {
		t.Fatal("boxed update did not preserve the old immutable object")
	}
}

func TestBoxedCacheConcurrency(t *testing.T) {
	var work sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		work.Add(1)
		go func() {
			defer work.Done()
			for value := -128; value <= 127; value++ {
				if BoxByte(int8(value)) != BoxByte(int8(value)) ||
					BoxShort(int16(value)) != BoxShort(int16(value)) ||
					BoxInteger(int32(value)) != BoxInteger(int32(value)) ||
					BoxLong(int64(value)) != BoxLong(int64(value)) {
					t.Errorf("concurrent cached boxing lost identity at %d", value)
					return
				}
				if BoxBoolean(true) != BoxBoolean(true) || BoxCharacter(65) != BoxCharacter(65) {
					t.Error("concurrent nonnumeric cache lost identity")
					return
				}
			}
		}()
	}
	work.Wait()
}

func TestBoxedNullAccessAndUnboxing(t *testing.T) {
	cases := map[string]func(){
		"Boolean":          func() { UnboxBoolean(nil) },
		"Byte":             func() { UnboxByte(nil) },
		"Short":            func() { UnboxShort(nil) },
		"Character":        func() { UnboxCharacter(nil) },
		"Integer":          func() { UnboxInteger(nil) },
		"Long":             func() { UnboxLong(nil) },
		"Float":            func() { UnboxFloat(nil) },
		"Double":           func() { UnboxDouble(nil) },
		"Number interface": func() { NumberDoubleValue((*Integer)(nil)) },
		"instance equals":  func() { (*Integer)(nil).Equals(nil) },
		"instance hash":    func() { (*Double)(nil).HashCode() },
		"instance text":    func() { (*Boolean)(nil).String() },
		"compare argument": func() { BoxLong(1).CompareTo(nil) },
		"compare receiver": func() { (*Character)(nil).CompareTo(BoxCharacter('a')) },
	}
	for name, invoke := range cases {
		t.Run(name, func(t *testing.T) { expectBoxedException(t, "NullPointerException", invoke) })
	}
	if BoxInteger(1).Equals(nil) || BoxInteger(1).Equals((*Integer)(nil)) {
		t.Fatal("nonnull wrapper equals null")
	}
}

func TestBoxedNominalEqualityHashAndText(t *testing.T) {
	type boxed interface {
		Equals(any) bool
		HashCode() int32
		String() string
	}
	cases := []struct {
		value boxed
		equal boxed
		hash  int32
		text  string
	}{
		{BoxBoolean(true), NewBoolean(true), 1231, "true"},
		{BoxBoolean(false), NewBoolean(false), 1237, "false"},
		{BoxByte(-8), NewByte(-8), -8, "-8"},
		{BoxShort(-300), NewShort(-300), -300, "-300"},
		{BoxCharacter('A'), NewCharacter('A'), 65, "A"},
		{BoxInteger(1000), NewInteger(1000), 1000, "1000"},
		{BoxLong(1 << 32), NewLong(1 << 32), 1, "4294967296"},
		{BoxFloat(1.5), NewFloat(1.5), int32(0x3fc00000), "1.5"},
		{BoxDouble(1.5), NewDouble(1.5), int32(0x3ff80000), "1.5"},
	}
	for _, tc := range cases {
		if !tc.value.Equals(tc.equal) || !tc.equal.Equals(tc.value) ||
			tc.value.HashCode() != tc.hash || tc.equal.HashCode() != tc.hash || tc.value.String() != tc.text {
			t.Errorf("%T equality/hash/text = %v/%d/%q, want true/%d/%q",
				tc.value, tc.value.Equals(tc.equal), tc.value.HashCode(), tc.value.String(), tc.hash, tc.text)
		}
		if tc.value.Equals(nil) {
			t.Errorf("%T equals nil", tc.value)
		}
	}
	if BoxCharacter(1).Equals(BoxInteger(1)) || BoxInteger(1).Equals(BoxLong(1)) ||
		BoxFloat(1).Equals(BoxDouble(1)) || BoxInteger(1).Equals(int32(1)) {
		t.Fatal("wrapper equality crossed a nominal type boundary")
	}
	if NewInteger(1000) == NewInteger(1000) {
		t.Fatal("equal fresh wrappers share reference identity")
	}
	if BoxByte(-128).CompareTo(BoxByte(127)) != -255 ||
		BoxShort(-32768).CompareTo(BoxShort(32767)) != -65535 ||
		BoxCharacter(0).CompareTo(BoxCharacter(65535)) != -65535 ||
		BoxInteger(math.MinInt32).CompareTo(BoxInteger(math.MaxInt32)) != -1 ||
		BoxLong(math.MaxInt64).CompareTo(BoxLong(math.MinInt64)) != 1 ||
		BoxBoolean(false).CompareTo(BoxBoolean(true)) != -1 {
		t.Fatal("wrapper comparison does not match Java")
	}
}

func TestBoxedFloatingNaNAndSignedZero(t *testing.T) {
	floatNaNs := []*Float{
		BoxFloat(math.Float32frombits(0x7fc00000)),
		BoxFloat(math.Float32frombits(0x7f800001)),
		BoxFloat(math.Float32frombits(0xffc00042)),
	}
	for _, value := range floatNaNs {
		if !value.Equals(floatNaNs[0]) || value.HashCode() != int32(0x7fc00000) ||
			value.CompareTo(floatNaNs[0]) != 0 || value.CompareTo(BoxFloat(float32(math.Inf(1)))) != 1 {
			t.Fatalf("Float NaN payload %x not canonicalized", math.Float32bits(value.FloatValue()))
		}
	}
	doubleNaNs := []*Double{
		BoxDouble(math.Float64frombits(0x7ff8000000000000)),
		BoxDouble(math.Float64frombits(0x7ff0000000000001)),
		BoxDouble(math.Float64frombits(0xfff8000000000042)),
	}
	for _, value := range doubleNaNs {
		if !value.Equals(doubleNaNs[0]) || value.HashCode() != int32(0x7ff80000) ||
			value.CompareTo(doubleNaNs[0]) != 0 || value.CompareTo(BoxDouble(math.Inf(1))) != 1 {
			t.Fatalf("Double NaN payload %x not canonicalized", math.Float64bits(value.DoubleValue()))
		}
	}
	negativeFloatZero := BoxFloat(float32(math.Copysign(0, -1)))
	negativeDoubleZero := BoxDouble(math.Copysign(0, -1))
	if negativeFloatZero.Equals(BoxFloat(0)) || negativeDoubleZero.Equals(BoxDouble(0)) ||
		negativeFloatZero.HashCode() != math.MinInt32 || negativeDoubleZero.HashCode() != math.MinInt32 ||
		negativeFloatZero.CompareTo(BoxFloat(0)) != -1 || negativeDoubleZero.CompareTo(BoxDouble(0)) != -1 ||
		negativeFloatZero.String() != "-0.0" || negativeDoubleZero.String() != "-0.0" {
		t.Fatal("floating wrapper signed-zero semantics lost")
	}
}

func TestBoxedNumberConversions(t *testing.T) {
	values := []JavaNumber{BoxByte(3), BoxShort(3), BoxInteger(3), BoxLong(3), BoxFloat(3), BoxDouble(3)}
	for _, value := range values {
		if value.ByteValue() != 3 || value.ShortValue() != 3 || value.IntValue() != 3 ||
			value.LongValue() != 3 || value.FloatValue() != 3 || value.DoubleValue() != 3 ||
			NumberDoubleValue(value) != 3 || NumberFloatValue(value) != 3 ||
			NumberLongValue(value) != 3 || NumberIntValue(value) != 3 ||
			NumberShortValue(value) != 3 || NumberByteValue(value) != 3 {
			t.Errorf("%T Number accessors failed", value)
		}
	}
	if BoxLong(math.MaxInt64).IntValue() != -1 || BoxInteger(130).ByteValue() != -126 ||
		BoxLong(65535).ShortValue() != -1 || BoxCharacter(0x1ffff).CharValue() != 65535 {
		t.Fatal("integral narrowing semantics")
	}
	// The +1 above a float midpoint is lost by an intermediate float64.
	largeLong := int64(1<<62) + int64(1<<38) + 1
	wantFloat := math.Float32frombits(math.Float32bits(float32(int64(1<<62))) + 1)
	if got := BoxLong(largeLong).FloatValue(); got != wantFloat {
		t.Fatalf("Long.floatValue = %v, want direct rounding %v", got, wantFloat)
	}
	if got := NumberFloatValue(largeLong); got != wantFloat {
		t.Fatalf("NumberFloatValue(long) = %v, want %v", got, wantFloat)
	}
	for _, value := range []JavaNumber{BoxFloat(float32(math.Inf(1))), BoxDouble(math.Inf(1))} {
		if value.IntValue() != math.MaxInt32 || value.LongValue() != math.MaxInt64 ||
			value.ByteValue() != -1 || value.ShortValue() != -1 {
			t.Errorf("%T positive infinity conversion", value)
		}
	}
	for _, value := range []JavaNumber{BoxFloat(float32(math.Inf(-1))), BoxDouble(math.Inf(-1))} {
		if value.IntValue() != math.MinInt32 || value.LongValue() != math.MinInt64 ||
			value.ByteValue() != 0 || value.ShortValue() != 0 {
			t.Errorf("%T negative infinity conversion", value)
		}
	}
	for _, value := range []JavaNumber{BoxFloat(float32(math.NaN())), BoxDouble(math.NaN())} {
		if value.IntValue() != 0 || value.LongValue() != 0 || value.ByteValue() != 0 || value.ShortValue() != 0 {
			t.Errorf("%T NaN conversion", value)
		}
	}
	if BoxDouble(-3.75).IntValue() != -3 || BoxFloat(-3.75).LongValue() != -3 {
		t.Fatal("floating conversion did not truncate toward zero")
	}
}

func TestBoxedValueOfFactories(t *testing.T) {
	if BooleanValueOf("TRUE") != BoxBoolean(true) || BooleanValueOf(nil) != BoxBoolean(false) ||
		ByteValueOf("7") != BoxByte(7) || ShortValueOf("7") != BoxShort(7) ||
		IntegerValueOf("7") != BoxInteger(7) || LongValueOf("7") != BoxLong(7) ||
		CharacterValueOf(rune('a')) != BoxCharacter('a') {
		t.Fatal("valueOf did not use boxing caches")
	}
	if ByteValueOfString("7f", 16).ByteValue() != 127 ||
		ShortValueOfString("7fff", 16).ShortValue() != 32767 ||
		IntegerValueOfString("7fffffff", 16).IntValue() != math.MaxInt32 ||
		LongValueOfString("7fffffffffffffff", 16).LongValue() != math.MaxInt64 {
		t.Fatal("radix valueOf failed")
	}
	if FloatValueOf("2.5").FloatValue() != 2.5 || DoubleValueOf("2.5").DoubleValue() != 2.5 ||
		IntegerValueOf(int8(7)) != BoxInteger(7) || LongValueOf(BoxInteger(7)) != BoxLong(7) {
		t.Fatal("string/numeric valueOf overloads failed")
	}
	expectBoxedException(t, "ClassCastException", func() { FloatValueOf(float64(1)) })
	expectBoxedException(t, "ClassCastException", func() { IntegerValueOf(int64(1)) })
	expectBoxedException(t, "NumberFormatException", func() { IntegerValueOf(nil) })
	expectBoxedException(t, "NullPointerException", func() { DoubleValueOf(nil) })
	if NewFloatFromDouble(1.25).FloatValue() != 1.25 {
		t.Fatal("Float(double) constructor")
	}
}
