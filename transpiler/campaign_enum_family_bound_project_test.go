package transpiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Retain the independently authored Enum-family project and all nine captured
// JVM processes. The runtime observations cover the F-bound factory and raw pollution,
// enum metadata, consumer checkcasts, aliases, generic methods and null storage.
func TestCampaignEnumFamilyBoundProjectJDK21(t *testing.T) {
	root := filepath.Join("testdata", "campaign", "enum_family_bound_v3")
	var provenance struct {
		MainClass    string            `json:"main_class"`
		ProjectFiles map[string]string `json:"project_files"`
		Observations []struct {
			Seed         int    `json:"seed"`
			Repeat       int    `json:"repeat"`
			Stdout       string `json:"stdout"`
			Stderr       string `json:"stderr"`
			StdoutSHA256 string `json:"stdout_sha256"`
			StderrSHA256 string `json:"stderr_sha256"`
		} `json:"observations"`
	}
	encoded, err := os.ReadFile(filepath.Join(root, "provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &provenance); err != nil {
		t.Fatal(err)
	}
	readPinned := func(name, wantHash string) string {
		t.Helper()
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(contents)
		if got := hex.EncodeToString(sum[:]); got != wantHash {
			t.Fatalf("frozen fixture %s: SHA256=%s, want %s", name, got, wantHash)
		}
		return string(contents)
	}
	files := make(map[string]string, len(provenance.ProjectFiles))
	for name, wantHash := range provenance.ProjectFiles {
		files[name] = readPinned("project/"+name, wantHash)
	}
	if len(provenance.Observations) != 9 {
		t.Fatalf("want nine frozen JVM observations, got %d", len(provenance.Observations))
	}
	observations := make([]campaignCompilerProjectObservation, 0, 9)
	for index, recorded := range provenance.Observations {
		wantSeed := []int{17, 41, 97}[index/3]
		wantRepeat := index%3 + 1
		if recorded.Seed != wantSeed || recorded.Repeat != wantRepeat {
			t.Fatalf("observation %d: seed/repeat=%d/%d, want %d/%d", index, recorded.Seed, recorded.Repeat, wantSeed, wantRepeat)
		}
		observations = append(observations, campaignCompilerProjectObservation{
			name:   fmt.Sprintf("seed-%d-repeat-%d", recorded.Seed, recorded.Repeat),
			args:   []string{strconv.Itoa(recorded.Seed)},
			stdout: readPinned(recorded.Stdout, recorded.StdoutSHA256),
			stderr: readPinned(recorded.Stderr, recorded.StderrSHA256),
		})
	}
	runCampaignCompilerStrictProjectObservations(t, files, provenance.MainClass, observations)
}
