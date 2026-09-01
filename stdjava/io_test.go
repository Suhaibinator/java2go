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
