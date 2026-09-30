package stdjava

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsGetJoinsAndCollapsesSeparators(t *testing.T) {
	for _, test := range []struct {
		name  string
		parts []string
		want  string
	}{
		{"components", []string{"alpha", "beta"}, "alpha/beta"},
		{"embedded separators", []string{"alpha/beta", "gamma"}, "alpha/beta/gamma"},
		{"redundant separators", []string{"alpha//beta/", "/gamma"}, "alpha/beta/gamma"},
		{"empty components dropped", []string{"alpha", "", "beta"}, "alpha/beta"},
		{"absolute preserved", []string{"/alpha", "beta"}, "/alpha/beta"},
		{"empty path", []string{""}, ""},
		{"dot segments kept", []string{"alpha/./beta/.."}, "alpha/./beta/.."},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := PathsGet(test.parts[0], test.parts[1:]...).ToString()
			if got != test.want {
				t.Fatalf("PathsGet(%q) = %q, want %q", test.parts, got, test.want)
			}
		})
	}
}

func TestPathNameElements(t *testing.T) {
	sample := PathsGet("alpha", "beta", "gamma.txt")
	if got := sample.GetNameCount(); got != 3 {
		t.Fatalf("GetNameCount = %d, want 3", got)
	}
	if got := sample.GetFileName().ToString(); got != "gamma.txt" {
		t.Fatalf("GetFileName = %q, want gamma.txt", got)
	}
	if got := sample.GetParent().ToString(); got != "alpha/beta" {
		t.Fatalf("GetParent = %q, want alpha/beta", got)
	}
	// Java gives the empty path a single (empty) name element and the root none.
	if got := PathsGet("").GetNameCount(); got != 1 {
		t.Fatalf("empty GetNameCount = %d, want 1", got)
	}
	if got := PathsGet("/").GetNameCount(); got != 0 {
		t.Fatalf("root GetNameCount = %d, want 0", got)
	}
	if parent := PathsGet("alpha").GetParent(); parent != nil {
		t.Fatalf("GetParent of a single element = %q, want null", parent.ToString())
	}
	if got := PathsGet("/alpha").GetParent().ToString(); got != "/" {
		t.Fatalf("GetParent of /alpha = %q, want /", got)
	}
}

func TestPathResolveNormalizeAndPrefixes(t *testing.T) {
	if got := PathsGet("alpha").Resolve("beta").ToString(); got != "alpha/beta" {
		t.Fatalf("Resolve = %q, want alpha/beta", got)
	}
	// An absolute argument replaces the receiver, and an empty one leaves it be.
	if got := PathsGet("alpha").Resolve("/beta").ToString(); got != "/beta" {
		t.Fatalf("Resolve absolute = %q, want /beta", got)
	}
	if got := PathsGet("alpha").Resolve("").ToString(); got != "alpha" {
		t.Fatalf("Resolve empty = %q, want alpha", got)
	}
	// Resolve accepts a Path as well as a string.
	if got := PathsGet("alpha").Resolve(PathsGet("beta")).ToString(); got != "alpha/beta" {
		t.Fatalf("Resolve(Path) = %q, want alpha/beta", got)
	}

	if got := PathsGet("alpha/./beta/../gamma").Normalize().ToString(); got != "alpha/gamma" {
		t.Fatalf("Normalize = %q, want alpha/gamma", got)
	}
	// Java's normalize drops "." and a fully cancelled "..", leaving the empty path.
	if got := PathsGet(".").Normalize().ToString(); got != "" {
		t.Fatalf("Normalize of . = %q, want the empty path", got)
	}
	if got := PathsGet("alpha/..").Normalize().ToString(); got != "" {
		t.Fatalf("Normalize of alpha/.. = %q, want the empty path", got)
	}
	if got := PathsGet("../alpha").Normalize().ToString(); got != "../alpha" {
		t.Fatalf("Normalize of ../alpha = %q, want ../alpha", got)
	}

	sample := PathsGet("alpha", "beta", "gamma.txt")
	if !sample.StartsWith("alpha/beta") {
		t.Fatalf("StartsWith(alpha/beta) = false")
	}
	// Java compares whole name elements, so a truncated element does not match.
	if sample.StartsWith("alph") {
		t.Fatalf("StartsWith(alph) = true")
	}
	if sample.StartsWith("/alpha") {
		t.Fatalf("StartsWith of an absolute prefix on a relative path = true")
	}
	if !sample.EndsWith("beta/gamma.txt") {
		t.Fatalf("EndsWith(beta/gamma.txt) = false")
	}
	if sample.EndsWith("amma.txt") {
		t.Fatalf("EndsWith(amma.txt) = true")
	}
	// The empty path is one empty name element, so it prefixes nothing but itself.
	if sample.StartsWith("") || sample.EndsWith("") {
		t.Fatalf("a non-empty path matched the empty path")
	}
	if !PathsGet("").StartsWith("") {
		t.Fatalf("the empty path does not start with itself")
	}
	if !PathsGet("alpha").ToAbsolutePath().StartsWith("/") {
		t.Fatalf("ToAbsolutePath is not rooted")
	}
}

