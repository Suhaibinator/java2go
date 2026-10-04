package transpiler

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCanonicalStringReplaceEmptyIdentityJVM(t *testing.T) {
	fixture := filepath.Join("testdata", "replace_empty_identity_prerequisite")
	source, err := os.ReadFile(filepath.Join(fixture, "Probe.java"))
	if err != nil {
		t.Fatal(err)
	}
	pom, err := os.ReadFile(filepath.Join(fixture, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var observations []campaignCompilerProjectObservation
	for _, seed := range []int{17, 41, 97} {
		expected, err := os.ReadFile(filepath.Join(fixture, "expected.seed-"+strconv.Itoa(seed)+".stdout"))
		if err != nil {
			t.Fatal(err)
		}
		for repeat := 1; repeat <= 3; repeat++ {
			observations = append(observations, campaignCompilerProjectObservation{
				name:   strconv.Itoa(seed) + "-" + strconv.Itoa(repeat),
				args:   []string{strconv.Itoa(seed)},
				stdout: string(expected),
			})
		}
	}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml":                              string(pom),
		"src/main/java/deleteempty/Probe.java": string(source),
	}, "deleteempty.Probe", observations)
}
