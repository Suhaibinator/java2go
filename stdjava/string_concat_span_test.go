package stdjava

import (
	"reflect"
	"slices"
	"strconv"
	"sync"
	"testing"
)

func TestJavaStringConcatSpanSemantics(t *testing.T) {
	t.Run("UTF16-fold-and-identity", func(t *testing.T) {
		rows := [][][]uint16{
			{nil, nil}, {nil, nil, nil}, {{'x'}, nil, nil}, {nil, {'x'}, nil},
			{{'A', 0, 0xD800}, {'|', 0xDC00}, {0xD801}, {0xDC28, 0}},
			{{0xD800}, {0xDC00}, {0xDFFF}}, {{0xDFFF}, {0xD800}, {0}},
		}
		for _, units := range rows {
			parts := make([]*JavaString, len(units))
			for index := range units {
				parts[index] = NewJavaStringUTF16(units[index])
			}
			want := concatJavaStringSpanFold(parts[0], parts[1], parts[2:])
			got := ConcatJavaStringSpan(parts[0], parts[1], parts[2:]...)
			if got == nil || !slices.Equal(got.units, want.units) || !got.Equals(want) || got.HashCode() != want.HashCode() || got.Length() != want.Length() {
				t.Fatal("span differs from original binary fold")
			}
			for _, part := range parts {
				if got == part {
					t.Fatal("input wrapper reused")
				}
			}
			for index, part := range parts {
				if !slices.Equal(part.units, units[index]) {
					t.Fatal("input payload changed")
				}
			}
			for index, unit := range got.units {
				if got.CharAt(int32(index)) != rune(unit) {
					t.Fatal("raw UTF16 unit changed")
				}
			}
		}
	})
	t.Run("201-strong-fresh-empty-results", func(t *testing.T) {
		empty := NewJavaStringUTF16(nil)
		var retained [201]*JavaString
		for index := range retained {
			got := ConcatJavaStringSpan(empty, empty, empty)
			if got == empty || got == nil || got.Length() != 0 {
				t.Fatal("empty result not fresh")
			}
			for _, previous := range retained[:index] {
				if got == previous {
					t.Fatal("result wrapper reused")
				}
			}
			retained[index] = got
		}
	})
	t.Run("literal-and-input-immutability", func(t *testing.T) {
		original := []uint16{'A', 0, 0xD800, 0xDC00}
		first := NewJavaStringUTF16(original)
		literal := JavaStringLiteralUTF16([]uint16{'|'})
		got := ConcatJavaStringSpan(first, literal, first)
		original[0] = 'Z'
		if first.CharAt(0) != 'A' || got.CharAt(0) != 'A' || literal != JavaStringLiteralUTF16([]uint16{'|'}) {
			t.Fatal("immutable input or intern identity changed")
		}
		if &got.units[0] == &first.units[0] {
			t.Fatal("fast output did not own its buffer")
		}
	})
	t.Run("nil-original-fold-exceptions", func(t *testing.T) {
		a := NewJavaStringUTF16([]uint16{'a'})
		capture := func(call func()) (value any) { defer func() { value = recover() }(); call(); return nil }
		for position := 0; position < 4; position++ {
			parts := []*JavaString{a, a, a, a}
			parts[position] = nil
			old := capture(func() { concatJavaStringSpanFold(parts[0], parts[1], parts[2:]) })
			got := capture(func() { ConcatJavaStringSpan(parts[0], parts[1], parts[2:]...) })
			if old == nil || got == nil || reflect.TypeOf(old) != reflect.TypeOf(got) {
				t.Fatal("nil fallback changed exception kind")
			}
		}
	})
	t.Run("checked-int32-boundaries-without-large-allocation", func(t *testing.T) {
		limit := int(^uint32(0) >> 1)
		for _, row := range []struct {
			total, next int
			want        int
			ok          bool
		}{
			{0, 0, 0, true}, {0, 1, 1, true}, {limit - 1, 1, limit, true}, {limit, 0, limit, true},
			{limit, 1, 0, false}, {1, limit, 0, false}, {-1, 0, 0, false}, {0, -1, 0, false},
		} {
			got, ok := checkedConcatUTF16Length(row.total, row.next)
			if got != row.want || ok != row.ok {
				t.Fatalf("checked(%d,%d)=(%d,%t)", row.total, row.next, got, ok)
			}
		}
		if strconv.IntSize == 64 {
			tooLarge := int64(limit) + 1
			wrapsZero := int64(1) << 32
			for _, value := range []int{int(tooLarge), int(wrapsZero), int(^uint(0) >> 1)} {
				if _, ok := checkedConcatUTF16Length(0, value); ok {
					t.Fatal("raw length outside int32 domain admitted")
				}
			}
		}
	})
	t.Run("concurrent-immutable-hash", func(t *testing.T) {
		first := NewJavaStringUTF16([]uint16{'x', 0, 0xD800})
		second := NewJavaStringUTF16([]uint16{0xDC00, 'y'})
		want := ConcatJavaStrings(ConcatJavaStrings(first, second), first)
		var wait sync.WaitGroup
		for worker := 0; worker < 8; worker++ {
			wait.Add(1)
			go func() {
				defer wait.Done()
				for index := 0; index < 32; index++ {
					got := ConcatJavaStringSpan(first, second, first)
					if !got.Equals(want) || got.HashCode() != want.HashCode() {
						t.Error("concurrent immutable result changed")
						return
					}
				}
			}()
		}
		wait.Wait()
	})
}
