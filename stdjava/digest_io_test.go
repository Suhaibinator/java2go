package stdjava

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDigestIOReadRangesAndEOF(t *testing.T) {
	stream := NewByteArrayInputStream([]byte{0, 127, 128, 255})
	out := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(9), int8(9), int8(9), int8(9), int8(9), int8(9))
	if got := InputStreamReadInto(stream, out, 1, 4); got != 4 {
		t.Fatalf("count=%d", got)
	}
	if got := out.Elements; got[0] != 9 || got[1] != 0 || got[2] != 127 || got[3] != -128 || got[4] != -1 || got[5] != 9 {
		t.Fatalf("incorrect signed-byte copy/range: %v", got)
	}
	if got := InputStreamReadInto(stream, out, 0, 0); got != 0 {
		t.Fatalf("empty read at EOF=%d", got)
	}
	if got := InputStreamReadInto(stream, out, 0, 1); got != -1 {
		t.Fatalf("EOF=%d", got)
	}
	assertDigestIOPanic(t, "IndexOutOfBoundsException", func() { InputStreamReadInto(stream, out, 5, 2) })
	assertDigestIOPanic(t, "NullPointerException", func() { InputStreamReadInto(stream, nil) })
}
func TestDigestIOChannelSharesCursorAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bytes")
	if err := os.WriteFile(path, []byte{1, 2, 3, 4, 5}, 0600); err != nil {
		t.Fatal(err)
	}
	random := NewRandomAccessFile(NewJavaFile(path), "r")
	channel := random.GetChannel()
	random.Seek(1)
	buffer := ByteBufferAllocate(3)
	if got := channel.Read(buffer); got != 3 {
		t.Fatalf("count=%d", got)
	}
	if got := random.GetFilePointer(); got != 4 {
		t.Fatalf("shared position=%d", got)
	}
	buffer.Flip()
	out := NewPrimitiveArray[int8](3, PrimitiveByteTypeID)
	buffer.GetInto(out)
	if out.Elements[0] != 2 || out.Elements[2] != 4 {
		t.Fatalf("buffer=%v", out.Elements)
	}
	buffer.Clear()
	if got := channel.Read(buffer); got != 1 {
		t.Fatalf("remaining read=%d", got)
	}
	buffer.Clear()
	if got := channel.Read(buffer); got != -1 {
		t.Fatalf("EOF=%d", got)
	}
	channel.Close()
	if channel.IsOpen() {
		t.Fatal("channel stayed open")
	}
	assertDigestIOPanic(t, "IOException", func() { random.Seek(0) })
	random.Close()
}
func TestDigestIOBufferedCloseAndOpenOptions(t *testing.T) {
	path := NewJavaPath(filepath.Join(t.TempDir(), "input"))
	FilesWrite(path, []byte{1, 2, 3})
	options := ReferenceArrayLiteralOf[OpenOption]("OpenOption", StandardOpenOptionRead)
	input := FilesNewInputStream(path, options)
	buffered := NewBufferedInputStream(input, 2)
	if got := InputStreamReadByte(buffered); got != 1 {
		t.Fatalf("first=%d", got)
	}
	buffered.Close()
	buffered.Close()
	assertDigestIOPanic(t, "IOException", func() { InputStreamReadByte(buffered) })
	assertDigestIOPanic(t, "IllegalArgumentException", func() { FilesNewInputStream(path, StandardOpenOptionWrite) })
	assertDigestIOPanic(t, "NullPointerException", func() { FilesNewInputStream(path, (*ReferenceArray)(nil)) })
	deleting := FilesNewInputStream(path, StandardOpenOptionDeleteOnClose)
	deleting.Close()
	if _, err := os.Stat(path.path); !os.IsNotExist(err) {
		t.Fatalf("DELETE_ON_CLOSE left path: %v", err)
	}
}
func assertDigestIOPanic(t *testing.T, kind string, call func()) {
	t.Helper()
	defer func() {
		if got := recover(); !CaughtAs(got, kind) {
			t.Fatalf("got %v, want %s", got, kind)
		}
	}()
	call()
}
