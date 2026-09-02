package stdjava

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// This file implements the java.io / java.util.Scanner basics that transpiled
// programs use, backed by os and bufio.
//
// Documented approximations:
//   - Java's checked IOExceptions are not threaded as Go errors; an I/O failure
//     panics with a stdjava IOException, preserving Java's "operation throws"
//     control flow rather than silently producing a zero value. The message text
//     is Go's, not the JDK's.
//   - Character encoding is the platform default (UTF-8) only; Java's
//     charset-selecting constructors are not modeled.
//   - Buffering/flush timing is approximated; PrintWriter flushes on Close.
//   - Java's byte is signed, so the byte-oriented streams here exchange the
//     *PrimitiveArray[int8] that the transpiler uses for byte[], and also accept
//     a Go []byte or a string for convenience.
//
// Deliberately out of scope, because nothing short of a much larger runtime
// models them usefully: RandomAccessFile, object serialization
// (ObjectInput/OutputStream, Serializable), the java.nio channel/buffer/selector
// stack, WatchService, charset-selecting constructors, and file
// permissions/attributes.

// throwIOException converts a Go I/O error into the panic that transpiled code
// observes as a thrown java.io.IOException.
func throwIOException(err error) {
	panic(NewIOException(err.Error()))
}

// --- shared source/destination plumbing -------------------------------------

// ioPathOf extracts a filesystem path from an io destination/source argument
// that is either a path string, a *JavaFile, or a *JavaPath. Every entry point
// that Java overloads across String/File/Path funnels through here.
func ioPathOf(arg any) string {
	switch v := arg.(type) {
	case string:
		return v
	case *JavaFile:
		return v.path
	case *JavaPath:
		return v.path
	default:
		panic("stdjava: unsupported file argument type")
	}
}

// ioSink is the write end shared by the PrintWriter/PrintStream/BufferedWriter/
// OutputStreamWriter shims. Java composes these by wrapping one writer in
// another, so a sink carries the flush and close behavior of whatever it wraps:
// closing an outer writer closes the stream underneath it, as in Java.
type ioSink struct {
	w     io.Writer
	flush func()
	close func()
}

// newFileSink opens (creating, then truncating or appending) a file as a
// buffered sink. It panics with IOException on failure.
func newFileSink(path string, appendMode bool) ioSink {
	flags := os.O_WRONLY | os.O_CREATE
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0o666)
	if err != nil {
		throwIOException(err)
	}
	buf := bufio.NewWriter(file)
	return ioSink{
		w:     buf,
		flush: func() { _ = buf.Flush() },
		close: func() { _ = buf.Flush(); _ = file.Close() },
	}
}

// nestedSink wraps an already-constructed stdjava writer, so the outer writer
// forwards flush/close to it.
func nestedSink(w io.Writer, flush, close func()) ioSink {
	return ioSink{w: w, flush: flush, close: close}
}

// ioSinkOf resolves a Java writer/stream destination argument: a path string, a
// File, a Path, or another stdjava writer to wrap. appendMode applies only when
// the destination names a file.
func ioSinkOf(dest any, appendMode bool) ioSink {
	switch v := dest.(type) {
	case string:
		return newFileSink(v, appendMode)
	case *JavaFile:
		return newFileSink(v.path, appendMode)
	case *JavaPath:
		return newFileSink(v.path, appendMode)
	case *FileOutputStream:
		return nestedSink(v, v.Flush, v.Close)
	case *ByteArrayOutputStream:
		return nestedSink(v, v.Flush, v.Close)
	case *StringWriter:
		return nestedSink(v, v.Flush, v.Close)
	case *OutputStreamWriter:
		return nestedSink(v, v.Flush, v.Close)
	case *BufferedWriter:
		return nestedSink(v, v.Flush, v.Close)
	case *PrintWriter:
		return nestedSink(v, v.Flush, v.Close)
	case *PrintStream:
		return nestedSink(v, v.Flush, v.Close)
	case io.Writer:
		// A raw Go writer (e.g. os.Stdout) is not ours to close.
		return nestedSink(v, func() {}, func() {})
	default:
		panic("stdjava: unsupported writer destination type")
	}
}

