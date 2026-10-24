package stdjava

import (
	"slices"
	"testing"
)

var singleByteEncode188Result *PrimitiveArray[int8]

func TestSingleByteEncode188AllocationBudget(t *testing.T) {
	units := make([]uint16, 4096)
	for i := range units {
		units[i] = uint16(i & 255)
	}
	value := NewJavaStringUTF16(units)
	for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
		t.Run(codec.Name(), func(t *testing.T) {
			allocations := testing.AllocsPerRun(100, func() { singleByteEncode188Result = JavaStringGetBytes(value, codec) })
			if PrimitiveArrayLength(singleByteEncode188Result) != 4096 {
				t.Fatal("wrong output length")
			}
			t.Logf("allocations/call=%.0f maximum=2", allocations)
			if allocations > 2 {
				t.Fatalf("single-byte encode allocations %.0f > 2", allocations)
			}
		})
	}
}

func TestSingleByteEncode188UnitsAndOwnership(t *testing.T) {
	cases := []struct {
		units        []uint16
		ascii, latin []int8
	}{
		{nil, []int8{}, []int8{}},
		{[]uint16{0, 'A', 127, 128, 255, 256, 0x7ff, 0xffff}, []int8{0, 'A', 127, '?', '?', '?', '?', '?'}, []int8{0, 'A', 127, -128, -1, '?', '?', '?'}},
		{[]uint16{0xd800}, []int8{'?'}, []int8{'?'}},
		{[]uint16{0xdc00}, []int8{'?'}, []int8{'?'}},
		{[]uint16{0xd800, 'A', 0xdc00}, []int8{'?', 'A', '?'}, []int8{'?', 'A', '?'}},
		{[]uint16{0xd800, 0xdc00}, []int8{'?'}, []int8{'?'}},
		{[]uint16{0xdbff, 0xdfff}, []int8{'?'}, []int8{'?'}},
		{[]uint16{0xd800, 0xd801, 0xdc00, 0xdc01}, []int8{'?', '?', '?'}, []int8{'?', '?', '?'}},
		{[]uint16{0xdc00, 0xd800, 0xdc00, 0xd800}, []int8{'?', '?', '?'}, []int8{'?', '?', '?'}},
	}
	for _, row := range cases {
		value := NewJavaStringUTF16(row.units)
		for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
			want := row.ascii
			if codec == ISO_8859_1 {
				want = row.latin
			}
			a, b := JavaStringGetBytes(value, codec), JavaStringGetBytes(value, codec)
			if a == b || a.ComponentType() != PrimitiveByteTypeID || !slices.Equal(a.Elements, want) || !slices.Equal(b.Elements, want) {
				t.Fatalf("%s units%x got%v/%v want%v", codec.Name(), row.units, a.Elements, b.Elements, want)
			}
			if len(a.Elements) > 0 {
				if &a.Elements[0] == &b.Elements[0] {
					t.Fatal("mutable output payload aliases")
				}
				a.Elements[0] ^= 127
			}
			if !slices.Equal(b.Elements, want) || !slices.Equal(JavaStringGetBytes(value, codec).Elements, want) || !slices.Equal(value.UTF16Copy(), row.units) {
				t.Fatal("output mutation changed another output or immutable input")
			}
		}
	}
	// Check every legal Java char independently, including every isolated surrogate.
	for unit := 0; unit <= 65535; unit++ {
		value := NewJavaStringUTF16([]uint16{uint16(unit)})
		for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
			maximum := 127
			if codec == ISO_8859_1 {
				maximum = 255
			}
			want := int8('?')
			if unit <= maximum {
				want = int8(unit)
			}
			actual := JavaStringGetBytes(value, codec)
			if len(actual.Elements) != 1 || actual.Elements[0] != want {
				t.Fatalf("%s unit%04x got%v want%d", codec.Name(), unit, actual.Elements, want)
			}
		}
	}
}

func TestSingleByteEncode188NamedNullAndRange(t *testing.T) {
	chars := PrimitiveArrayLiteral[rune](PrimitiveCharTypeID, 0xd83d, 0xde00, 'A')
	for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
		name := NewJavaStringUTF16(charsetEncodingASCIIUnits(codec.Name()))
		for _, row := range []struct {
			offset, count int32
			want          []int8
		}{{0, 1, []int8{'?'}}, {1, 1, []int8{'?'}}, {0, 2, []int8{'?'}}, {1, 2, []int8{'?', 'A'}}} {
			value := JavaStringFromCharsRange(chars, row.offset, row.count)
			direct, named := JavaStringGetBytes(value, codec), JavaStringGetBytesNamed(value, name)
			if direct == named || !slices.Equal(direct.Elements, row.want) || !slices.Equal(named.Elements, row.want) {
				t.Fatal("range selection changed surrogate pairing or named contract")
			}
		}
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytes(nil, codec) })
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(nil, name) })
	}
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytes(nil, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytes(NewJavaStringUTF16(nil), nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(nil, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(NewJavaStringUTF16(nil), nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromCharsRange(nil, -1, -1) })
	for _, bounds := range [][2]int32{{-1, 1}, {0, -1}, {2, 2}, {2147483647, 2147483647}} {
		charsetEncodingReferenceFailure(t, "StringIndexOutOfBoundsException", func() any { return JavaStringFromCharsRange(chars, bounds[0], bounds[1]) })
	}
	value := NewJavaStringUTF16([]uint16{'A'})
	for _, name := range []*JavaString{NewJavaStringUTF16(charsetEncodingASCIIUnits("x-no-singlebyte188")), NewJavaStringUTF16([]uint16{'x', 0xd800})} {
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(nil, name) })
		failure := charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", func() any { return JavaStringGetBytesNamed(value, name) })
		if JavaThrowableMessageDefault(failure) != name {
			t.Fatal("encoding exception lost original name identity")
		}
	}
	charsetEncodingReferenceFailure(t, "UnsupportedOperationException", func() any { return JavaStringGetBytes(value, &Charset{name: "US-ASCII"}) })
}
