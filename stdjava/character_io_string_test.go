package stdjava

import (
	"slices"
	"testing"
)

type writerStringProbe struct {
	*WriterBase
	units   []rune
	buffers []*PrimitiveArray[rune]
	held    []bool
	fail    bool
}

func (w *writerStringProbe) JavaWriterState() *WriterBase { return w.WriterBase }
func (w *writerStringProbe) JavaWriterWriteChars(execution *Execution, chars *PrimitiveArray[rune], off, length int32) {
	w.units = append(w.units, chars.Elements[off:off+length]...)
	w.buffers = append(w.buffers, chars)
	w.held = append(w.held, ThreadHoldsLockExecution(execution, w.Lock))
	if w.fail {
		panic(NewIllegalStateException("callback"))
	}
}

type writerStringOverrideProbe struct {
	*writerStringProbe
	strings []*JavaString
	ranges  []*JavaString
	bounds  [][2]int32
}

func (w *writerStringOverrideProbe) JavaWriterWriteString(_ *Execution, value *JavaString) {
	w.strings = append(w.strings, value)
}
func (w *writerStringOverrideProbe) JavaWriterWriteStringRange(_ *Execution, value *JavaString, off, length int32) {
	w.ranges = append(w.ranges, value)
	w.bounds = append(w.bounds, [2]int32{off, length})
}

func TestWriterCanonicalStringDefaultsUTF16AndMonitor(t *testing.T) {
	execution := NewExecution()
	lock := &struct{ marker int }{}
	w := &writerStringProbe{WriterBase: NewWriterBase(lock)}
	text := NewJavaStringUTF16([]uint16{'A', 0xd800, 0xdc00, 0xdfff, 'Z'})
	WriterWriteExecution(execution, w, text)
	WriterWriteExecution(execution, w, text, 1, 3)
	WriterWriteExecution(execution, w, int32(0x1f600))
	want := []rune{'A', 0xd800, 0xdc00, 0xdfff, 'Z', 0xd800, 0xdc00, 0xdfff, 0xf600}
	if !slices.Equal(w.units, want) {
		t.Fatalf("units = %x, want %x", w.units, want)
	}
	for i, buffer := range w.buffers {
		if buffer != w.buffers[0] || len(buffer.Elements) != 1024 || !w.held[i] {
			t.Fatalf("callback %d lost cached buffer or writer lock", i)
		}
	}
	large := NewJavaStringUTF16(make([]uint16, 1025))
	WriterWriteExecution(execution, w, large)
	WriterWriteExecution(execution, w, text)
	if len(w.buffers[3].Elements) != 1025 || w.buffers[3] == w.buffers[0] || w.buffers[4] != w.buffers[0] {
		t.Fatal("large write replaced the cached small buffer")
	}
	w.fail = true
	func() {
		defer func() {
			if _, ok := recover().(IllegalStateException); !ok {
				t.Fatal("callback exception was not propagated")
			}
		}()
		WriterWriteExecution(execution, w, text)
	}()
	if ThreadHoldsLockExecution(execution, lock) {
		t.Fatal("writer lock retained after callback panic")
	}
}

func TestWriterCanonicalStringVirtualDispatchAndNull(t *testing.T) {
	execution := NewExecution()
	w := &writerStringOverrideProbe{writerStringProbe: &writerStringProbe{WriterBase: NewWriterBase(&struct{ marker int }{})}}
	text := NewJavaStringUTF16([]uint16{0xd800, 'x'})
	WriterWriteExecution(execution, w, text)
	WriterWriteExecution(execution, w, (*JavaString)(nil))
	WriterWriteExecution(execution, w, text, -3, -4)
	WriterWriteExecution(execution, w, (*JavaString)(nil), -7, -9)
	if len(w.strings) != 2 || w.strings[0] != text || w.strings[1] != nil {
		t.Fatal("String override lost reference identity or null")
	}
	if len(w.ranges) != 2 || w.ranges[0] != text || w.ranges[1] != nil || w.bounds[0] != [2]int32{-3, -4} || w.bounds[1] != [2]int32{-7, -9} {
		t.Fatal("range override did not observe unvalidated arguments")
	}
	WriterWriteDefaultExecution(execution, w, text)
	if len(w.strings) != 2 || len(w.ranges) != 3 || w.ranges[2] != text || w.bounds[2] != [2]int32{0, 2} {
		t.Fatal("super.write(String) did not retain virtual range dispatch")
	}
	WriterWriteDefaultExecution(execution, w, text, 0, 1)
	if len(w.ranges) != 3 || !slices.Equal(w.units, []rune{0xd800}) {
		t.Fatal("super.write(String,int,int) did not bypass range override with exact UTF16")
	}
}

