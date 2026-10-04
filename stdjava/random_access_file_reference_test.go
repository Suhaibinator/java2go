package stdjava

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRAFStringModePathOrdering(t *testing.T) {
	e := NewExecution()
	nilPath := (*JavaString)(nil)
	readonlyPanic(t, "IllegalArgumentException", func() { NewRandomAccessFileStringExecution(e, nilPath, JavaStringFromHostUTF8("bad")) })
	v := readonlyPanic(t, "NullPointerException", func() { NewRandomAccessFileStringExecution(e, nilPath, JavaStringFromHostUTF8("r")) })
	if JavaThrowableMessageExecution(e, v) != nil {
		t.Fatal("valid mode/null path message must be Java null")
	}
	v = readonlyPanic(t, "NullPointerException", func() { NewRandomAccessFileStringExecution(e, nilPath, nil) })
	if !JavaThrowableMessageExecution(e, v).Equals(JavaStringFromHostUTF8("Cannot invoke \"String.equals(Object)\" because \"mode\" is null")) {
		t.Fatal("null mode helpful message")
	}
	nul := NewJavaStringUTF16([]uint16{'a', 0, 'b'})
	v = readonlyPanic(t, "FileNotFoundException", func() { NewRandomAccessFileStringExecution(e, nul, JavaStringFromHostUTF8("rw")) })
	literal := JavaStringLiteralUTF16([]uint16{'I', 'n', 'v', 'a', 'l', 'i', 'd', ' ', 'f', 'i', 'l', 'e', ' ', 'p', 'a', 't', 'h'})
	if JavaThrowableMessageExecution(e, v) != literal {
		t.Fatal("invalid-path interned literal")
	}
	for _, mode := range [][]uint16{{}, {'x'}, {'r', 'w', 'w'}, {'R', 'W'}, {'r', 0}, {0xD83D, 0xDE00}, {0xD800}} {
		v = readonlyPanic(t, "IllegalArgumentException", func() { NewRandomAccessFileStringExecution(e, nul, NewJavaStringUTF16(mode)) })
		msg := JavaThrowableMessageExecution(e, v).UTF16Copy()
		prefix := JavaStringFromHostUTF8("Illegal mode \"").UTF16Copy()
		suffix := JavaStringFromHostUTF8("\" must be one of \"r\", \"rw\", \"rws\", or \"rwd\"").UTF16Copy()
		if !slices.Equal(msg[:len(prefix)], prefix) || !slices.Equal(msg[len(prefix):len(prefix)+len(mode)], mode) || !slices.Equal(msg[len(prefix)+len(mode):], suffix) {
			t.Fatal("mode lost UTF16 diagnostic")
		}
	}
}
func TestRAFStringNativeIdentityAndResourceCleanup(t *testing.T) {
	e := NewExecution()
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte{1, 2, 3, 4}, 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"r", "rw", "rws", "rwd"} {
		f := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), CopyJavaString(JavaStringFromHostUTF8(mode)))
		firstChannel, secondChannel := f.GetChannel(), f.GetChannel()
		if f.Length() != 4 || firstChannel == nil || firstChannel != secondChannel {
			t.Fatal("state/channel identity")
		}
		f.SeekPosition(2)
		if f.ReadByteValue() != 3 || f.GetFilePointer() != 3 {
			t.Fatal("cursor")
		}
		f.GetChannel().Close()
		readonlyPanic(t, "IOException", func() { f.GetFilePointer() })
		f.Close()
	}
	opened, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	readonlyPanic(t, "FileNotFoundException", func() { randomAccessFileReferenceFinishOpen(NewJavaFileReference(JavaStringFromHostUTF8(dir)), opened) })
	if _, err = opened.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("directory descriptor was not closed: %v", err)
	}
	opened, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	readonlyPanic(t, "FileNotFoundException", func() {
		randomAccessFileReferenceFinishOpen(NewJavaFileReference(JavaStringFromHostUTF8(path)), opened)
	})
	surrogate := NewJavaStringUTF16(append(JavaStringFromHostUTF8(filepath.Join(dir, "missing-")).UTF16Copy(), 0xD800))
	v := readonlyPanic(t, "FileNotFoundException", func() { NewRandomAccessFileStringExecution(e, surrogate, JavaStringFromHostUTF8("r")) })
	units := JavaThrowableMessageExecution(e, v).UTF16Copy()
	if !slices.Equal(units[:surrogate.Length()], surrogate.units) {
		t.Fatal("native failure lost original path UTF16")
	}
	device := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8("/dev/null"), JavaStringFromHostUTF8("r"))
	device.Close()
}
