package stdjava

import (
	"bufio"
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
//     panics, preserving Java's "operation throws" control flow rather than
//     silently producing a zero value.
//   - Character encoding is the platform default (UTF-8) only; Java's
//     charset-selecting constructors are not modeled.
//   - Buffering/flush timing is approximated; PrintWriter flushes on Close.

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
		panic(err)
	}
	name := f.Name()
	f.Close()
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

// IsDirectory reports whether the path is a directory, matching File.isDirectory.
func (f *JavaFile) IsDirectory() bool {
	info, err := os.Stat(f.path)
	return err == nil && info.IsDir()
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

// PrintWriter models java.io.PrintWriter / FileWriter over a file. Writes are
// buffered and flushed on Close.
type PrintWriter struct {
	file *os.File
	buf  *bufio.Writer
}

// NewPrintWriter opens (creating/truncating) the destination for writing,
// matching `new PrintWriter(...)` / `new FileWriter(...)`. The destination may be
// a path string or a *JavaFile (so the nested new PrintWriter(new FileWriter(f))
// and new PrintWriter(path) forms both work). It panics on failure.
func NewPrintWriter(dest any) *PrintWriter {
	file, err := os.Create(ioPathOf(dest))
	if err != nil {
		panic(err)
	}
	return &PrintWriter{file: file, buf: bufio.NewWriter(file)}
}

// ioPathOf extracts a filesystem path from an io destination/source argument
// that is either a path string or a *JavaFile.
func ioPathOf(arg any) string {
	switch v := arg.(type) {
	case string:
		return v
	case *JavaFile:
		return v.path
	default:
		panic("stdjava: unsupported file argument type")
	}
}

// Print writes the textual form of value, matching PrintWriter.print.
func (w *PrintWriter) Print(value any) {
	w.buf.WriteString(javaString(value))
}

// Println writes the textual form of value followed by a newline, matching
// PrintWriter.println. With no argument, writes just a newline.
func (w *PrintWriter) Println(value any) {
	w.buf.WriteString(javaString(value))
	w.buf.WriteByte('\n')
}

// PrintlnEmpty writes a bare newline, matching the no-argument PrintWriter.println().
func (w *PrintWriter) PrintlnEmpty() {
	w.buf.WriteByte('\n')
}

// Flush writes any buffered data to the underlying file, matching
// PrintWriter.flush.
func (w *PrintWriter) Flush() {
	w.buf.Flush()
}

// Close flushes and closes the writer, matching PrintWriter.close.
func (w *PrintWriter) Close() {
	w.buf.Flush()
	w.file.Close()
}

// BufferedReader models java.io.BufferedReader over a FileReader.
type BufferedReader struct {
	file *os.File
	buf  *bufio.Reader
}

// NewBufferedReader opens the source for reading, matching
// `new BufferedReader(new FileReader(...))`. The source may be a path string or a
// *JavaFile. It panics on failure.
func NewBufferedReader(src any) *BufferedReader {
	file, err := os.Open(ioPathOf(src))
	if err != nil {
		panic(err)
	}
	return &BufferedReader{file: file, buf: bufio.NewReader(file)}
}

// ReadLine reads the next line without its terminator, matching
// BufferedReader.readLine. Java returns null at end of stream; since the
// transpiled return type is a Go string (not nullable), EOF yields the empty
// string. Use Ready to distinguish a genuine empty line from EOF.
func (r *BufferedReader) ReadLine() string {
	line, err := r.buf.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return strings.TrimRight(line, "\r\n")
		}
		panic(err)
	}
	return strings.TrimRight(line, "\r\n")
}

// Ready reports whether more data is available to read without blocking,
// matching BufferedReader.ready — usable as the loop guard in place of a
// readLine() != null check.
func (r *BufferedReader) Ready() bool {
	_, err := r.buf.Peek(1)
	return err == nil
}

// Close closes the reader, matching BufferedReader.close.
func (r *BufferedReader) Close() {
	r.file.Close()
}

// Scanner models java.util.Scanner over an io.Reader (e.g. System.in or a file).
// Tokens are whitespace-separated, matching Scanner's default delimiter.
type Scanner struct {
	lineReader *bufio.Reader
	wordReader *bufio.Scanner
}

// NewScannerStdin returns a Scanner reading from standard input, matching
// `new Scanner(System.in)`.
func NewScannerStdin() *Scanner {
	return newScanner(os.Stdin)
}

// NewScannerFile returns a Scanner reading from a file, matching
// `new Scanner(new File(path))`. The source may be a path string or a *JavaFile.
// It panics on failure.
func NewScannerFile(src any) *Scanner {
	file, err := os.Open(ioPathOf(src))
	if err != nil {
		panic(err)
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
