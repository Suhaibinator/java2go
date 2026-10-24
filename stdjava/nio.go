package stdjava

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// This file implements the java.nio.file surface that modern Java code reaches
// for instead of java.io: Path/Paths and the static Files helpers. It is layered
// on the same os-backed plumbing as io.go, and shares that file's documented
// approximations:
//
//   - Checked IOExceptions are not threaded as Go errors; a failure panics with
//     a stdjava IOException.
//   - UTF-8 only; the charset-taking overloads are not modeled.
//   - Path syntax is the Unix one ("/" separator, a single "/" root). Windows
//     drive letters and UNC paths are not modeled.
//   - Every Files entry point takes its path as `any` and accepts either a
//     *JavaPath or a plain path string, since transpiled code reaches these
//     methods with both; the same ioPathOf helper resolves both here and in
//     io.go.
//
// Deliberately out of scope: OpenOption/CopyOption/LinkOption flags (Files.copy
// and Files.move here always replace an existing target), FileSystem/FileStore,
// channels and buffers, WatchService, symbolic-link-aware traversal
// (Files.walk/walkFileTree), and file permissions and attributes
// (PosixFilePermission, BasicFileAttributes).

// JavaPath models java.nio.file.Path: an abstract path, not an open handle.
type JavaPath struct {
	path string
}

// NewJavaPath returns a Path for an already-assembled pathname. Java code
// reaches Paths through Paths.get / Path.of, which normalize their arguments;
// this entry point exists for internal conversions such as File.toPath.
func NewJavaPath(path string) *JavaPath {
	return &JavaPath{path: path}
}

// PathsGet joins the given components into a Path, matching both Paths.get and
// Path.of. Redundant and trailing separators are collapsed the way Java does,
// but "." and ".." are kept — only Normalize removes them.
func PathsGet(first string, more ...string) *JavaPath {
	return &JavaPath{path: joinJavaPath(append([]string{first}, more...))}
}

// joinJavaPath implements Paths.get's assembly rule: split every component on
// the separator, drop empty elements, and rejoin, preserving whether the result
// is absolute.
func joinJavaPath(parts []string) string {
	absolute := len(parts) > 0 && strings.HasPrefix(parts[0], "/")
	elements := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, element := range strings.Split(part, "/") {
			if element != "" {
				elements = append(elements, element)
			}
		}
	}
	joined := strings.Join(elements, "/")
	if absolute {
		return "/" + joined
	}
	return joined
}

