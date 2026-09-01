package fuzz

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CorpusDir returns the corpus root under the module root.
func CorpusDir(root string) string {
	return filepath.Join(root, "fuzz", "corpus")
}

// categoryDir maps a divergence category to its on-disk subdirectory name.
func categoryDir(c Category) string {
	switch c {
	case TranspileCrash:
		return "transpile_crash"
	case GoCompileError:
		return "go_compile_error"
	case GoRuntimeError:
		return "go_runtime_error"
	case OutputMismatch:
		return "output_mismatch"
	default:
		return "other"
	}
}

// CorpusEntry is one saved divergence: the (shrunk) Java program plus the Java
// and Go outputs at the time it was found.
type CorpusEntry struct {
	Category Category
	Seed     int64
	Source   string
	Expected string // Java stdout (the oracle)
	Actual   string // Go stdout (or the error detail for *_ERROR categories)
	Path     string // .java path once saved
}

// Save writes a divergence to fuzz/corpus/<category>/<name>.java with sibling
// .expected and .actual files. The file name embeds the seed and a short content
// hash so re-finding the same shrunk program twice does not create a duplicate.
func (e CorpusEntry) Save(root string) (string, error) {
	return e.SaveNamed(root, corpusName(e.Seed, e.Source))
}

// SaveNamed is Save with a caller-chosen file stem, used to give curated minimal
// repros descriptive filenames.
func (e CorpusEntry) SaveNamed(root, name string) (string, error) {
	dir := filepath.Join(CorpusDir(root), categoryDir(e.Category))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	javaPath := filepath.Join(dir, name+".java")
	if err := os.WriteFile(javaPath, []byte(e.Source), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, name+".expected"), []byte(e.Expected), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, name+".actual"), []byte(e.Actual), 0o644); err != nil {
		return "", err
	}
	return javaPath, nil
}

// corpusName builds a stable, collision-resistant file stem from the seed and a
// content hash, so the same shrunk repro always maps to the same file.
func corpusName(seed int64, source string) string {
	sum := sha1.Sum([]byte(source))
	return fmt.Sprintf("seed%d_%s", seed, hex.EncodeToString(sum[:4]))
}

// Fingerprint returns a content hash of the (shrunk) source, used to dedupe
// divergences by their minimized form rather than by originating seed.
func Fingerprint(source string) string {
	sum := sha1.Sum([]byte(source))
	return hex.EncodeToString(sum[:8])
}

// LoadCorpus walks the corpus directory and returns every saved entry, so a
// replay test can re-run them all.
func LoadCorpus(root string) ([]CorpusEntry, error) {
	base := CorpusDir(root)
	var entries []CorpusEntry
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		expected, _ := os.ReadFile(strings.TrimSuffix(path, ".java") + ".expected")
		actual, _ := os.ReadFile(strings.TrimSuffix(path, ".java") + ".actual")
		cat := categoryFromDir(filepath.Base(filepath.Dir(path)))
		entries = append(entries, CorpusEntry{
			Category: cat,
			Source:   string(src),
			Expected: string(expected),
			Actual:   string(actual),
			Path:     path,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func categoryFromDir(name string) Category {
	switch name {
	case "transpile_crash":
		return TranspileCrash
	case "go_compile_error":
		return GoCompileError
	case "go_runtime_error":
		return GoRuntimeError
	case "output_mismatch":
		return OutputMismatch
	default:
		return OK
	}
}
