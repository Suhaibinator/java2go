package transpiler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const gsonGenericParentFixtureDirectory = "testdata/gson_generic_parent_origin"
const gsonGenericParentFixtureManifestSHA = "b7d03936a9ef1363efef9444bbcb4a9fe99d29b5b2f6a55b9ff05729fce51b86"

func gsonGenericParentInputRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("JAVA2GO_GSON_ORIGIN_INPUT")
	if root == "" {
		root = filepath.FromSlash(gsonGenericParentFixtureDirectory)
	}
	if err := verifyGsonGenericParentFixture(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func verifyGsonGenericParentFixture(root string) error {
	manifest, err := os.ReadFile(filepath.Join(filepath.FromSlash(gsonGenericParentFixtureDirectory), "SOURCE-MANIFEST.json"))
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", sha256.Sum256(manifest)) != gsonGenericParentFixtureManifestSHA {
		return fmt.Errorf("original Gson114 manifest byte pin mismatch")
	}
	var inputs map[string]string
	if err := json.Unmarshal(manifest, &inputs); err != nil {
		return err
	}
	if len(inputs) != 114 {
		return fmt.Errorf("original Gson fixture has %d source pins; want114", len(inputs))
	}
	for name, expected := range inputs {
		if filepath.IsAbs(name) || strings.Contains(name, "..") || !strings.HasSuffix(name, ".java") {
			return fmt.Errorf("invalid original Gson source manifest path %q", name)
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != expected {
			return fmt.Errorf("original Gson source byte pin mismatch: %s", name)
		}
	}
	return nil
}

func TestGsonGenericParentFixturePortability(t *testing.T) {
	t.Setenv("JAVA2GO_GSON_ORIGIN_INPUT", "")
	if root := gsonGenericParentInputRoot(t); root != filepath.FromSlash(gsonGenericParentFixtureDirectory) {
		t.Fatal("unset override must use checked-in original source fixture")
	}
	absolute, err := filepath.Abs(filepath.FromSlash(gsonGenericParentFixtureDirectory))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("JAVA2GO_GSON_ORIGIN_INPUT", absolute)
	if gsonGenericParentInputRoot(t) != absolute {
		t.Fatal("explicit byte-identical fixture override must remain usable")
	}
	if err := verifyGsonGenericParentFixture(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing source root admitted")
	}
	copied := t.TempDir()
	err = filepath.WalkDir(filepath.FromSlash(gsonGenericParentFixtureDirectory), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".java" {
			return nil
		}
		relative, err := filepath.Rel(filepath.FromSlash(gsonGenericParentFixtureDirectory), path)
		if err != nil {
			return err
		}
		target := filepath.Join(copied, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyGsonGenericParentFixture(copied); err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(copied, "src/main/java/snapshot/app/Main.java")
	data, err := os.ReadFile(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(changed, append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyGsonGenericParentFixture(copied); err == nil {
		t.Fatal("modified fixture override must be rejected")
	}
}
