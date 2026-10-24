package stdjava

import (
	"path/filepath"
	"testing"
)

func TestCIWriterPathLegacyOpenFailureRetainsIOException(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "leaf")
	for _, destination := range []any{path, NewJavaFile(path)} {
		func() {
			defer func() {
				if thrown := recover(); thrown == nil {
					t.Fatal("missing parent directory did not throw")
				} else if _, ok := thrown.(IOException); !ok {
					t.Fatalf("legacy destination %T threw %T, want IOException", destination, thrown)
				}
			}()
			NewPrintWriter(destination)
		}()
	}
}
