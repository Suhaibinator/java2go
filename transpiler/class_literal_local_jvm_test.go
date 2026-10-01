package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Local definitions take the hoist path rather than ParseDecls. Check actual
// JDK behavior for scalar and array literals, allocation identity, block-local
// name shadowing, and class initialization separately from instance creation.
func TestClassLiteralLocalDeclarationsJDK21(t *testing.T) {
	campaignCollectionReferenceJDK21(t)
	root := filepath.Join("testdata", "classliteral_local94", "project")
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(content)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Captured from the unchanged valid standalone JDK21 fixture before repair.
	runCampaignCompilerStrictProjectOracle(t, files, "local.app.Main", "before:0:0:true:false:true:true\narrays:true:true:true\nafter:0:1:true\nshadow:true:true:true\ninitialized:1:true\nnominal:true:true\n")
}
