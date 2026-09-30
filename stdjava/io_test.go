package stdjava

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileAndWriterReaderRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.txt")

	w := NewPrintWriter(path)
	w.Println("alpha")
	w.Print("be")
	w.Println("ta")
	w.Close()

	f := NewJavaFile(path)
	if !f.Exists() {
		t.Fatalf("File.Exists = false after write")
	}
	if f.GetName() != "data.txt" {
		t.Fatalf("File.GetName = %q, want data.txt", f.GetName())
	}
	if f.IsDirectory() {
		t.Fatalf("File.IsDirectory = true for a file")
	}

	r := NewBufferedReader(path)
	if got := r.ReadLine(); got != "alpha" {
		t.Fatalf("line 1 = %q, want alpha", got)
	}
	if got := r.ReadLine(); got != "beta" {
		t.Fatalf("line 2 = %q, want beta", got)
	}
	r.Close()
}

func TestFileExistsFalseAndDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.txt")
	if NewJavaFile(path).Exists() {
		t.Fatalf("Exists = true for missing file")
	}
	NewPrintWriter(path).Close()
	f := NewJavaFile(path)
	if !f.Exists() {
		t.Fatalf("Exists = false after create")
	}
	if !f.Delete() {
		t.Fatalf("Delete = false")
	}
	if f.Exists() {
		t.Fatalf("Exists = true after delete")
	}
}

func TestBufferedReaderReadLineIntoDistinguishesEmptyLineAndEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.txt")
	w := NewPrintWriter(path)
	w.Println("")
	w.Print("tail")
	w.Close()

	r := NewBufferedReader(path)
	defer r.Close()
	line := "unchanged"
	if ok := r.ReadLineInto(&line); !ok || line != "" {
		t.Fatalf("empty line = %q, ok=%t", line, ok)
	}
	if ok := r.ReadLineInto(&line); !ok || line != "tail" {
		t.Fatalf("unterminated line = %q, ok=%t", line, ok)
	}
	if ok := r.ReadLineInto(&line); ok {
		t.Fatalf("EOF reported another line %q", line)
	}
}

func TestBufferedReaderReadLineOKRecognizesJavaLineTerminators(t *testing.T) {
	path := filepath.Join(t.TempDir(), "terminators.txt")
	if err := os.WriteFile(path, []byte("cr\rcrlf\r\nlf\ntail"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := NewBufferedReader(path)
	defer r.Close()
	for index, want := range []string{"cr", "crlf", "lf", "tail"} {
		got, ok := r.ReadLineOK()
		if !ok || got != want {
			t.Fatalf("line %d = %q, ok=%t; want %q, true", index, got, ok, want)
		}
	}
	if got, ok := r.ReadLineOK(); ok {
		t.Fatalf("EOF returned %q, true", got)
	}
}

func TestScannerFileTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nums.txt")
	w := NewPrintWriter(path)
	w.Println("10 20 hello 3.5")
	w.Close()

	s := NewScannerFile(path)
	if got := s.NextInt(); got != 10 {
		t.Fatalf("NextInt = %d, want 10", got)
	}
	if got := s.NextInt(); got != 20 {
		t.Fatalf("NextInt = %d, want 20", got)
	}
	if got := s.Next(); got != "hello" {
		t.Fatalf("Next = %q, want hello", got)
	}
	if got := s.NextDouble(); got != 3.5 {
		t.Fatalf("NextDouble = %v, want 3.5", got)
	}
	s.Close()
}

func TestPrintWriterAppendKeepsExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "append.txt")

	w := NewPrintWriter(path)
	w.Println("first")
	w.Close()

	a := NewPrintWriterAppend(path, true)
	a.Println("second")
	a.Close()

	r := NewBufferedReader(path)
	defer r.Close()
	for _, want := range []string{"first", "second"} {
		if got := r.ReadLine(); got != want {
			t.Fatalf("line = %q, want %q", got, want)
		}
	}

	// The same constructor with append=false truncates, as `new FileWriter(f, false)` does.
	o := NewPrintWriterAppend(path, false)
	o.Println("only")
	o.Close()
	if got := NewJavaFile(path).Length(); got != int64(len("only\n")) {
		t.Fatalf("length after truncating write = %d", got)
	}
}

func TestBufferedWriterWritesLinesThroughToTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buffered.txt")

	w := NewBufferedWriter(path)
	w.WriteString("alpha")
	w.NewLine()
	w.WriteString("beta")
	w.NewLine()
	w.Close()

	if got := FilesReadString(path); got != "alpha\nbeta\n" {
		t.Fatalf("BufferedWriter produced %q", got)
	}
}

func TestNestedWritersFlushAndCloseThroughToTheDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested.txt")

	// new PrintWriter(new BufferedWriter(new FileWriter(path))): closing the outer
	// writer must close the whole chain, as in Java.
	inner := NewBufferedWriter(path)
	outer := NewPrintWriter(inner)
	outer.Println("through")
	outer.Close()

	if got := FilesReadString(path); got != "through\n" {
		t.Fatalf("nested writers produced %q", got)
	}
}

