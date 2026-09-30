package transpiler

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCanonicalStringSearchJVM21(t *testing.T) {
	fixture := filepath.Join("testdata", "canonical_string_search")
	files := make(map[string]string)
	for _, name := range []string{"SearchProbe.java", "CampaignStringSearch.java", "foreign/String.java"} {
		content, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		files["src/main/java/"+name] = string(content)
	}
	pom, err := os.ReadFile(filepath.Join(fixture, "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	files["pom.xml"] = string(pom)
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
	runCampaignCompilerStrictProjectObservations(t, files, "SearchProbe", observations)
}
