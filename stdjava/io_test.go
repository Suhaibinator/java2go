package stdjava

import (
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
