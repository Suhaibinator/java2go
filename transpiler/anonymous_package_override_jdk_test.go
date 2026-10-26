package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnonymousPackageOverrideJDK21(t *testing.T) {
	files := map[string]string{}
	root := filepath.Join("testdata", "anonymous_package_override", "project")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, main, want string }{
		{"same-package", "dispatchproof.AnonymousOverrideWitness", "41:42:2\n"},
		{"dual-selector", "dispatchproof.DualSelectorWitness", "5:7\n"},
		{"alias-collision", "dispatchproof.AliasCollisionWitness", "9:5:7\n"},
		{"hidden-alias-collision", "dispatchproof.HiddenAliasCollisionWitness", "9:5:7\n"},
		{"foreign-package", "dispatchproof.ForeignPackageControl", "1:2\n"},
	} {
		t.Run(tc.name, func(t *testing.T) { runCampaignCompilerStrictProjectOracle(t, files, tc.main, tc.want) })
	}
}

func TestSyntheticInheritedSourceControlsJDK21(t *testing.T) {
	root := filepath.Join("testdata", "anonymous_package_override", "project")
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, main, want string }{
		{"local-and-named", "dispatchproof.LocalOverrideWitness", "11:12\n"},
		{"foreign-signature-owner", "dispatchproof.SignatureWitness", "1:2\n"},
	} {
		t.Run(tc.name, func(t *testing.T) { runCampaignCompilerStrictProjectOracle(t, files, tc.main, tc.want) })
	}
}