func TestStringWriterCollectsWhatAPrintWriterWrites(t *testing.T) {
	memory := NewStringWriter()
	w := NewPrintWriter(memory)
	w.Print("in-")
	w.Println("memory")
	w.Flush()

	if got := memory.String(); got != "in-memory\n" {
		t.Fatalf("StringWriter = %q, want in-memory\\n", got)
	}
}

func TestByteArrayOutputStreamCapturesAPrintStream(t *testing.T) {
	captured := NewByteArrayOutputStream()
	s := NewPrintStream(captured)
	s.Print("captured")
	s.Flush()

	if got := captured.String(); got != "captured" {
		t.Fatalf("ByteArrayOutputStream = %q", got)
	}
	if got := captured.Size(); got != 8 {
		t.Fatalf("Size = %d, want 8", got)
	}
	if got := captured.ToByteArray().Elements[0]; got != int8('c') {
		t.Fatalf("ToByteArray[0] = %d, want %d", got, int8('c'))
	}

	captured.Reset()
	if got := captured.Size(); got != 0 {
		t.Fatalf("Size after Reset = %d", got)
	}
}

func TestByteStreamRoundTripThroughAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bytes.bin")

	out := NewFileOutputStream(path)
	out.WriteBytes(int32('h'))
	out.WriteBytes(PrimitiveArrayLiteral(PrimitiveByteTypeID, int8('e'), int8('y')))
	out.Close()

	in := NewFileInputStream(path)
	defer in.Close()
	if got := in.ReadByteValue(); got != int32('h') {
		t.Fatalf("first byte = %d, want %d", got, int32('h'))
	}
	rest := in.ReadAllBytes().Elements
	if len(rest) != 2 || rest[0] != int8('e') || rest[1] != int8('y') {
		t.Fatalf("ReadAllBytes = %v", rest)
	}
	// Java's read() reports -1 rather than throwing at end of stream.
	if got := in.ReadByteValue(); got != -1 {
		t.Fatalf("read at EOF = %d, want -1", got)
	}
}

func TestByteArrayInputStreamReadsAndReportsRemaining(t *testing.T) {
	in := NewByteArrayInputStream("abc")
	if got := in.Available(); got != 3 {
		t.Fatalf("Available = %d, want 3", got)
	}
	if got := in.ReadByteValue(); got != int32('a') {
		t.Fatalf("ReadByteValue = %d", got)
	}
	if got := in.Available(); got != 2 {
		t.Fatalf("Available after one read = %d, want 2", got)
	}
	in.Close()
}

func TestBufferedReaderWrapsInMemoryAndStreamSources(t *testing.T) {
	// new BufferedReader(new StringReader(...)) and the InputStreamReader form
	// both compose through the shared source plumbing.
	fromString := NewBufferedReader(NewStringReader("one\ntwo\n"))
	if got := fromString.Lines().Count(); got != 2 {
		t.Fatalf("Lines over a StringReader counted %d, want 2", got)
	}
	fromString.Close()

	fromBytes := NewBufferedReader(NewInputStreamReader(NewByteArrayInputStream("alpha\nbeta")))
	if got := fromBytes.ReadLine(); got != "alpha" {
		t.Fatalf("line over an InputStreamReader = %q, want alpha", got)
	}
	fromBytes.Close()
}

func TestBufferedReaderLinesDrainsTheRemainingLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lines.txt")
	FilesWriteString(path, "a\nb\nc\n")

	r := NewBufferedReader(path)
	defer r.Close()
	if got := r.ReadLine(); got != "a" {
		t.Fatalf("first line = %q", got)
	}
	// Lines() starts where the reader is, matching BufferedReader.lines().
	if got := r.Lines().ToSlice(); len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("Lines = %q", got)
	}
}

func TestFileMetadataAndCreationHelpers(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "made", "deeper")
	d := NewJavaFile(dir)
	if !d.Mkdirs() {
		t.Fatalf("Mkdirs = false for a new directory")
	}
	// Java's mkdirs reports false when the directory is already there.
	if d.Mkdirs() {
		t.Fatalf("Mkdirs = true for an existing directory")
	}
	if !d.IsDirectory() || d.IsFile() {
		t.Fatalf("IsDirectory/IsFile disagree for a directory")
	}

	f := NewJavaFile(filepath.Join(dir, "new.txt"))
	if !f.CreateNewFile() {
		t.Fatalf("CreateNewFile = false for a new file")
	}
	if f.CreateNewFile() {
		t.Fatalf("CreateNewFile = true for an existing file")
	}
	if !f.IsFile() {
		t.Fatalf("IsFile = false for a regular file")
	}
	if !filepath.IsAbs(f.GetAbsolutePath()) {
		t.Fatalf("GetAbsolutePath = %q, want an absolute path", f.GetAbsolutePath())
	}
	if got := f.ToPath().GetFileName().ToString(); got != "new.txt" {
		t.Fatalf("ToPath().GetFileName() = %q, want new.txt", got)
	}
}

func TestOpeningAMissingFileThrowsIOException(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	defer func() {
		thrown := recover()
		if thrown == nil {
			t.Fatalf("opening a missing file did not throw")
		}
		if _, ok := thrown.(IOException); !ok {
			t.Fatalf("threw %T, want IOException", thrown)
		}
	}()
	NewBufferedReader(missing)
}
