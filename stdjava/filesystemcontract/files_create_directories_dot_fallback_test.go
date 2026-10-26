//go:build darwin || linux

package directoryoracle_test

import (
	j "github.com/NickyBoy89/java2go/stdjava"
	"os"
	"path/filepath"
	"testing"
)

// Three fresh public-JDK21 observations: create only leaf; return the same
// original a/../leaf Path, whose directory check remains false because a is
// absent. This exercises fallback relativize, not the first mkdir fast path.
func TestFilesCreateDirectoriesMissingIntermediateDotDot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "owned")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	path := j.NewJavaPath(root + "/a/../leaf")
	result, class := directoryInvoke(path)
	if class != "" {
		t.Fatalf("exception=%q, JDK21 returns normally", class)
	}
	if result != path {
		t.Error("result identity differs from JDK21")
	}
	if j.FilesIsDirectory(result) {
		t.Error("directory=true, JDK21=false for unchanged original path")
	}
	if got := directorySnapshot(t, root); got != ".:D|leaf:D" {
		t.Errorf("after=%q, JDK21=%q", got, ".:D|leaf:D")
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		t.Errorf("cleanup=%v", err)
	}
}
