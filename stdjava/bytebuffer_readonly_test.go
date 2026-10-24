package stdjava

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func readonlyPanic(t *testing.T, name string, action func()) any {
	t.Helper()
	var result any
	func() { defer func() { result = recover() }(); action() }()
	failure, ok := result.(Throwable)
	if !ok || failure.ThrowableTypeName() != name {
		t.Fatalf("want %s, recovered %#v", name, result)
	}
	return result
}
func TestReadonlyHeapMutationsAndOrder(t *testing.T) {
	for _, caseValue := range []struct {
		name   string
		invoke func(*ByteBuffer)
	}{
		{"scalar", func(b *ByteBuffer) { b.Put(2) }}, {"indexed", func(b *ByteBuffer) { b.PutAt(-1, 2) }},
		{"range-null", func(b *ByteBuffer) { b.PutArray(nil, -1, -1) }}, {"buffer-self", func(b *ByteBuffer) { b.PutBuffer(b) }},
		{"absolute-array", func(b *ByteBuffer) { b.PutArrayAt(-1, nil, -1, -1) }}, {"absolute-buffer", func(b *ByteBuffer) { b.PutBufferAt(-1, nil, -1, -1) }},
		{"compact", func(b *ByteBuffer) { b.Compact() }}, {"array", func(b *ByteBuffer) { b.Array() }}, {"offset", func(b *ByteBuffer) { b.ArrayOffset() }},
		{"char", func(b *ByteBuffer) { b.PutChar(65535) }}, {"char-index", func(b *ByteBuffer) { b.PutCharAt(-1, 65535) }},
		{"short", func(b *ByteBuffer) { b.PutShort(2) }}, {"short-index", func(b *ByteBuffer) { b.PutShortAt(-1, 2) }},
		{"int", func(b *ByteBuffer) { b.PutInt(2) }}, {"int-index", func(b *ByteBuffer) { b.PutIntAt(-1, 2) }},
		{"long", func(b *ByteBuffer) { b.PutLong(2) }}, {"long-index", func(b *ByteBuffer) { b.PutLongAt(-1, 2) }},
		{"float", func(b *ByteBuffer) { b.PutFloat(2) }}, {"float-index", func(b *ByteBuffer) { b.PutFloatAt(-1, 2) }},
		{"double", func(b *ByteBuffer) { b.PutDouble(2) }}, {"double-index", func(b *ByteBuffer) { b.PutDoubleAt(-1, 2) }},
	} {
		t.Run(caseValue.name, func(t *testing.T) {
			array := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 0, 1, 2, 3)
			b := ByteBufferWrap(array).SetPosition(1).Mark().AsReadOnlyBuffer()
			before := slices.Clone(array.Elements)
			failure := readonlyPanic(t, "ReadOnlyBufferException", func() { caseValue.invoke(b) })
			if JavaThrowableMessageExecution(NewExecution(), failure) != nil {
				t.Fatal("RO exception message must be Java null")
			}
			if !slices.Equal(before, array.Elements) || b.position != 1 || b.limit != 4 || b.mark != 1 {
				t.Fatal("failed operation changed buffer")
			}
			if !isSubtypeOf("ReadOnlyBufferException", "UnsupportedOperationException") {
				t.Fatal("missing catch ancestry")
			}
		})
	}
	b := ByteBufferAllocate(0).AsReadOnlyBuffer()
	readonlyPanic(t, "NullPointerException", func() { b.PutArray(nil) })
	readonlyPanic(t, "NullPointerException", func() { b.PutArrayAtWhole(-1, nil) })
	readonlyPanic(t, "ReadOnlyBufferException", func() { b.PutArray(NewPrimitiveArray[int8](0, PrimitiveByteTypeID)) })
}
func TestReadonlyHeapViewAliasesAndBulk(t *testing.T) {
	a := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 0, 1, 2, 3, 4, 5)
	m := ByteBufferWrap(a, 1, 4).Mark().SetOrder(ByteOrderLITTLE_ENDIAN)
	r := m.AsReadOnlyBuffer()
	d := r.Duplicate()
	s := r.Slice()
	r2 := r.AsReadOnlyBuffer()
	if r == r2 || r == m || r.HasArray() || !r.IsReadOnly() || !d.IsReadOnly() || !s.IsReadOnly() || r.Order() != ByteOrderBIG_ENDIAN || d.Order() != ByteOrderBIG_ENDIAN || s.Order() != ByteOrderBIG_ENDIAN {
		t.Fatal("view metadata")
	}
	m.PutAt(1, 9)
	if r.Get(1) != 9 || s.Get(0) != 9 || r.position != 1 || d.position != 1 || r.array != a {
		t.Fatal("alias/cursor")
	}
	r.SetPosition(3).Reset()
	if r.position != 1 || d.position != 1 {
		t.Fatal("mark")
	}
	dest := ByteBufferWrap(a)
	dest.SetPosition(2)
	src := dest.AsReadOnlyBuffer().SetPosition(1).SetLimit(4)
	dest.PutBuffer(src)
	if dest.position != 5 || src.position != 4 || !slices.Equal(a.Elements, []int8{0, 9, 9, 2, 3, 5}) {
		t.Fatalf("source copy %v", a.Elements)
	}
	b := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 0, 1, 2, 3, 4, 5)).SetPosition(2)
	b.PutBufferAt(1, b, 0, 4)
	b.PutArrayAt(0, PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 8, 9, 10), 1, 2)
	b.PutArrayAtWhole(4, PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 6, 7))
	if b.position != 2 || !slices.Equal(b.array.Elements, []int8{9, 10, 1, 2, 6, 7}) {
		t.Fatalf("absolute copy %v", b.array.Elements)
	}
	chars := ByteBufferAllocate(4).SetOrder(ByteOrderLITTLE_ENDIAN)
	chars.PutChar(0xD800).PutCharAt(2, 0xFFFF).SetPosition(4).Flip()
	if chars.GetChar() != 0xD800 || chars.GetChar(2) != 0xFFFF || chars.position != 2 {
		t.Fatal("char units")
	}
}
func TestReadonlyFileChannelEntryOrdering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte{1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	f := NewRandomAccessFile(path, "rw")
	c := f.GetChannel()
	e := NewExecution()
	b := ByteBufferAllocate(0).AsReadOnlyBuffer()
	v := readonlyPanic(t, "IllegalArgumentException", func() { FileChannelReadExecution(e, c, b) })
	if !slices.Equal(JavaThrowableMessageExecution(e, v).UTF16Copy(), []uint16{'R', 'e', 'a', 'd', '-', 'o', 'n', 'l', 'y', ' ', 'b', 'u', 'f', 'f', 'e', 'r'}) {
		t.Fatal("readonly consumer message")
	}
	c.Close()
	readonlyPanic(t, "ClosedChannelException", func() { FileChannelReadExecution(e, c, nil) })
	f = NewRandomAccessFile(path, "rw")
	c = f.GetChannel()
	thread := newNamedThread("entry")
	e.thread = thread
	thread.Interrupt()
	v = readonlyPanic(t, "ClosedByInterruptException", func() { FileChannelReadExecution(e, c, nil) })
	if c.IsOpen() || JavaThrowableMessageExecution(e, v) != nil || !thread.interrupted {
		t.Fatal("preinterrupt lifecycle")
	}
	readonlyPanic(t, "IOException", func() { f.GetFilePointer() })
	f = NewRandomAccessFile(path, "rw")
	c = f.GetChannel()
	if err := f.state.file.Close(); err != nil {
		t.Fatal(err)
	}
	v = readonlyPanic(t, "ClosedByInterruptException", func() { FileChannelReadExecution(e, c, b) })
	if c.IsOpen() || JavaThrowableMessageExecution(e, v) != nil {
		t.Fatal("close IOException displaced CBI")
	}
}

func TestReadonlyFileChannelMessageIdentity(t *testing.T) {
	f := NewRandomAccessFile(filepath.Join(t.TempDir(), "file"), "rw")
	defer f.Close()
	c := f.GetChannel()
	e := NewExecution()
	first := readonlyPanic(t, "IllegalArgumentException", func() { FileChannelReadExecution(e, c, ByteBufferAllocate(0).AsReadOnlyBuffer()) })
	second := readonlyPanic(t, "IllegalArgumentException", func() { FileChannelReadExecution(e, c, ByteBufferAllocate(2).AsReadOnlyBuffer()) })
	a, b := JavaThrowableMessageExecution(e, first), JavaThrowableMessageExecution(e, second)
	literal := JavaStringLiteralUTF16([]uint16{'R', 'e', 'a', 'd', '-', 'o', 'n', 'l', 'y', ' ', 'b', 'u', 'f', 'f', 'e', 'r'})
	if a != b || a != literal {
		t.Fatal("JDK read-only consumer message is a shared interned literal")
	}
}