// pathElements returns the name elements of a path, i.e. the components after
// any root. Java gives the empty path one, empty, name element and the root
// none, which is what makes startsWith("") false for every other path.
func (p *JavaPath) pathElements() []string {
	if p.path == "" {
		return []string{""}
	}
	trimmed := strings.TrimPrefix(p.path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// isAbsolute reports whether the path starts at the root.
func (p *JavaPath) isAbsolute() bool {
	return strings.HasPrefix(p.path, "/")
}

// ToString returns the pathname, matching Path.toString.
func (p *JavaPath) ToString() string {
	return p.path
}

// String makes a Path render as its pathname through fmt and through the
// javaString helper that the print shims use.
func (p *JavaPath) String() string {
	return p.ToString()
}

// GetFileName returns the last name element as a Path, matching
// Path.getFileName. It returns nil (Java's null) for a path with no elements,
// i.e. the root.
func (p *JavaPath) GetFileName() *JavaPath {
	elements := p.pathElements()
	if len(elements) == 0 {
		return nil
	}
	return &JavaPath{path: elements[len(elements)-1]}
}

// GetParent returns the path without its last element, matching Path.getParent.
// It returns nil (Java's null) when there is no parent.
func (p *JavaPath) GetParent() *JavaPath {
	elements := p.pathElements()
	if len(elements) == 0 {
		return nil
	}
	if len(elements) == 1 {
		if p.isAbsolute() {
			return &JavaPath{path: "/"}
		}
		return nil
	}
	parent := strings.Join(elements[:len(elements)-1], "/")
	if p.isAbsolute() {
		parent = "/" + parent
	}
	return &JavaPath{path: parent}
}

// GetNameCount returns the number of name elements, matching Path.getNameCount.
func (p *JavaPath) GetNameCount() int32 {
	return int32(len(p.pathElements()))
}

// Resolve appends other to this path, matching Path.resolve: an absolute other
// replaces this path, and an empty other leaves it unchanged. other may be a
// *JavaPath or a path string.
func (p *JavaPath) Resolve(other any) *JavaPath {
	suffix := ioPathOf(other)
	if strings.HasPrefix(suffix, "/") {
		return &JavaPath{path: joinJavaPath([]string{suffix})}
	}
	if suffix == "" {
		return &JavaPath{path: p.path}
	}
	if p.path == "" {
		return &JavaPath{path: joinJavaPath([]string{suffix})}
	}
	return &JavaPath{path: joinJavaPath([]string{p.path, suffix})}
}

// ToAbsolutePath resolves this path against the working directory, matching
// Path.toAbsolutePath.
func (p *JavaPath) ToAbsolutePath() *JavaPath {
	if p.isAbsolute() {
		return &JavaPath{path: p.path}
	}
	abs, err := filepath.Abs(p.path)
	if err != nil {
		return &JavaPath{path: p.path}
	}
	return &JavaPath{path: abs}
}

// Normalize removes "." elements and resolves "..", matching Path.normalize.
// Like Java's, this is purely lexical: no symbolic link is consulted.
func (p *JavaPath) Normalize() *JavaPath {
	if p.path == "" {
		return &JavaPath{path: ""}
	}
	cleaned := filepath.Clean(p.path)
	if cleaned == "." {
		// Java's normalize drops "." entirely, leaving the empty path.
		cleaned = ""
	}
	return &JavaPath{path: cleaned}
}

// StartsWith reports whether this path begins with the given path, matching
// Path.startsWith. The comparison is element-by-element, so "a/bc" does not
// start with "a/b". other may be a *JavaPath or a path string.
func (p *JavaPath) StartsWith(other any) bool {
	prefix := &JavaPath{path: joinJavaPath([]string{ioPathOf(other)})}
	if prefix.isAbsolute() != p.isAbsolute() {
		return false
	}
	prefixElements := prefix.pathElements()
	elements := p.pathElements()
	if len(prefixElements) > len(elements) {
		return false
	}
	for i, element := range prefixElements {
		if elements[i] != element {
			return false
		}
	}
	return true
}

// EndsWith reports whether this path ends with the given path, matching
// Path.endsWith. Like startsWith the comparison is element-by-element; an
// absolute argument must match this path in full.
func (p *JavaPath) EndsWith(other any) bool {
	suffix := &JavaPath{path: joinJavaPath([]string{ioPathOf(other)})}
	suffixElements := suffix.pathElements()
	elements := p.pathElements()
	if suffix.isAbsolute() {
		return p.isAbsolute() && len(suffixElements) == len(elements) && p.path == suffix.path
	}
	if len(suffixElements) > len(elements) {
		return false
	}
	offset := len(elements) - len(suffixElements)
	for i, element := range suffixElements {
		if elements[offset+i] != element {
			return false
		}
	}
	return true
}

// ToFile returns this path as a java.io.File, matching Path.toFile.
func (p *JavaPath) ToFile() *JavaFile {
	return NewJavaFile(p.path)
}

// The Java String overloads validate through the Unix Path provider before
// comparing components. Canonical Strings never pass through host formatting.
func (p *JavaPath) StartsWithReference(other any) bool {
	ReferenceRequireNonNull(p)
	ReferenceRequireNonNull(other)
	if text, ok := other.(*JavaString); ok {
		other = PathsGetReference(text)
	}
	return p.StartsWith(other)
}

func (p *JavaPath) EndsWithReference(other any) bool {
	ReferenceRequireNonNull(p)
	ReferenceRequireNonNull(other)
	if text, ok := other.(*JavaString); ok {
		other = PathsGetReference(text)
	}
	return p.EndsWith(other)
}

// --- java.nio.file.Files ----------------------------------------------------

// FilesReadAllLines reads every line of a file into a List, matching
// Files.readAllLines. Line terminators are not included and a trailing newline
// does not produce an extra empty element, as in Java.
func FilesReadAllLines(path any) *List[string] {
	return NewListFrom(filesLineSlice(path)...)
}

func FilesReadAllLinesReference(path any) *List[*JavaString] {
	return NewListFrom(filesLineReferences(path)...)
}

// Canonical line imports use the reporting CharsetDecoder policy, which groups
// malformed input differently from Files.readString's optimized String decoder.
func filesLineReferences(path any) []*JavaString {
	ReferenceRequireNonNull(path)
	data, err := os.ReadFile(ioPathOf(path))
	if err != nil {
		throwIOException(err)
	}
	for offset := 0; offset < len(data); {
		point, size, _ := javaUTF8Unit(data[offset:], true)
		if point == utf8.RuneError && !utf8.Valid(data[offset:offset+size]) {
			panic(newThrowableBase("MalformedInputException", fmt.Sprintf("Input length = %d", size)))
		}
		offset += size
	}
	if len(data) == 0 {
		return nil
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if strings.HasSuffix(normalized, "\n") {
		lines = lines[:len(lines)-1]
	}
	result := make([]*JavaString, len(lines))
	for index, line := range lines {
		result[index] = JavaStringFromHostUTF8(line)
	}
	return result
}

func FilesLinesReference(path any) Stream[*JavaString] {
	return StreamOfSlice(filesLineReferences(path))
}

// filesLineSlice is the shared line-splitting backend of readAllLines and lines.
func filesLineSlice(path any) []string {
	reader := NewBufferedReader(ioPathOf(path))
	defer reader.Close()
	var lines []string
	for {
		line, ok := reader.ReadLineOK()
		if !ok {
			return lines
		}
		lines = append(lines, line)
	}
}

// FilesLines returns the lines of a file as a Stream, matching Files.lines.
// Java's stream is lazy and must be closed; this one reads the file eagerly and
// closes it before returning, so a missing close in the source cannot leak.
func FilesLines(path any) Stream[string] {
	return StreamOfSlice(filesLineSlice(path))
}

// FilesReadString reads a whole file as UTF-8 text, matching Files.readString.
func FilesReadString(path any) string {
	data, err := os.ReadFile(ioPathOf(path))
	if err != nil {
		throwIOException(err)
	}
	return string(data)
}

// FilesReadStringReference imports file bytes into the canonical Java String ABI.
func FilesReadStringReference(path any) *JavaString {
	ReferenceRequireNonNull(path)
	data, err := os.ReadFile(ioPathOf(path))
	if err != nil {
		throwIOException(err)
	}
	if len(data) == 0 {
		return JavaStringLiteralUTF16(nil)
	}
	for offset := 0; offset < len(data); {
		point, size := utf8.DecodeRune(data[offset:])
		if point == utf8.RuneError && size == 1 {
			remaining := data[offset:]
			malformedLength := 1
			lead := remaining[0]
			// Files.readString uses the JDK String decoder's strict path. Complete
			// malformed units report their full width; truncated units follow its
			// separate three-byte second-byte check rather than replacement grouping.
			if lead >= 0xe0 && lead <= 0xef {
				if len(remaining) >= 3 {
					malformedLength = 3
				} else if len(remaining) == 2 && (remaining[1]&0xc0 != 0x80 || lead == 0xe0 && remaining[1] < 0xa0) {
					malformedLength = 2
				}
			} else if lead >= 0xf0 && lead <= 0xf7 && len(remaining) >= 4 {
				malformedLength = 4
			}
			panic(newThrowableBase("MalformedInputException", fmt.Sprintf("Input length = %d", malformedLength)))
		}
		offset += size
	}
	return JavaStringFromHostUTF8(string(data))
}

func init() { RegisterException("MalformedInputException", "CharacterCodingException") }

// FilesWriteString writes text to a file, creating or truncating it, matching
// Files.writeString. It returns the path, as Java does.
func FilesWriteString(path any, content string) *JavaPath {
	target := ioPathOf(path)
	if err := os.WriteFile(target, []byte(content), 0o666); err != nil {
		throwIOException(err)
	}
	return &JavaPath{path: target}
}

// FilesWrite writes bytes or lines to a file, creating or truncating it,
// matching both Files.write(path, byte[]) and
// Files.write(path, Iterable<CharSequence>). Each line of the iterable form is
// followed by a newline, as in Java. It returns the path.
func FilesWrite(path any, content any) *JavaPath {
	target := ioPathOf(path)
	var data []byte
	switch v := content.(type) {
	case *List[*JavaString]:
		ReferenceRequireNonNull(v)
		for _, line := range v.Slice() {
			if line == nil {
				line = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
			}
			data = append(data, unsignedBytes(JavaStringGetBytes(line, UTF_8).Elements)...)
			data = append(data, '\n')
		}
	case *List[string]:
		data = []byte(strings.Join(v.Slice(), "\n"))
		if len(v.Slice()) > 0 {
			data = append(data, '\n')
		}
	default:
		data = javaBytes(content)
	}
	if err := os.WriteFile(target, data, 0o666); err != nil {
		throwIOException(err)
	}
	return &JavaPath{path: target}
}

// FilesExists reports whether the file or directory exists, matching
// Files.exists.
func FilesExists(path any) bool {
	_, err := os.Stat(ioPathOf(path))
	return err == nil
}

// FilesIsDirectory reports whether the path is a directory, matching
// Files.isDirectory.
func FilesIsDirectory(path any) bool {
	info, err := os.Stat(ioPathOf(path))
	return err == nil && info.IsDir()
}

// FilesIsRegularFile reports whether the path is a regular file, matching
// Files.isRegularFile.
func FilesIsRegularFile(path any) bool {
	info, err := os.Stat(ioPathOf(path))
	return err == nil && info.Mode().IsRegular()
}

// FilesSize returns the file size in bytes, matching Files.size. Unlike
// File.length it throws rather than returning 0 for a missing file.
func FilesSize(path any) int64 {
	info, err := os.Stat(ioPathOf(path))
	if err != nil {
		throwIOException(err)
	}
	return info.Size()
}

// FilesCreateDirectories creates the directory and any missing parents,
// matching Files.createDirectories. An already-existing directory is not an
// error, as in Java. It returns the path. Files.createDirectory, which refuses
// to create parents and throws for an existing directory, is not modeled.
func FilesCreateDirectories(path any) *JavaPath {
	target := ioPathOf(path)
	if err := os.MkdirAll(target, 0o777); err != nil {
		throwIOException(err)
	}
	return &JavaPath{path: target}
}

// FilesCreateFile creates an empty file, matching Files.createFile. It throws
// if the file already exists (Java's FileAlreadyExistsException is reported as
// an IOException, its supertype). It returns the path.
func FilesCreateFile(path any) *JavaPath {
	target := ioPathOf(path)
	file, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		throwIOException(err)
	}
	_ = file.Close()
	return &JavaPath{path: target}
}

// FilesDelete removes a file or empty directory, matching Files.delete, which
// throws when the target does not exist.
func FilesDelete(path any) {
	if err := os.Remove(ioPathOf(path)); err != nil {
		throwIOException(err)
	}
}

// FilesDeleteIfExists removes a file or empty directory if present and reports
// whether it did, matching Files.deleteIfExists.
func FilesDeleteIfExists(path any) bool {
	err := os.Remove(ioPathOf(path))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	throwIOException(err)
	return false
}

// FilesCopy copies a regular file, matching Files.copy. Java refuses to
// overwrite unless REPLACE_EXISTING is passed; since copy options are not
// modeled, an existing target is always replaced.
func FilesCopy(source, target any) *JavaPath {
	destination := ioPathOf(target)
	data, err := os.ReadFile(ioPathOf(source))
	if err != nil {
		throwIOException(err)
	}
	if err := os.WriteFile(destination, data, 0o666); err != nil {
		throwIOException(err)
	}
	return &JavaPath{path: destination}
}

// FilesMove moves or renames a file, matching Files.move. As with FilesCopy,
// copy options are not modeled, so an existing target is always replaced.
func FilesMove(source, target any) *JavaPath {
	destination := ioPathOf(target)
	if err := os.Rename(ioPathOf(source), destination); err != nil {
		throwIOException(err)
	}
	return &JavaPath{path: destination}
}
