package stdjava

import (
	"crypto/sha256"
	"slices"
	"testing"
)

func TestNIOHeapCompactStatefulContract(t *testing.T) {
	a := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(91), int8(92), int8(1), int8(2), int8(3), int8(4), int8(5), int8(6), int8(7), int8(8), int8(93), int8(94))
	root := ByteBufferWrap(a, 2, 8).SetOrder(ByteOrderLITTLE_ENDIAN)
	view := root.Slice().SetOrder(ByteOrderLITTLE_ENDIAN).SetPosition(2).Mark().SetLimit(7)
	sibling := view.Duplicate().SetOrder(ByteOrderLITTLE_ENDIAN).SetPosition(1).Mark()
	if view.Compact() != view || view.Array() != a || view.Order() != ByteOrderLITTLE_ENDIAN {
		t.Fatal("compact lost identity/backing/order")
	}
	if !slices.Equal(a.Elements, []int8{91, 92, 3, 4, 5, 6, 7, 6, 7, 8, 93, 94}) {
		t.Fatalf("overlap/backing %v", a.Elements)
	}
	if view.Position() != 5 || view.Limit() != 8 || view.Capacity() != 8 || view.ArrayOffset() != 2 || root.Position() != 2 || root.Limit() != 10 || sibling.Position() != 1 || sibling.Limit() != 7 {
		t.Fatal("compact changed independent view cursor or window")
	}
	expectBoxedException(t, "InvalidMarkException", func() { view.Reset() })
	sibling.Reset()
	if view.GetShort(0) != 1027 {
		t.Fatal("compact changed endianness")
	}
	view.Put(11).Put(12).Flip()
	d := JavaMessageDigestGetInstance(JavaStringLiteralUTF16([]uint16{'S', 'H', 'A', '-', '2', '5', '6'}))
	d.UpdateReference(view)
	got := d.Digest()
	want := sha256.Sum256([]byte{3, 4, 5, 6, 7, 11, 12})
	for i, b := range want {
		if got.Elements[i] != int8(b) {
			t.Fatalf("digest byte %d differs", i)
		}
	}
	if view.Position() != 7 || view.Limit() != 7 {
		t.Fatal("digest consumption after compact/append/flip")
	}
	nested := sibling.SetPosition(2).SetLimit(6).Slice().SetPosition(1).Mark()
	nested.Compact()
	if !slices.Equal(a.Elements, []int8{91, 92, 3, 4, 6, 7, 11, 11, 12, 8, 93, 94}) || nested.Position() != 3 || nested.Limit() != 4 || nested.ArrayOffset() != 4 || sibling.Position() != 2 || sibling.Limit() != 6 {
		t.Fatalf("nested view compact: %v", a.Elements)
	}
	expectBoxedException(t, "InvalidMarkException", func() { nested.Reset() })
}
func TestNIOHeapCompactEmptyAndNullContract(t *testing.T) {
	for _, b := range []*ByteBuffer{ByteBufferAllocate(4).SetPosition(4).Mark(), ByteBufferAllocate(0).Mark()} {
		b.SetOrder(ByteOrderLITTLE_ENDIAN)
		a := b.Array()
		before := slices.Clone(a.Elements)
		if b.Compact() != b || b.Position() != 0 || b.Limit() != b.Capacity() || b.Order() != ByteOrderLITTLE_ENDIAN || b.Array() != a || !slices.Equal(before, a.Elements) {
			t.Fatal("empty compact")
		}
		expectBoxedException(t, "InvalidMarkException", func() { b.Reset() })
	}
	b := ByteBufferWrap(PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(17), int8(18), int8(19))).Mark()
	b.Compact()
	if b.Position() != 3 || b.Limit() != 3 || !slices.Equal(b.Array().Elements, []int8{17, 18, 19}) {
		t.Fatal("full remaining compact")
	}
	expectBoxedException(t, "InvalidMarkException", func() { b.Reset() })
	expectBoxedException(t, "NullPointerException", func() { var missing *ByteBuffer; missing.Compact() })
}