func TestWriterCanonicalStringDefaultBoundsAndNull(t *testing.T) {
	execution := NewExecution()
	w := &writerStringProbe{WriterBase: NewWriterBase(&struct{ marker int }{})}
	text := NewJavaStringUTF16([]uint16{0xd800, 'x'})
	for _, bounds := range [][2]int32{{-1, 1}, {0, -1}, {0, 3}, {2147483647, 1}} {
		func() {
			defer func() {
				if _, ok := recover().(StringIndexOutOfBoundsException); !ok {
					t.Fatalf("bounds %v did not throw StringIndexOutOfBoundsException", bounds)
				}
			}()
			WriterWriteExecution(execution, w, text, bounds[0], bounds[1])
		}()
		if ThreadHoldsLockExecution(execution, w.Lock) {
			t.Fatalf("bounds %v retained writer lock", bounds)
		}
	}
	for _, bounds := range [][]int32{nil, {0, 0}} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Fatalf("null String with bounds %v did not throw NullPointerException", bounds)
				}
			}()
			WriterWriteExecution(execution, w, (*JavaString)(nil), bounds...)
		}()
		if ThreadHoldsLockExecution(execution, w.Lock) {
			t.Fatal("null String retained writer lock")
		}
	}
	if len(w.buffers) != 0 {
		t.Fatal("invalid default write reached char[] callback")
	}
	WriterWriteExecution(execution, w, text, 2, 0)
	if len(w.buffers) != 1 || !w.held[0] || len(w.units) != 0 {
		t.Fatal("empty valid range did not reach synchronized callback")
	}
}

func TestWriterNativeStringAPIsRemainUsable(t *testing.T) {
	execution := NewExecution()
	w := &writerStringProbe{WriterBase: NewWriterBase(&struct{ marker int }{})}
	WriterWriteExecution(execution, w, "A😀Z", 1, 2)
	WriterWriteStringRangeExecution(execution, w, "Q", 0, 1)
	WriterWriteStringRangeDefaultExecution(execution, w, "R", 0, 1)
	if !slices.Equal(w.units, []rune{0xd83d, 0xde00, 'Q', 'R'}) {
		t.Fatalf("native Writer APIs produced %x", w.units)
	}
}

type writerStringSequence struct{ text, sub *JavaString }

func (s *writerStringSequence) Length() int32                                 { return 2 }
func (s *writerStringSequence) CharAt(int32) rune                             { return 'x' }
func (s *writerStringSequence) StringJava2goExecution(*Execution) *JavaString { return s.text }
func (s *writerStringSequence) SubSequenceJava2goExecution(*Execution, int32, int32) any {
	return s.sub
}

func TestWriterAppendCanonicalStringConversion(t *testing.T) {
	execution := NewExecution()
	w := &writerStringOverrideProbe{writerStringProbe: &writerStringProbe{WriterBase: NewWriterBase(&struct{ marker int }{})}}
	text := NewJavaStringUTF16([]uint16{0xd800, 'x'})
	sub := NewJavaStringUTF16([]uint16{0xdfff})
	sequence := &writerStringSequence{text: text, sub: sub}
	if WriterAppendExecution(execution, w, text) != w || WriterAppendExecution(execution, w, sequence) != w || WriterAppendExecution(execution, w, sequence, 0, 1) != w || WriterAppendExecution(execution, w, text, 0, 1) != w || WriterAppendExecution(execution, w, (*JavaString)(nil)) != w {
		t.Fatal("append did not return writer")
	}
	if len(w.strings) != 5 || w.strings[0] != text || w.strings[1] != text || w.strings[2] != sub || !slices.Equal(w.strings[3].UTF16Copy(), []uint16{0xd800}) || !slices.Equal(w.strings[4].UTF16Copy(), []uint16{'n', 'u', 'l', 'l'}) {
		t.Fatal("append lost canonical String identity, UTF16, virtual subsequence, or null conversion")
	}
}
