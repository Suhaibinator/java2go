package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This is the complete frozen supplemental project, including all seeded
// observations first executed three times each on JDK21 before test evolution.
func TestInstanceGenericMethodPointerFullProjectJDK21(t *testing.T) {
	root := filepath.Join("testdata", "instance_generic_pointer_identity136")
	files := map[string]string{}
	for _, name := range []string{"pom.xml", "src/main/java/app/Main.java", "src/main/java/bridge/Box.java", "src/main/java/model/Foo.java", "src/main/java/foreign/Foo.java", "src/main/java/bridge/Controls.java", "src/main/java/model/Cleanup.java"} {
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
	runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t, files, "app.Main", observations, 30*time.Second)
}
