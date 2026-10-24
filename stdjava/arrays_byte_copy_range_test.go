package stdjava

import (
	"slices"
	"testing"
)

func TestArraysByteCopyRangeFreshStorageComponentAndPadding(t *testing.T) {
	original := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(-128), int8(-1), int8(0), int8(127))
	for _, row := range []struct {
		from, to int32
		want     []int8
	}{
		{0, 4, []int8{-128, -1, 0, 127}}, {1, 3, []int8{-1, 0}}, {3, 7, []int8{127, 0, 0, 0}}, {4, 6, []int8{0, 0}}, {2, 2, []int8{}},
	} {
		before := slices.Clone(original.Elements)
		first := ArraysByteCopyOfRange(original, row.from, row.to)
		second := ArraysByteCopyOfRange(original, row.from, row.to)
		if first == original || first == second {
			t.Fatal("result wrapper identity reused")
		}
		if first.ComponentType() != PrimitiveByteTypeID || first.JavaArrayTypeID() != ArrayTypeID(PrimitiveByteTypeID) {
			t.Fatal("result lost byte descriptor")
		}
		if !slices.Equal(first.Elements, row.want) || !slices.Equal(original.Elements, before) {
			t.Fatalf("copy=%v input=%v want%v", first.Elements, original.Elements, row.want)
		}
		if len(first.Elements) > 0 {
			first.Elements[0] = 37
			if !slices.Equal(second.Elements, row.want) || !slices.Equal(original.Elements, before) {
				t.Fatal("copy shares storage")
			}
		}
	}
	empty := NewPrimitiveArray[int8](0, PrimitiveByteTypeID)
	a := ArraysByteCopyOfRange(empty, 0, 0)
	b := ArraysByteCopyOfRange(empty, 0, 0)
	if a == empty || a == b {
		t.Fatal("empty full-range clone reused identity")
	}
}

func TestArraysByteCopyRangeJDK21ExceptionOrder(t *testing.T) {
	original := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(1), int8(2), int8(3))
	for _, row := range []struct {
		input          *PrimitiveArray[int8]
		from, to       int32
		class, message string
	}{
		{nil, 0, 0, "NPE", ""}, {nil, 0, -1, "NPE", ""}, {nil, 1, 0, "IAE", "1 > 0"}, {original, 0, -1, "IAE", "0 > -1"},
		{nil, 1, 1, "NPE", ""}, {nil, -1, -1, "NPE", ""}, {original, -1, -1, "AIOOBE", ""}, {original, 4, 4, "AIOOBE", ""},
		{nil, -1, 2147483647, "NAS", "-2147483648"}, {original, -1, 2147483647, "NAS", "-2147483648"},
		{nil, -2147483648, 0, "NAS", "-2147483648"}, {original, -2147483648, 0, "NAS", "-2147483648"},
		{nil, -2147483648, 2147483647, "NAS", "-1"}, {original, -2147483648, 2147483647, "NAS", "-1"},
		{nil, 0, -2147483648, "NPE", ""}, {original, 0, -2147483648, "IAE", "0 > -2147483648"},
	} {
		before := slices.Clone(original.Elements)
		func() {
			defer func() {
				r := recover()
				class := ""
				switch r.(type) {
				case NullPointerException:
					class = "NPE"
				case IllegalArgumentException:
					class = "IAE"
				case NegativeArraySizeException:
					class = "NAS"
				case ArrayIndexOutOfBoundsException:
					class = "AIOOBE"
				}
				if class != row.class {
					t.Fatalf("from%d to%d panic%T want%s", row.from, row.to, r, row.class)
				}
				if row.message != "" && GetMessage(r) != row.message {
					t.Fatalf("message=%q want%q", GetMessage(r), row.message)
				}
			}()
			ArraysByteCopyOfRange(row.input, row.from, row.to)
			t.Fatal("expected exception")
		}()
		if !slices.Equal(original.Elements, before) {
			t.Fatal("failed copy changed source")
		}
	}
}
