package stdjava

import "testing"

func TestSingleByteEncode188CheckedLength(t *testing.T) {
	for _, row := range []struct {
		units []uint16
		want  int
	}{
		{nil, 0}, {[]uint16{'A'}, 1}, {[]uint16{0xd800}, 1}, {[]uint16{0xdc00}, 1}, {[]uint16{0xd800, 0xdc00}, 1}, {[]uint16{0xdbff, 0xdfff}, 1}, {[]uint16{0xd800, 'A', 0xdc00}, 3}, {[]uint16{0xd800, 0xd801, 0xdc00, 0xdc01}, 3},
	} {
		for _, limit := range []int{-1, 0, 1, 2, 3, 4, 2147483647, int(^uint(0) >> 1)} {
			n, ok := javaSingleByteOwnedLength(row.units, limit)
			if ok != (limit >= row.want) || (ok && n != row.want) || (!ok && n != 0) {
				t.Fatalf("units%x limit%d got(%d,%t), want%d", row.units, limit, n, ok, row.want)
			}
		}
	}
	// Output-length refusal uses a small bound, never a huge allocation attempt.
	units := []uint16{0xd800, 0xdc00, 'A'}
	if n, ok := javaSingleByteOwnedLength(units, 1); ok || n != 0 {
		t.Fatal("over-limit result must select legacy fallback")
	}
	if n, ok := javaSingleByteOwnedLength(units, 2); !ok || n != 2 {
		t.Fatal("surrogate pair counts as one byte")
	}
}