// ioSource is the read end shared by the reader/stream shims, mirroring ioSink.
type ioSource struct {
	r     io.Reader
	close func()
}

// newFileSource opens a file for reading. It panics with IOException on failure.
func newFileSource(path string) ioSource {
	file, err := os.Open(path)
	if err != nil {
		throwIOException(err)
	}
	return ioSource{r: file, close: func() { _ = file.Close() }}
}

// ioSourceOf resolves a Java reader/stream source argument: a path string, a
// File, a Path, or another stdjava reader to wrap.
func ioSourceOf(src any) ioSource {
	switch v := src.(type) {
	case string:
		return newFileSource(v)
	case *JavaFile:
		return newFileSource(v.path)
	case *JavaPath:
		return newFileSource(v.path)
	case *FileInputStream:
		return ioSource{r: v, close: v.Close}
	case *ByteArrayInputStream:
		return ioSource{r: v, close: v.Close}
	case *StringReader:
		return ioSource{r: v, close: v.Close}
	case *InputStreamReader:
		return ioSource{r: v, close: v.Close}
	case *BufferedReader:
		return ioSource{r: v, close: v.Close}
	case io.Reader:
		return ioSource{r: v, close: func() {}}
	default:
		panic("stdjava: unsupported reader source type")
	}
}

// javaBytes normalizes the byte-payload forms transpiled code can produce into a
// Go []byte. A Java byte[] arrives as a *PrimitiveArray[int8], since Java's byte
// is signed.
func javaBytes(value any) []byte {
	switch v := value.(type) {
	case []byte:
		return v
	case *PrimitiveArray[int8]:
		return unsignedBytes(v.Elements)
	case []int8:
		return unsignedBytes(v)
	case string:
		return []byte(v)
	default:
		return []byte(javaString(value))
	}
}

// unsignedBytes reinterprets Java's signed bytes as Go bytes.
func unsignedBytes(data []int8) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = byte(b)
	}
	return out
}

// signedByteArray reinterprets Go bytes as a Java byte[].
func signedByteArray(data []byte) *PrimitiveArray[int8] {
	elements := make([]int8, len(data))
	for i, b := range data {
		elements[i] = int8(b)
	}
	return PrimitiveArrayLiteral(PrimitiveByteTypeID, elements...)
}

// --- java.io.File -----------------------------------------------------------

// JavaFile models java.io.File: a path, not an open handle.
type JavaFile struct {
	path string
}

// NewJavaFile returns a File for the given pathname, matching `new File(path)`.
func NewJavaFile(path string) *JavaFile {
	return &JavaFile{path: path}
}

// CreateTempFile creates a new empty file in the default temp directory with the
// given prefix and suffix, matching File.createTempFile, and returns a File for
// it. It panics on failure.
func CreateTempFile(prefix, suffix string) *JavaFile {
	f, err := os.CreateTemp("", prefix+"*"+suffix)
	if err != nil {
		throwIOException(err)
	}
	name := f.Name()
	_ = f.Close()
	return &JavaFile{path: name}
}

// Exists reports whether the file or directory exists, matching File.exists.
func (f *JavaFile) Exists() bool {
	_, err := os.Stat(f.path)
	return err == nil
}

// GetName returns the final path component, matching File.getName.
func (f *JavaFile) GetName() string {
	return filepath.Base(f.path)
}

// GetPath returns the pathname string, matching File.getPath.
func (f *JavaFile) GetPath() string {
	return f.path
}

// GetAbsolutePath returns the path resolved against the working directory,
// matching File.getAbsolutePath.
func (f *JavaFile) GetAbsolutePath() string {
	abs, err := filepath.Abs(f.path)
	if err != nil {
		return f.path
	}
	return abs
}

// IsDirectory reports whether the path is a directory, matching File.isDirectory.
func (f *JavaFile) IsDirectory() bool {
	info, err := os.Stat(f.path)
	return err == nil && info.IsDir()
}

// IsFile reports whether the path is a regular file, matching File.isFile.
func (f *JavaFile) IsFile() bool {
	info, err := os.Stat(f.path)
	return err == nil && info.Mode().IsRegular()
}

