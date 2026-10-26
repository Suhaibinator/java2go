package stdjava

import (
	"slices"
	"testing"
)

var singleByteDecode185Result *JavaString

// Setup and expected values stay outside the measured calls. Keeping the
// result in a package sink forces both the Java object and its payload to escape.
// The original decode/transform/rune/clone pipeline must fail this budget.
func TestSingleByteDecode185AllocationBudget(t *testing.T) {
	input := NewPrimitiveArray[int8](4096, PrimitiveByteTypeID)
	for index := range input.Elements {
		input.Elements[index] = int8(index)
	}
	for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
		t.Run(codec.Name(), func(t *testing.T) {
			allocations := testing.AllocsPerRun(100, func() {
				singleByteDecode185Result = JavaStringFromBytes(input, codec)
			})
			if singleByteDecode185Result.Length() != 4096 {
				t.Fatal("result not consumed or wrong length")
			}
			t.Logf("allocations/call=%.0f maximum=2", allocations)
			if allocations > 2 {
				t.Fatalf("single-byte decode allocations %.0f > 2", allocations)
			}
		})
	}
}

func TestSingleByteDecode185AllBytesAndOwnership(t *testing.T) {
	for _, codec := range []*Charset{US_ASCII, ISO_8859_1} {
		t.Run(codec.Name(), func(t *testing.T) {
			input := NewPrimitiveArray[int8](256, PrimitiveByteTypeID)
			want := make([]uint16, 256)
			for index := range input.Elements {
				input.Elements[index] = int8(index)
				want[index] = uint16(index)
				if codec == US_ASCII && index >= 128 {
					want[index] = 0xfffd
				}
			}
			first := JavaStringFromBytes(input, codec)
			second := JavaStringFromBytes(input, codec)
			if first == second || &first.units[0] == &second.units[0] || !slices.Equal(first.UTF16Copy(), want) || !first.Equals(second) {
				t.Fatal("all-byte widening, replacement, or independent fresh results changed")
			}
			clear(input.Elements)
			exposed := first.UTF16Copy()
			clear(exposed)
			encoded := JavaStringGetBytes(first, codec)
			clear(encoded.Elements)
			for range 20 {
				JavaStringFromBytes(input, codec)
			}
			if !slices.Equal(first.UTF16Copy(), want) || !slices.Equal(second.UTF16Copy(), want) {
				t.Fatal("mutable input, output, or later call changed retained result")
			}
			empty := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID)
			a, b := JavaStringFromBytes(empty, codec), JavaStringFromBytes(empty, codec)
			if a == b || a == JavaStringLiteralUTF16(nil) || a.Length() != 0 || b.Length() != 0 {
				t.Fatal("empty results must keep fresh noninterned wrapper identity")
			}
		})
	}
}

func TestSingleByteDecode185NamedRangeAndExceptionOrder(t *testing.T) {
	input := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 90, 65, -128, -1, 90)
	for _, row := range []struct {
		name  string
		codec *Charset
		want  []uint16
	}{
		{"US-ASCII", US_ASCII, []uint16{65, 0xfffd, 0xfffd}},
		{"ISO-8859-1", ISO_8859_1, []uint16{65, 128, 255}},
	} {
		name := NewJavaStringUTF16(charsetEncodingASCIIUnits(row.name))
		a := JavaStringFromBytesRange(input, 1, 3, row.codec)
		b := JavaStringFromBytesRangeNamed(input, 1, 3, name)
		if a == b || !slices.Equal(a.UTF16Copy(), row.want) || !a.Equals(b) {
			t.Fatal("named/range decoding changed units or identity")
		}
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytes(nil, row.codec) })
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytesNamed(nil, name) })
		charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytesRange(nil, -1, -1, row.codec) })
		for _, bounds := range [][2]int32{{-1, 1}, {0, -1}, {4, 2}, {2147483647, 2147483647}} {
			charsetEncodingReferenceFailure(t, "StringIndexOutOfBoundsException", func() any { return JavaStringFromBytesRange(input, bounds[0], bounds[1], row.codec) })
		}
	}
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytes(nil, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytes(input, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytesNamed(nil, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytesRange(nil, -1, -1, nil) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytesRangeNamed(nil, -1, -1, nil) })
	for _, name := range []*JavaString{
		NewJavaStringUTF16(charsetEncodingASCIIUnits("x-java2go-no-such-singlebyte185")),
		NewJavaStringUTF16([]uint16{'x', 0xd800}),
	} {
		for _, action := range []func() any{
			func() any { return JavaStringFromBytesNamed(nil, name) },
			func() any { return JavaStringFromBytesRangeNamed(nil, -1, -1, name) },
			func() any { return JavaStringFromBytesRangeNamed(input, -1, -1, name) },
		} {
			failure := charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", action)
			if JavaThrowableMessageDefault(failure) != name {
				t.Fatal("name resolution lost precedence or original detailMessage identity")
			}
		}
	}
	// Dispatch is by canonical singleton identity, never just a codec's name.
	charsetEncodingReferenceFailure(t, "UnsupportedOperationException", func() any { return JavaStringFromBytes(input, &Charset{name: "US-ASCII"}) })
}

func TestSingleByteDecode185UTF8UTF16Fallback(t *testing.T) {
	for _, row := range []struct {
		codec *Charset
		bytes []byte
		want  []uint16
	}{
		{UTF_8, []byte{65, 0xf0, 0x9f, 0x98, 0x80, 0xed, 0xa0, 0x80, 66}, []uint16{65, 0xd83d, 0xde00, 0xfffd, 66}},
		{UTF_16BE, []byte{0, 65, 0xd8, 0x3d, 0xde, 0, 0}, []uint16{65, 0xd83d, 0xde00, 0xfffd}},
		{UTF_16LE, []byte{65, 0, 0x3d, 0xd8, 0, 0xde, 0}, []uint16{65, 0xd83d, 0xde00, 0xfffd}},
		{UTF_16, []byte{0xff, 0xfe, 65, 0, 0x3d, 0xd8, 0, 0xde, 0}, []uint16{65, 0xd83d, 0xde00, 0xfffd}},
	} {
		input := signedByteArray(row.bytes)
		a, b := JavaStringFromBytes(input, row.codec), JavaStringFromBytes(input, row.codec)
		if a == b || !slices.Equal(a.UTF16Copy(), row.want) || !a.Equals(b) {
			t.Fatalf("%s fallback changed: %x want %x", row.codec.Name(), a.UTF16Copy(), row.want)
		}
	}
}
