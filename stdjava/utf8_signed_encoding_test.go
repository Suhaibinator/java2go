package stdjava

import (
	"runtime"
	"slices"
	"testing"
)

func TestUTF8OwnedEncodingCheckedLength(t *testing.T) {
	cases := []struct {
		name  string
		units []uint16
		want  int
	}{
		{"empty", nil, 0}, {"nul", []uint16{0}, 1}, {"ascii-edge", []uint16{0x7f}, 1},
		{"two-min", []uint16{0x80}, 2}, {"two-max", []uint16{0x7ff}, 2},
		{"three-min", []uint16{0x800}, 3}, {"three-max", []uint16{0xffff}, 3},
		{"high", []uint16{0xd800}, 1}, {"low", []uint16{0xdc00}, 1},
		{"pair-min", []uint16{0xd800, 0xdc00}, 4}, {"pair-max", []uint16{0xdbff, 0xdfff}, 4},
		{"high-high-low-low", []uint16{0xd800, 0xd801, 0xdc00, 0xdc01}, 6},
		{"pair-at-end", []uint16{'A', 0x80, 0xd83d, 0xde00}, 7},
		{"mixed", []uint16{0, 0x7ff, 0x800, 0xd800, 'B', 0xdc00, 0xdbff, 0xdfff}, 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, limit := range []int{-1, 0, 1, 2, 3, 4, 5, 6, 7, 12, 13, int(^uint(0) >> 1)} {
				got, ok := javaUTF8OwnedLength(tc.units, limit)
				wantOK := limit >= tc.want
				if ok != wantOK || (ok && got != tc.want) || (!ok && got != 0) {
					t.Fatalf("limit%d got(%d,%t), want length%d ok%t", limit, got, ok, tc.want, wantOK)
				}
			}
			encoded, ok := encodeJavaUTF8OwnedSigned(tc.units)
			if !ok || len(encoded) != tc.want {
				t.Fatalf("encoded length=%d ok=%t want%d", len(encoded), ok, tc.want)
			}
		})
	}
	byteCases := []struct {
		units []uint16
		want  []byte
	}{
		{[]uint16{0, 0x7f, 0x80, 0x7ff, 0x800, 0xffff}, []byte{0, 0x7f, 0xc2, 0x80, 0xdf, 0xbf, 0xe0, 0xa0, 0x80, 0xef, 0xbf, 0xbf}},
		{[]uint16{0xd800, 0xdc00}, []byte{0xf0, 0x90, 0x80, 0x80}},
		{[]uint16{0xdbff, 0xdfff}, []byte{0xf4, 0x8f, 0xbf, 0xbf}},
		{[]uint16{0xd800, 0xd801, 0xdc00, 0xdc01}, []byte{'?', 0xf0, 0x90, 0x90, 0x80, '?'}},
	}
	for _, tc := range byteCases {
		got := JavaStringGetBytes(NewJavaStringUTF16(tc.units), UTF_8)
		if len(got.Elements) != len(tc.want) {
			t.Fatalf("units%x byte length%d want%d", tc.units, len(got.Elements), len(tc.want))
		}
		for index, b := range tc.want {
			if got.Elements[index] != int8(b) {
				t.Fatalf("units%x byte%d=%d want%d", tc.units, index, got.Elements[index], int8(b))
			}
		}
	}
	for value := 0; value < 65536; value++ {
		want := 1
		switch {
		case value < 0x80:
		case value < 0x800:
			want = 2
		case value >= 0xd800 && value <= 0xdfff:
		default:
			want = 3
		}
		units := []uint16{uint16(value)}
		for _, limit := range []int{want - 1, want, int(^uint(0) >> 1)} {
			got, ok := javaUTF8OwnedLength(units, limit)
			if ok != (limit >= want) || (ok && got != want) || (!ok && got != 0) {
				t.Fatalf("unit%04x limit%d length%d ok%t want%d", value, limit, got, ok, want)
			}
		}
	}
	for _, high := range []uint16{0xd800, 0xd801, 0xdbfe, 0xdbff} {
		for _, low := range []uint16{0xdc00, 0xdc01, 0xdffe, 0xdfff} {
			for _, limit := range []int{0, 3, 4, 5, int(^uint(0) >> 1)} {
				n, ok := javaUTF8OwnedLength([]uint16{high, low}, limit)
				if ok != (limit >= 4) || (ok && n != 4) || (!ok && n != 0) {
					t.Fatalf("pair%04x/%04x limit%d got(%d,%t)", high, low, limit, n, ok)
				}
			}
		}
	}
}

func TestUTF8OwnedEncodingRetainedAcrossGC(t *testing.T) {
	value := NewJavaStringUTF16([]uint16{0, 'A', 0xe9, 0x65e5, 0xd83d, 0xde00, 0xd800, 'B', 0xdc00})
	original := value.UTF16Copy()
	var retained []*PrimitiveArray[int8]
	var saved [][]int8
	for range 8 {
		result := JavaStringGetBytes(value, UTF_8)
		if result.ComponentType() != PrimitiveByteTypeID {
			t.Fatal("byte descriptor lost")
		}
		for _, prior := range retained {
			if result == prior {
				t.Fatal("fresh output object reused")
			}
		}
		retained = append(retained, result)
		saved = append(saved, append([]int8(nil), result.Elements...))
	}
	retained[0].Elements[0] = 127
	for range 4 {
		runtime.GC()
		for range 32 {
			temporary := JavaStringGetBytes(value, UTF_8)
			temporary.Elements[0] = 42
		}
	}
	for index := 1; index < len(retained); index++ {
		if !slices.Equal(retained[index].Elements, saved[index]) {
			t.Fatalf("retained output%d changed", index)
		}
	}
	if !slices.Equal(value.UTF16Copy(), original) || !slices.Equal(JavaStringGetBytes(value, UTF_8).Elements, saved[1]) {
		t.Fatal("output mutation reached immutable input/later encoding")
	}
	runtime.KeepAlive(retained)
}
