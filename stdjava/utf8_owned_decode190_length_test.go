package stdjava

import (
	"slices"
	"testing"
)

func TestUTF8Decode190CheckedLength(t *testing.T) {
	for _, row := range []struct {
		raw  []byte
		want []uint16
	}{
		{nil, nil}, {[]byte{0, 127}, []uint16{0, 127}},
		{[]byte{0xc2, 0x80, 0xdf, 0xbf}, []uint16{128, 2047}},
		{[]byte{0xe0, 0xa0, 0x80, 0xef, 0xbf, 0xbf}, []uint16{2048, 65535}},
		{[]byte{0xf0, 0x90, 0x80, 0x80, 0xf4, 0x8f, 0xbf, 0xbf}, []uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}},
		{[]byte{0xed, 0xa0, 0x80}, []uint16{0xfffd}},
		{[]byte{0xf0, 0x9f, 65, 0x80}, []uint16{0xfffd, 65, 0xfffd}},
		{[]byte{0xf0, 0x9f, 0x98}, []uint16{0xfffd}},
		{[]byte{0xf4, 0x90, 0x80, 0x80}, []uint16{0xfffd, 0xfffd, 0xfffd, 0xfffd}},
	} {
		input := signedByteArray(row.raw)
		for _, limit := range []int{-1, 0, 1, 2, 3, 4, len(row.want) - 1, len(row.want), int(^uint(0) >> 1)} {
			got, ok := javaUTF8OwnedDecodeLength(input.Elements, limit)
			wantOK := limit >= len(row.want)
			if ok != wantOK || (ok && got != len(row.want)) || (!ok && got != 0) {
				t.Fatalf("input%x limit%d got(%d,%t) want length%d ok%t", row.raw, limit, got, ok, len(row.want), wantOK)
			}
		}
		a, ok := decodeJavaUTF8OwnedSigned(input.Elements)
		if !ok || !slices.Equal(a.UTF16Copy(), row.want) {
			t.Fatalf("input%x got%x ok%t want%x", row.raw, a.UTF16Copy(), ok, row.want)
		}
	}
}
