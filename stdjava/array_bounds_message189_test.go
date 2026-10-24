package stdjava

import "testing"

// Expected messages are frozen from the portable Java21 bytecode controls in
// testdata/array-bounds189/ArrayBounds189.java. Reflection and utility APIs have
// separate contracts and are deliberately not used as array-bytecode oracles.
func arrayBounds189Require(t *testing.T, action func(), kind, message string, checkMessage bool) {
	t.Helper()
	var recovered any
	func() { defer func() { recovered = recover() }(); action() }()
	normalized := NormalizePanic(recovered)
	if !CaughtAs(normalized, kind) {
		t.Errorf("exception=%T %v, want %s", normalized, normalized, kind)
		return
	}
	if checkMessage && GetMessage(normalized) != message {
		t.Errorf("array message=%q want frozen JDK21 %q", GetMessage(normalized), message)
	}
}
func arrayBounds189Primitive[T comparable](t *testing.T, component TypeID, value T) {
	t.Helper()
	for _, row := range []struct {
		name          string
		length, index int32
		message       string
	}{
		{"negative", 2, -1, "Index -1 out of bounds for length 2"},
		{"high", 2, 2, "Index 2 out of bounds for length 2"},
		{"empty", 0, 0, "Index 0 out of bounds for length 0"},
	} {
		t.Run(row.name, func(t *testing.T) {
			array := NewPrimitiveArray[T](row.length, component)
			arrayBounds189Require(t, func() { PrimitiveArrayGet(array, row.index) }, "ArrayIndexOutOfBoundsException", row.message, true)
			arrayBounds189Require(t, func() { PrimitiveArrayAssign(array, row.index, value) }, "ArrayIndexOutOfBoundsException", row.message, true)
			for _, element := range array.Elements {
				var zero T
				if element != zero {
					t.Fatal("failed checked write mutated array")
				}
			}
		})
	}
	array := NewPrimitiveArray[T](2, component)
	if PrimitiveArrayAssign(array, int32(1), value) != value || PrimitiveArrayGet(array, int32(1)) != value {
		t.Fatal("successful access/result changed")
	}
}
func TestArrayBounds189PrimitiveCheckedMessages(t *testing.T) {
	t.Run("boolean", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveBooleanTypeID, true) })
	t.Run("byte", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveByteTypeID, int8(7)) })
	t.Run("short", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveShortTypeID, int16(7)) })
	t.Run("char", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveCharTypeID, uint16(7)) })
	t.Run("int", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveIntTypeID, int32(7)) })
	t.Run("long", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveLongTypeID, int64(7)) })
	t.Run("float", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveFloatTypeID, float32(7)) })
	t.Run("double", func(t *testing.T) { arrayBounds189Primitive(t, PrimitiveDoubleTypeID, float64(7)) })
}
func TestArrayBounds189ReferenceRankAndIndexMessages(t *testing.T) {
	for _, row := range []struct {
		name          string
		length, index int32
		message       string
	}{
		{"negative", 2, -1, "Index -1 out of bounds for length 2"},
		{"high", 2, 2, "Index 2 out of bounds for length 2"},
		{"empty", 0, 0, "Index 0 out of bounds for length 0"},
		{"minimum-int", 2, -2147483648, "Index -2147483648 out of bounds for length 2"},
		{"maximum-int", 2, 2147483647, "Index 2147483647 out of bounds for length 2"},
	} {
		t.Run(row.name, func(t *testing.T) {
			a := NewReferenceArray(row.length, ObjectTypeID)
			value := JavaStringFromHostUTF8("stored")
			arrayBounds189Require(t, func() { ReferenceArrayGet[any](a, row.index, ObjectTypeID) }, "ArrayIndexOutOfBoundsException", row.message, true)
			arrayBounds189Require(t, func() { ReferenceArraySet(a, row.index, value) }, "ArrayIndexOutOfBoundsException", row.message, true)
			arrayBounds189Require(t, func() { ReferenceArrayAssign[any](a, row.index, value, ObjectTypeID) }, "ArrayIndexOutOfBoundsException", row.message, true)
			for _, x := range a.elements {
				if x != nil {
					t.Fatal("failed reference write mutated array")
				}
			}
		})
	}
	a := NewPrimitiveArray[int32](2, PrimitiveIntTypeID)
	arrayBounds189Require(t, func() { PrimitiveArrayGet(a, int8(-128)) }, "ArrayIndexOutOfBoundsException", "Index -128 out of bounds for length 2", true)
	arrayBounds189Require(t, func() { PrimitiveArrayGet(a, int16(32767)) }, "ArrayIndexOutOfBoundsException", "Index 32767 out of bounds for length 2", true)
	arrayBounds189Require(t, func() { PrimitiveArrayGet(a, uint16(65535)) }, "ArrayIndexOutOfBoundsException", "Index 65535 out of bounds for length 2", true)
	row := NewPrimitiveArray[int32](1, PrimitiveIntTypeID)
	outer := NewReferenceArray(2, ArrayTypeID(PrimitiveIntTypeID))
	ReferenceArraySet(outer, 0, row)
	arrayBounds189Require(t, func() { ReferenceArrayGet[*PrimitiveArray[int32]](outer, int32(2), ArrayTypeID(PrimitiveIntTypeID)) }, "ArrayIndexOutOfBoundsException", "Index 2 out of bounds for length 2", true)
	arrayBounds189Require(t, func() {
		PrimitiveArrayGet(ReferenceArrayGet[*PrimitiveArray[int32]](outer, 0, ArrayTypeID(PrimitiveIntTypeID)), 1)
	}, "ArrayIndexOutOfBoundsException", "Index 1 out of bounds for length 1", true)
}
func TestArrayBounds189LegacyCheckedStores(t *testing.T) {
	arrayBounds189Require(t, func() { ArraySet([]int32{1, 2}, int8(-1), int32(7)) }, "ArrayIndexOutOfBoundsException", "Index -1 out of bounds for length 2", true)
	arrayBounds189Require(t, func() { ArraySet([]int32{1, 2}, int32(2), int32(7)) }, "ArrayIndexOutOfBoundsException", "Index 2 out of bounds for length 2", true)
	arrayBounds189Require(t, func() { ArraySet([]int32{}, int32(-1), int32(7)) }, "ArrayIndexOutOfBoundsException", "Index -1 out of bounds for length 0", true)
	arrayBounds189Require(t, func() { ArraySet([]int32(nil), int32(-1), int32(7)) }, "NullPointerException", "", false)
	a := []int32{1, 2}
	if ArraySet(a, int16(1), int32(7)) != 7 || a[1] != 7 {
		t.Fatal("legacy successful store/result changed")
	}
}
func TestArrayBounds189EvaluationAndRefusalOrder(t *testing.T) {
	markerValue := NewIllegalStateException("marker")
	marker := &markerValue
	for _, name := range []string{"primitive-negative", "reference-negative", "primitive-null", "reference-null", "abrupt-index", "abrupt-rhs", "reference-abrupt-rhs", "component-check"} {
		t.Run(name, func(t *testing.T) {
			trace := ""
			primitive := NewPrimitiveArray[int32](2, PrimitiveIntTypeID)
			reference := NewReferenceArray(2, StringTypeID)
			pa := func() *PrimitiveArray[int32] {
				trace += "A"
				if name == "primitive-null" {
					return nil
				}
				return primitive
			}
			ra := func() *ReferenceArray {
				trace += "A"
				if name == "reference-null" || name == "reference-abrupt-rhs" {
					return nil
				}
				return reference
			}
			index := func() int32 {
				trace += "I"
				if name == "abrupt-index" {
					panic(marker)
				}
				if name == "component-check" {
					return 0
				}
				return -1
			}
			value := func() int32 {
				trace += "V"
				if name == "abrupt-rhs" || name == "reference-abrupt-rhs" {
					panic(marker)
				}
				return 7
			}
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				if name == "reference-negative" || name == "reference-null" || name == "reference-abrupt-rhs" || name == "component-check" {
					ReferenceArrayAssign[any](ra(), index(), IntegerValueOf(value()), ObjectTypeID)
				} else {
					PrimitiveArrayAssign(pa(), index(), value())
				}
			}()
			wantTrace := "AIV"
			if name == "abrupt-index" {
				wantTrace = "AI"
			}
			if trace != wantTrace {
				t.Fatalf("trace=%q want %q", trace, wantTrace)
			}
			if name == "abrupt-index" || name == "abrupt-rhs" || name == "reference-abrupt-rhs" {
				if recovered != marker {
					t.Fatalf("abrupt exception identity changed: %T", recovered)
				}
			} else {
				normalized := NormalizePanic(recovered)
				kind := "ArrayIndexOutOfBoundsException"
				if name == "primitive-null" || name == "reference-null" {
					kind = "NullPointerException"
				}
				if name == "component-check" {
					kind = "ArrayStoreException"
				}
				if !CaughtAs(normalized, kind) {
					t.Fatalf("exception=%T %v want %s", normalized, normalized, kind)
				}
				if kind == "ArrayIndexOutOfBoundsException" && GetMessage(normalized) != "Index -1 out of bounds for length 2" {
					t.Error("bounds message differs from frozen JDK21")
				}
			}
			if primitive.Elements[0] != 0 || primitive.Elements[1] != 0 || reference.elements[0] != nil || reference.elements[1] != nil {
				t.Fatal("failed evaluation/check modified saved target")
			}
		})
	}
}
func TestArrayBounds189SharedAsListMessages(t *testing.T) {
	a := NewReferenceArray(2, StringTypeID)
	list := AsListArray[*JavaString](a, StringTypeID)
	arrayBounds189Require(t, func() { list.Get(-1) }, "ArrayIndexOutOfBoundsException", "Index -1 out of bounds for length 2", true)
	arrayBounds189Require(t, func() { list.Set(2, JavaStringFromHostUTF8("value")) }, "ArrayIndexOutOfBoundsException", "Index 2 out of bounds for length 2", true)
}
func TestArrayBounds189ConstructorAndUnsharedAPIShields(t *testing.T) {
	if NewArrayIndexOutOfBoundsException("caller message").Message() != "caller message" {
		t.Fatal("explicit exception constructor changed")
	}
	// This utility has its own pre-existing (incomplete) contract. Its current
	// empty message is deliberately preserved, not relabeled as bytecode parity.
	arrayBounds189Require(t, func() { ArraysByteCopyOfRange(NewPrimitiveArray[int8](2, PrimitiveByteTypeID), -1, 1) }, "ArrayIndexOutOfBoundsException", "", true)
	arrayBounds189Require(t, func() { PrimitiveArrayAssign((*PrimitiveArray[int32])(nil), -1, int32(7)) }, "NullPointerException", "array access on null", true)
	arrayBounds189Require(t, func() { ReferenceArrayGet[any](nil, -1, ObjectTypeID) }, "NullPointerException", "array access on null", true)
}