func TestPathFileInterop(t *testing.T) {
	dir := t.TempDir()
	path := PathsGet(dir).Resolve("interop.txt")
	if got := path.ToFile().GetName(); got != "interop.txt" {
		t.Fatalf("ToFile().GetName = %q, want interop.txt", got)
	}
	if got := NewJavaFile(path.ToString()).ToPath().ToString(); got != path.ToString() {
		t.Fatalf("File.ToPath round trip = %q, want %q", got, path.ToString())
	}
}

func TestFilesTextRoundTrip(t *testing.T) {
	path := PathsGet(t.TempDir()).Resolve("notes.txt")

	FilesWriteString(path, "alpha\nbeta\n")
	if got := FilesReadString(path); got != "alpha\nbeta\n" {
		t.Fatalf("FilesReadString = %q", got)
	}
	if got := FilesSize(path); got != 11 {
		t.Fatalf("FilesSize = %d, want 11", got)
	}

	lines := FilesReadAllLines(path)
	if got := lines.Size(); got != 2 {
		t.Fatalf("FilesReadAllLines size = %d, want 2", got)
	}
	if got := lines.Get(0); got != "alpha" {
		t.Fatalf("line 0 = %q, want alpha", got)
	}
	if got := FilesLines(path).Count(); got != 2 {
		t.Fatalf("FilesLines count = %d, want 2", got)
	}

	// A path string is accepted everywhere a Path is.
	if got := FilesReadString(path.ToString()); got != "alpha\nbeta\n" {
		t.Fatalf("FilesReadString(string) = %q", got)
	}
}

func TestFilesWriteLinesAndBytes(t *testing.T) {
	dir := t.TempDir()

	fromList := PathsGet(dir).Resolve("list.txt")
	FilesWrite(fromList, NewListFrom("one", "two"))
	if got := FilesReadString(fromList); got != "one\ntwo\n" {
		t.Fatalf("FilesWrite(List) wrote %q", got)
	}

	// Java's byte[] reaches the runtime as a *PrimitiveArray[int8].
	fromBytes := PathsGet(dir).Resolve("bytes.txt")
	FilesWrite(fromBytes, PrimitiveArrayLiteral(PrimitiveByteTypeID, int8('h'), int8('i')))
	if got := FilesReadString(fromBytes); got != "hi" {
		t.Fatalf("FilesWrite(byte[]) wrote %q", got)
	}

	empty := PathsGet(dir).Resolve("empty.txt")
	FilesWrite(empty, NewList[string]())
	if got := FilesSize(empty); got != 0 {
		t.Fatalf("FilesWrite of no lines produced %d bytes", got)
	}
}

