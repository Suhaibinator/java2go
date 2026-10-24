package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Supplementary ordinary Java evidence for the bounded public diamond fix.
// Every application/source class is nongeneric. This does not replace the
// separately held original Class<Collection<T>> or Date+Collection<T> fixtures.
// Expected files must be captured from fresh JDK21 runs of the frozen source.
func TestCollectionDiamondPublic201OrdinaryProjectOracle(t *testing.T) {
	root := filepath.Join("testdata", "collection_diamond_public201")
	project := filepath.Join(root, "project")
	files := map[string]string{}
	err := filepath.WalkDir(project, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[relative] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var observations []campaignCompilerProjectObservation
	for _, seed := range []string{"17", "41", "97"} {
		stdout, err := os.ReadFile(filepath.Join(root, "expected", seed+".stdout"))
		if err != nil {
			t.Fatal(err)
		}
		observations = append(observations, campaignCompilerProjectObservation{name: "seed-" + seed, args: []string{seed}, stdout: string(stdout)})
	}
	runCampaignCompilerStrictProjectObservations(t, files, "probe.app.Main", observations)
}
