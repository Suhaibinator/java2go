package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Preserve the complete seven-member project and the original whole JVM
// observations first frozen three times each before assertion evolution.
func TestBoxedGenericIdentityFullProjectJDK21(t *testing.T) {
	root := filepath.Join("testdata", "boxed_generic_identity136")
	files := map[string]string{}
	for _, name := range []string{"pom.xml", "src/main/java/app/Main.java", "src/main/java/generic/Identity.java", "src/main/java/model/State.java", "src/main/java/model/Resource.java", "src/main/java/shadow/Integer.java", "src/main/java/shadow/Probe.java"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(data)
	}
	var observations []campaignCompilerProjectObservation
	for _, seed := range []int{17, 41, 97} {
		want, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("expected.seed-%d.stdout", seed)))
		if err != nil {
			t.Fatal(err)
		}
		for repeat := 1; repeat <= 3; repeat++ {
			observations = append(observations, campaignCompilerProjectObservation{name: fmt.Sprintf("seed%d-repeat%d", seed, repeat), args: []string{fmt.Sprint(seed)}, stdout: string(want)})
		}
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", observations)
}