func TestFilesDirectoryAndExistenceQueries(t *testing.T) {
	root := PathsGet(t.TempDir())
	nested := root.Resolve("a").Resolve("b")

	if FilesExists(nested) {
		t.Fatalf("FilesExists = true before creation")
	}
	FilesCreateDirectories(nested)
	if !FilesExists(nested) || !FilesIsDirectory(nested) {
		t.Fatalf("FilesCreateDirectories did not create a directory")
	}
	// Java's createDirectories is idempotent for an existing directory.
	FilesCreateDirectories(nested)

	file := nested.Resolve("f.txt")
	FilesCreateFile(file)
	if !FilesIsRegularFile(file) {
		t.Fatalf("FilesIsRegularFile = false for a created file")
	}
	if FilesIsDirectory(file) {
		t.Fatalf("FilesIsDirectory = true for a regular file")
	}
	if got := FilesSize(file); got != 0 {
		t.Fatalf("created file size = %d, want 0", got)
	}
}

func TestFilesCopyMoveAndDelete(t *testing.T) {
	dir := PathsGet(t.TempDir())
	source := dir.Resolve("source.txt")
	FilesWriteString(source, "payload")

	copied := dir.Resolve("copied.txt")
	FilesCopy(source, copied)
	if got := FilesReadString(copied); got != "payload" {
		t.Fatalf("FilesCopy produced %q", got)
	}
	if !FilesExists(source) {
		t.Fatalf("FilesCopy removed the source")
	}

	moved := dir.Resolve("moved.txt")
	FilesMove(copied, moved)
	if FilesExists(copied) {
		t.Fatalf("FilesMove left the source behind")
	}
	if got := FilesReadString(moved); got != "payload" {
		t.Fatalf("FilesMove produced %q", got)
	}

	if !FilesDeleteIfExists(moved) {
		t.Fatalf("FilesDeleteIfExists = false for an existing file")
	}
	if FilesDeleteIfExists(moved) {
		t.Fatalf("FilesDeleteIfExists = true for a missing file")
	}
	FilesDelete(source)
	if FilesExists(source) {
		t.Fatalf("FilesDelete left the file behind")
	}
}

func TestFilesFailuresThrowIOException(t *testing.T) {
	missing := PathsGet(t.TempDir()).Resolve("missing.txt")

	for name, operation := range map[string]func(){
		"readString":   func() { FilesReadString(missing) },
		"size":         func() { FilesSize(missing) },
		"delete":       func() { FilesDelete(missing) },
		"readAllLines": func() { FilesReadAllLines(missing) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				thrown := recover()
				if thrown == nil {
					t.Fatalf("%s on a missing file did not throw", name)
				}
				if _, ok := thrown.(IOException); !ok {
					t.Fatalf("%s threw %T, want IOException", name, thrown)
				}
			}()
			operation()
		})
	}
}

func TestFilesCreateFileRejectsExistingFile(t *testing.T) {
	path := PathsGet(t.TempDir()).Resolve("once.txt")
	FilesCreateFile(path)
	defer func() {
		if thrown := recover(); thrown == nil {
			t.Fatalf("FilesCreateFile on an existing file did not throw")
		}
	}()
	FilesCreateFile(path)
}

func TestPathRendersAsItsPathnameThroughFmt(t *testing.T) {
	// Print shims route values through javaString, which honours fmt.Stringer.
	if got := javaString(PathsGet("alpha", "beta")); got != "alpha/beta" {
		t.Fatalf("javaString(Path) = %q, want alpha/beta", got)
	}
}

func TestFilesReadAllLinesMatchesJavaLineSplitting(t *testing.T) {
	dir := t.TempDir()
	for _, test := range []struct {
		name    string
		content string
		want    []string
	}{
		{"trailing newline", "a\nb\n", []string{"a", "b"}},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
		{"empty file", "", nil},
		{"blank lines preserved", "a\n\nb\n", []string{"a", "", "b"}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(test.name, " ", "_"))
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got := FilesReadAllLines(path).Slice()
			if len(got) != len(test.want) {
				t.Fatalf("lines = %q, want %q", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("lines = %q, want %q", got, test.want)
				}
			}
		})
	}
}
