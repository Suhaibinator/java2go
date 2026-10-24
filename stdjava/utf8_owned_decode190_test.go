package stdjava

import (
	"slices"
	"testing"
)

var utf8Decode190Result *JavaString

func TestUTF8Decode190AllocationBudget(t *testing.T) {
	for _, row := range []struct {
		name    string
		pattern []byte
		want    int32
	}{{"ASCII", []byte{65, 0, 127, 66}, 4096}, {"supplementary", []byte{0xf0, 0x9f, 0x98, 0x80}, 2048}, {"malformed", []byte{0xed, 0xa0, 0x80, 65}, 2048}} {
		t.Run(row.name, func(t *testing.T) {
			input := NewPrimitiveArray[int8](4096, PrimitiveByteTypeID)
			for i := range input.Elements {
				input.Elements[i] = int8(row.pattern[i%len(row.pattern)])
			}
			allocations := testing.AllocsPerRun(100, func() { utf8Decode190Result = JavaStringFromBytes(input, UTF_8) })
			if utf8Decode190Result.Length() != row.want {
				t.Fatalf("length%d want%d", utf8Decode190Result.Length(), row.want)
			}
			t.Logf("UTF8 decode allocations/call=%.0f maximum=2", allocations)
			if allocations > 2 {
				t.Fatalf("UTF8 decode allocations %.0f > 2", allocations)
			}
		})
	}
}
func TestUTF8Decode190UnitsAndOwnership(t *testing.T) {
	for _, row := range []struct {
		raw  []byte
		want []uint16
	}{{nil, nil}, {[]byte{0, 127, 0xc2, 0x80, 0xdf, 0xbf, 0xe0, 0xa0, 0x80, 0xef, 0xbf, 0xbf}, []uint16{0, 127, 128, 2047, 2048, 65535}}, {[]byte{0xf0, 0x90, 0x80, 0x80, 0xf4, 0x8f, 0xbf, 0xbf}, []uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}}, {[]byte{0xed, 0xa0, 0x80}, []uint16{0xfffd}}, {[]byte{0xed, 0xbf, 0xbf}, []uint16{0xfffd}}, {[]byte{0xe0, 0x80, 0x80}, []uint16{0xfffd, 0xfffd, 0xfffd}}, {[]byte{0xf0, 0x9f, 65, 0x80}, []uint16{0xfffd, 65, 0xfffd}}, {[]byte{0xe1, 0x80}, []uint16{0xfffd}}, {[]byte{0xf0, 0x9f, 0x98}, []uint16{0xfffd}}, {[]byte{0xff, 0xc0, 0xaf}, []uint16{0xfffd, 0xfffd, 0xfffd}}} {
		input := signedByteArray(row.raw)
		a, b := JavaStringFromBytes(input, UTF_8), JavaStringFromBytes(input, UTF_8)
		if a == b || !slices.Equal(a.UTF16Copy(), row.want) || !a.Equals(b) {
			t.Fatalf("input%x got%x want%x", row.raw, a.UTF16Copy(), row.want)
		}
		if len(row.want) > 0 && &a.units[0] == &b.units[0] {
			t.Fatal("results share storage")
		}
		clear(input.Elements)
		exposed := a.UTF16Copy()
		clear(exposed)
		if !slices.Equal(a.UTF16Copy(), row.want) || !a.Equals(b) {
			t.Fatal("input/output alias")
		}
		if len(row.want) == 0 && a == JavaStringLiteralUTF16(nil) {
			t.Fatal("empty interned")
		}
	}
	input := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 65)
	charsetEncodingReferenceFailure(t, "UnsupportedOperationException", func() any { return JavaStringFromBytes(input, &Charset{name: "UTF-8"}) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringFromBytes(nil, nil) })
}