// Length returns the file size in bytes, matching File.length (0 if absent).
func (f *JavaFile) Length() int64 {
	info, err := os.Stat(f.path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// Delete removes the file and reports success, matching File.delete.
func (f *JavaFile) Delete() bool {
	return os.Remove(f.path) == nil
}

// Mkdir creates the directory but not its parents and reports whether it did,
// matching File.mkdir.
func (f *JavaFile) Mkdir() bool {
	return os.Mkdir(f.path, 0o777) == nil
}

// Mkdirs creates the directory and any missing parents and reports whether it
// created anything, matching File.mkdirs (false when the directory already
// exists).
func (f *JavaFile) Mkdirs() bool {
	if f.Exists() {
		return false
	}
	return os.MkdirAll(f.path, 0o777) == nil
}

// CreateNewFile creates the file only if it does not exist and reports whether
// it did so, matching File.createNewFile.
func (f *JavaFile) CreateNewFile() bool {
	file, err := os.OpenFile(f.path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		if os.IsExist(err) {
			return false
		}
		throwIOException(err)
	}
	_ = file.Close()
	return true
}

// ToPath returns this file as a Path, matching File.toPath.
func (f *JavaFile) ToPath() *JavaPath {
	return NewJavaPath(f.path)
}

// --- character writers ------------------------------------------------------

// PrintWriter models java.io.PrintWriter / FileWriter over a file or another
// writer. Writes are buffered and flushed on Close.
type PrintWriter struct {
	sink ioSink
}

// NewPrintWriter opens (creating/truncating) the destination for writing,
// matching `new PrintWriter(...)` / `new FileWriter(...)`. The destination may be
// a path string, a *JavaFile, a *JavaPath, or another stdjava writer to wrap. It
// panics on failure.
func NewPrintWriter(dest any) *PrintWriter {
	return &PrintWriter{sink: ioSinkOf(dest, false)}
}

// NewPrintWriterAppend matches the two-argument `new FileWriter(dest, append)`:
// when append is true existing content is kept and writes go to the end.
func NewPrintWriterAppend(dest any, appendMode bool) *PrintWriter {
	return &PrintWriter{sink: ioSinkOf(dest, appendMode)}
}

// Print writes the textual form of value, matching PrintWriter.print.
func (w *PrintWriter) Print(value any) {
	_, _ = io.WriteString(w.sink.w, javaString(value))
}

// Println writes the textual form of value followed by a newline, matching
// PrintWriter.println. With no argument, writes just a newline.
func (w *PrintWriter) Println(value any) {
	_, _ = io.WriteString(w.sink.w, javaString(value)+"\n")
}

// PrintlnEmpty writes a bare newline, matching the no-argument PrintWriter.println().
func (w *PrintWriter) PrintlnEmpty() {
	_, _ = io.WriteString(w.sink.w, "\n")
}

// Write makes a PrintWriter usable as the destination of another writer,
// mirroring how Java nests Writers. Java's own write(String) maps to Print.
func (w *PrintWriter) Write(p []byte) (int, error) {
	return w.sink.w.Write(p)
}

// Flush writes any buffered data through to the underlying destination, matching
// PrintWriter.flush.
func (w *PrintWriter) Flush() {
	w.sink.flush()
}

// Close flushes and closes the writer and anything it wraps, matching
// PrintWriter.close.
func (w *PrintWriter) Close() {
	w.sink.close()
}

// BufferedWriter models java.io.BufferedWriter. It adds Java's newLine() to the
// wrapped writer; the buffering itself is already provided by the sink.
type BufferedWriter struct {
	sink ioSink
}

// NewBufferedWriter wraps a destination for buffered writing, matching
// `new BufferedWriter(new FileWriter(...))`. The destination may be a path
// string, a *JavaFile, a *JavaPath, or another stdjava writer.
func NewBufferedWriter(dest any) *BufferedWriter {
	return &BufferedWriter{sink: ioSinkOf(dest, false)}
}

// WriteString writes the textual form of value, matching BufferedWriter.write.
func (w *BufferedWriter) WriteString(value any) {
	_, _ = io.WriteString(w.sink.w, javaString(value))
}

// NewLine writes the line separator, matching BufferedWriter.newLine. The
// separator is always "\n" here, not the platform's line.separator property.
func (w *BufferedWriter) NewLine() {
	_, _ = io.WriteString(w.sink.w, "\n")
}

// Write makes a BufferedWriter usable as another writer's destination.
func (w *BufferedWriter) Write(p []byte) (int, error) {
	return w.sink.w.Write(p)
}

// Flush pushes buffered data through, matching BufferedWriter.flush.
func (w *BufferedWriter) Flush() {
	w.sink.flush()
}

// Close flushes and closes this writer and anything it wraps, matching
// BufferedWriter.close.
func (w *BufferedWriter) Close() {
	w.sink.close()
}

// OutputStreamWriter models java.io.OutputStreamWriter: the character view of a
// byte stream. Only the platform UTF-8 encoding is modeled.
type OutputStreamWriter struct {
	sink ioSink
}

// NewOutputStreamWriter wraps a byte destination for character writing, matching
// `new OutputStreamWriter(out)`.
func NewOutputStreamWriter(dest any) *OutputStreamWriter {
	return &OutputStreamWriter{sink: ioSinkOf(dest, false)}
}

// NewOutputStreamWriterStdout wraps standard output, matching
// `new OutputStreamWriter(System.out)`.
func NewOutputStreamWriterStdout() *OutputStreamWriter {
	return &OutputStreamWriter{sink: nestedSink(os.Stdout, func() {}, func() {})}
}

// WriteString writes the textual form of value, matching
// OutputStreamWriter.write.
func (w *OutputStreamWriter) WriteString(value any) {
	_, _ = io.WriteString(w.sink.w, javaString(value))
}

// Write makes an OutputStreamWriter usable as another writer's destination.
func (w *OutputStreamWriter) Write(p []byte) (int, error) {
	return w.sink.w.Write(p)
}

// Flush pushes buffered data through, matching OutputStreamWriter.flush.
func (w *OutputStreamWriter) Flush() {
	w.sink.flush()
}

// Close flushes and closes this writer and the stream underneath, matching
// OutputStreamWriter.close.
func (w *OutputStreamWriter) Close() {
	w.sink.close()
}

// StringWriter models java.io.StringWriter: a writer that accumulates into an
// in-memory string.
type StringWriter struct {
	builder strings.Builder
}

// NewStringWriter returns an empty StringWriter, matching `new StringWriter()`.
func NewStringWriter() *StringWriter {
	return &StringWriter{}
}

// WriteString appends the textual form of value, matching StringWriter.write.
func (w *StringWriter) WriteString(value any) {
	w.builder.WriteString(javaString(value))
}

// Write makes a StringWriter usable as another writer's destination.
func (w *StringWriter) Write(p []byte) (int, error) {
	return w.builder.Write(p)
}

// String returns everything written so far, matching StringWriter.toString.
func (w *StringWriter) String() string {
	return w.builder.String()
}

// Flush is a no-op kept for API parity with StringWriter.flush.
func (w *StringWriter) Flush() {}

// Close is a no-op kept for API parity with StringWriter.close, which Java also
// documents as having no effect.
func (w *StringWriter) Close() {}

// PrintStream models java.io.PrintStream for destinations other than System.out,
// which the transpiler lowers straight to fmt. Writes are buffered and flushed
// on Close, so Java's autoflush-on-newline is not modeled.
type PrintStream struct {
	sink ioSink
}

// NewPrintStream opens the destination for writing, matching
// `new PrintStream(...)`. The destination may be a path string, a *JavaFile, a
// *JavaPath, or another stdjava stream to wrap.
func NewPrintStream(dest any) *PrintStream {
	return &PrintStream{sink: ioSinkOf(dest, false)}
}

// Print writes the textual form of value, matching PrintStream.print.
func (s *PrintStream) Print(value any) {
	_, _ = io.WriteString(s.sink.w, javaString(value))
}

// Println writes the textual form of value followed by a newline, matching
// PrintStream.println.
func (s *PrintStream) Println(value any) {
	_, _ = io.WriteString(s.sink.w, javaString(value)+"\n")
}

// PrintlnEmpty writes a bare newline, matching the no-argument
// PrintStream.println().
func (s *PrintStream) PrintlnEmpty() {
	_, _ = io.WriteString(s.sink.w, "\n")
}

// Write makes a PrintStream usable as another writer's destination.
func (s *PrintStream) Write(p []byte) (int, error) {
	return s.sink.w.Write(p)
}

// Flush pushes buffered data through, matching PrintStream.flush.
func (s *PrintStream) Flush() {
	s.sink.flush()
}

// Close flushes and closes the stream, matching PrintStream.close.
func (s *PrintStream) Close() {
	s.sink.close()
}

// --- byte streams -----------------------------------------------------------

// FileOutputStream models java.io.FileOutputStream. Writes are buffered and
// flushed on Close.
type FileOutputStream struct {
	file *os.File
	buf  *bufio.Writer
}

// NewFileOutputStream opens (creating/truncating) the destination for writing,
// matching `new FileOutputStream(dest)`. The destination may be a path string, a
// *JavaFile, or a *JavaPath.
func NewFileOutputStream(dest any) *FileOutputStream {
	return NewFileOutputStreamAppend(dest, false)
}

// NewFileOutputStreamAppend matches the two-argument
// `new FileOutputStream(dest, append)`.
func NewFileOutputStreamAppend(dest any, appendMode bool) *FileOutputStream {
	flags := os.O_WRONLY | os.O_CREATE
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(ioPathOf(dest), flags, 0o666)
	if err != nil {
		throwIOException(err)
	}
	return &FileOutputStream{file: file, buf: bufio.NewWriter(file)}
}

// WriteBytes writes a single byte value or a whole byte array, matching both
// FileOutputStream.write(int) and write(byte[]).
func (s *FileOutputStream) WriteBytes(value any) {
	switch v := value.(type) {
	case int32:
		_ = s.buf.WriteByte(byte(v))
	case int:
		_ = s.buf.WriteByte(byte(v))
	default:
		_, _ = s.buf.Write(javaBytes(value))
	}
}

// Write makes a FileOutputStream usable as another writer's destination.
func (s *FileOutputStream) Write(p []byte) (int, error) {
	return s.buf.Write(p)
}

// Flush pushes buffered data to the file, matching FileOutputStream.flush.
func (s *FileOutputStream) Flush() {
	_ = s.buf.Flush()
}

// Close flushes and closes the file, matching FileOutputStream.close.
func (s *FileOutputStream) Close() {
	_ = s.buf.Flush()
	_ = s.file.Close()
}

// FileInputStream models java.io.FileInputStream.
type FileInputStream struct {
	file *os.File
	buf  *bufio.Reader
}

// NewFileInputStream opens the source for reading, matching
// `new FileInputStream(src)`. The source may be a path string, a *JavaFile, or a
// *JavaPath. It panics on failure.
func NewFileInputStream(src any) *FileInputStream {
	file, err := os.Open(ioPathOf(src))
	if err != nil {
		throwIOException(err)
	}
	return &FileInputStream{file: file, buf: bufio.NewReader(file)}
}

// ReadByteValue returns the next byte as an unsigned value in 0..255, or -1 at
// end of stream, matching InputStream.read(). It is not named ReadByte because
// that name is reserved in Go for the (byte, error) io.ByteReader signature.
func (s *FileInputStream) ReadByteValue() int32 {
	b, err := s.buf.ReadByte()
	if err != nil {
		if err == io.EOF {
			return -1
		}
		throwIOException(err)
	}
	return int32(b)
}

// ReadAllBytes reads the remainder of the stream, matching
// InputStream.readAllBytes.
func (s *FileInputStream) ReadAllBytes() *PrimitiveArray[int8] {
	data, err := io.ReadAll(s.buf)
	if err != nil {
		throwIOException(err)
	}
	return signedByteArray(data)
}

// Available reports the bytes buffered for a non-blocking read, matching
// InputStream.available. Java's value is an estimate too, but for a file it is
// the whole remainder rather than just the buffered part.
func (s *FileInputStream) Available() int32 {
	return int32(s.buf.Buffered())
}

// Read makes a FileInputStream usable as another reader's source.
func (s *FileInputStream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

// Close closes the file, matching FileInputStream.close.
func (s *FileInputStream) Close() {
	_ = s.file.Close()
}

// ByteArrayOutputStream models java.io.ByteArrayOutputStream: an in-memory byte
// sink, most often used to capture what another stream writes.
type ByteArrayOutputStream struct {
	buf bytes.Buffer
}

// NewByteArrayOutputStream returns an empty stream, matching
// `new ByteArrayOutputStream()`.
func NewByteArrayOutputStream() *ByteArrayOutputStream {
	return &ByteArrayOutputStream{}
}

// WriteBytes writes a single byte value or a whole byte array, matching
// ByteArrayOutputStream.write(int) and write(byte[]).
func (s *ByteArrayOutputStream) WriteBytes(value any) {
	switch v := value.(type) {
	case int32:
		s.buf.WriteByte(byte(v))
	case int:
		s.buf.WriteByte(byte(v))
	default:
		s.buf.Write(javaBytes(value))
	}
}

// Write makes a ByteArrayOutputStream usable as another writer's destination.
func (s *ByteArrayOutputStream) Write(p []byte) (int, error) {
	return s.buf.Write(p)
}

// ToByteArray returns a copy of the accumulated bytes, matching
// ByteArrayOutputStream.toByteArray.
func (s *ByteArrayOutputStream) ToByteArray() *PrimitiveArray[int8] {
	return signedByteArray(s.buf.Bytes())
}

// String decodes the accumulated bytes as UTF-8, matching
// ByteArrayOutputStream.toString.
func (s *ByteArrayOutputStream) String() string {
	return s.buf.String()
}

// Size returns the number of bytes written, matching
// ByteArrayOutputStream.size.
func (s *ByteArrayOutputStream) Size() int32 {
	return int32(s.buf.Len())
}

// Reset discards the accumulated bytes, matching ByteArrayOutputStream.reset.
func (s *ByteArrayOutputStream) Reset() {
	s.buf.Reset()
}

// Flush is a no-op kept for API parity with ByteArrayOutputStream.flush.
func (s *ByteArrayOutputStream) Flush() {}

// Close is a no-op kept for API parity with ByteArrayOutputStream.close, which
// Java also documents as having no effect.
func (s *ByteArrayOutputStream) Close() {}

// ByteArrayInputStream models java.io.ByteArrayInputStream over an in-memory
// byte array.
type ByteArrayInputStream struct {
	reader *bytes.Reader
}

// NewByteArrayInputStream reads from the given bytes, matching
// `new ByteArrayInputStream(data)`. data may be Java's byte[] ([]int8), a Go
// []byte, or a string.
func NewByteArrayInputStream(data any) *ByteArrayInputStream {
	return &ByteArrayInputStream{reader: bytes.NewReader(javaBytes(data))}
}

// ReadByteValue returns the next byte as an unsigned value in 0..255, or -1 at
// end of stream, matching InputStream.read(). It is not named ReadByte because
// that name is reserved in Go for the (byte, error) io.ByteReader signature.
func (s *ByteArrayInputStream) ReadByteValue() int32 {
	b, err := s.reader.ReadByte()
	if err != nil {
		return -1
	}
	return int32(b)
}

// ReadAllBytes reads the remainder of the stream, matching
// InputStream.readAllBytes.
func (s *ByteArrayInputStream) ReadAllBytes() *PrimitiveArray[int8] {
	data, err := io.ReadAll(s.reader)
	if err != nil {
		throwIOException(err)
	}
	return signedByteArray(data)
}

// Available reports the bytes remaining, matching
// ByteArrayInputStream.available.
func (s *ByteArrayInputStream) Available() int32 {
	return int32(s.reader.Len())
}

// Read makes a ByteArrayInputStream usable as another reader's source.
func (s *ByteArrayInputStream) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

// Close is a no-op kept for API parity with ByteArrayInputStream.close.
func (s *ByteArrayInputStream) Close() {}

// --- character readers ------------------------------------------------------

// StringReader models java.io.StringReader: a reader over an in-memory string.
type StringReader struct {
	reader *strings.Reader
}

// NewStringReader reads from the given string, matching
// `new StringReader(s)`.
func NewStringReader(s string) *StringReader {
	return &StringReader{reader: strings.NewReader(s)}
}

// ReadChar returns the next character as its code point, or -1 at end of stream,
// matching Reader.read(). Java returns a UTF-16 code unit; this returns a full
// rune, which differs above the basic multilingual plane.
func (r *StringReader) ReadChar() int32 {
	ch, _, err := r.reader.ReadRune()
	if err != nil {
		return -1
	}
	return int32(ch)
}

// Read makes a StringReader usable as another reader's source.
func (r *StringReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

// Close is a no-op kept for API parity with StringReader.close.
func (r *StringReader) Close() {}

// InputStreamReader models java.io.InputStreamReader: the character view of a
// byte stream. Only the platform UTF-8 encoding is modeled.
type InputStreamReader struct {
	src ioSource
}

// NewInputStreamReader wraps a byte source for character reading, matching
// `new InputStreamReader(in)`.
func NewInputStreamReader(src any) *InputStreamReader {
	return &InputStreamReader{src: ioSourceOf(src)}
}

// NewInputStreamReaderStdin wraps standard input, matching
// `new InputStreamReader(System.in)`.
func NewInputStreamReaderStdin() *InputStreamReader {
	return &InputStreamReader{src: ioSource{r: os.Stdin, close: func() {}}}
}

// Read makes an InputStreamReader usable as another reader's source; this is how
// `new BufferedReader(new InputStreamReader(System.in))` composes.
func (r *InputStreamReader) Read(p []byte) (int, error) {
	return r.src.r.Read(p)
}

// Close closes the stream underneath, matching InputStreamReader.close.
func (r *InputStreamReader) Close() {
	r.src.close()
}

// BufferedReader models java.io.BufferedReader over a file, a FileReader, or an
// InputStreamReader.
type BufferedReader struct {
	src ioSource
	buf *bufio.Reader
}

// NewBufferedReader opens the source for reading, matching
// `new BufferedReader(new FileReader(...))`. The source may be a path string, a
// *JavaFile, a *JavaPath, or another stdjava reader (so
// `new BufferedReader(new InputStreamReader(System.in))` composes). It panics on
// failure.
func NewBufferedReader(src any) *BufferedReader {
	source := ioSourceOf(src)
	return &BufferedReader{src: source, buf: bufio.NewReader(source.r)}
}

// ReadLineOK reads the next line without its terminator and separately reports
// whether a line was present. The boolean preserves BufferedReader.readLine's
// null-at-EOF distinction without making every transpiled Java String a pointer.
func (r *BufferedReader) ReadLineOK() (string, bool) {
	line := make([]byte, 0, 80)
	for {
		b, err := r.buf.ReadByte()
		if err != nil {
			if err == io.EOF {
				if len(line) == 0 {
					return "", false
				}
				return string(line), true
			}
			throwIOException(err)
		}

		switch b {
		case '\n':
			return string(line), true
		case '\r':
			// Java accepts CR, LF, and CRLF. Consume the LF half of CRLF but
			// leave any other following byte for the next call.
			if next, err := r.buf.Peek(1); err == nil && next[0] == '\n' {
				_, _ = r.buf.ReadByte()
			}
			return string(line), true
		default:
			line = append(line, b)
		}
	}
}

// ReadLine reads the next line without its terminator. Direct calls use the Go
// string zero value at EOF; canonical Java `while ((line = r.readLine()) !=
// null)` loops are lowered to ReadLineInto so empty lines remain distinguishable
// from EOF.
func (r *BufferedReader) ReadLine() string {
	line, _ := r.ReadLineOK()
	return line
}

// ReadLineInto implements the value-producing readLine/null-check loop idiom.
// It updates target only when a line is available and returns the Java
// `readLine() != null` condition.
func (r *BufferedReader) ReadLineInto(target *string) bool {
	line, ok := r.ReadLineOK()
	if ok {
		*target = line
	}
	return ok
}

// Lines returns the remaining lines as a Stream, matching BufferedReader.lines.
// Java's stream is lazy over the open reader; this one drains the reader eagerly,
// so the reader is exhausted as soon as Lines returns.
func (r *BufferedReader) Lines() Stream[string] {
	var lines []string
	for {
		line, ok := r.ReadLineOK()
		if !ok {
			return StreamOfSlice(lines)
		}
		lines = append(lines, line)
	}
}

// Ready reports whether more data is available to read without blocking,
// matching BufferedReader.ready — usable as the loop guard in place of a
// readLine() != null check.
func (r *BufferedReader) Ready() bool {
	_, err := r.buf.Peek(1)
	return err == nil
}

// Read makes a BufferedReader usable as another reader's source.
func (r *BufferedReader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}

// Close closes the reader and the source underneath, matching
// BufferedReader.close.
func (r *BufferedReader) Close() {
	r.src.close()
}

// --- java.util.Scanner ------------------------------------------------------

// Scanner models java.util.Scanner over an io.Reader (e.g. System.in or a file).
// Tokens are whitespace-separated, matching Scanner's default delimiter.
type Scanner struct {
	wordReader *bufio.Scanner
}

// NewScannerStdin returns a Scanner reading from standard input, matching
// `new Scanner(System.in)`.
func NewScannerStdin() *Scanner {
	return newScanner(os.Stdin)
}

// NewScannerFile returns a Scanner reading from a file, matching
// `new Scanner(new File(path))`. The source may be a path string, a *JavaFile, or
// a *JavaPath. It panics on failure.
func NewScannerFile(src any) *Scanner {
	file, err := os.Open(ioPathOf(src))
	if err != nil {
		throwIOException(err)
	}
	return newScanner(file)
}

func newScanner(r io.Reader) *Scanner {
	// Two views over the same stream are not generally safe, so a Scanner uses
	// the word scanner for token reads (nextInt/next/nextDouble) OR the line
	// reader for nextLine; mixing the two is a known approximation. The word
	// scanner is the default since token reads dominate.
	ws := bufio.NewScanner(r)
	ws.Split(bufio.ScanWords)
	return &Scanner{wordReader: ws}
}

// NextInt reads the next whitespace-delimited token as an int32, matching
// Scanner.nextInt. It panics if there is no token or it is not an integer.
func (s *Scanner) NextInt() int32 {
	if !s.wordReader.Scan() {
		panic("Scanner.nextInt: no more tokens")
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s.wordReader.Text()), 10, 32)
	if err != nil {
		panic(err)
	}
	return int32(v)
}

// NextLong reads the next token as an int64, matching Scanner.nextLong.
func (s *Scanner) NextLong() int64 {
	if !s.wordReader.Scan() {
		panic("Scanner.nextLong: no more tokens")
	}
	v, err := strconv.ParseInt(strings.TrimSpace(s.wordReader.Text()), 10, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// NextDouble reads the next token as a float64, matching Scanner.nextDouble.
func (s *Scanner) NextDouble() float64 {
	if !s.wordReader.Scan() {
		panic("Scanner.nextDouble: no more tokens")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s.wordReader.Text()), 64)
	if err != nil {
		panic(err)
	}
	return v
}

// Next reads the next whitespace-delimited token as a string, matching
// Scanner.next.
func (s *Scanner) Next() string {
	if !s.wordReader.Scan() {
		panic("Scanner.next: no more tokens")
	}
	return s.wordReader.Text()
}

// NextLine reads the rest of the current line, matching Scanner.nextLine. With
// the word-split scanner this returns the next whitespace-delimited token; full
// line semantics (including leading whitespace) are a documented approximation,
// since mixing token and line reads over one bufio source cannot be done exactly.
func (s *Scanner) NextLine() string {
	if !s.wordReader.Scan() {
		panic("Scanner.nextLine: no more input")
	}
	return s.wordReader.Text()
}

// HasNext reports whether another token is available, matching Scanner.hasNext.
func (s *Scanner) HasNext() bool {
	// bufio.Scanner cannot peek; this is a best-effort approximation that advances
	// the scanner. Mixing HasNext with the Next* readers is a known divergence.
	return s.wordReader.Scan()
}

// Close is a no-op kept for API parity with Scanner.close.
func (s *Scanner) Close() {}

// javaString renders a value the way Java's print/println would for the common
// cases (strings, booleans, numbers, and fmt.Stringer collections).
func javaString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case rune:
		return string(v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
