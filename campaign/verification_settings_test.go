package campaign

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A run that fails before compilation must still record which verification gate
// was requested, otherwise its saved evidence can be mistaken for another gate.
func TestVerificationSettingsSurviveFailedRun(t *testing.T) {
	root := t.TempDir()
	report, err := Run(context.Background(), Config{Repository: root, Fixture: root, Artifacts: filepath.Join(root, "runs"), Race: true, StressRuns: 20})
	if err == nil || report.Passed {
		t.Fatal("missing fixture must fail")
	}
	data, readErr := os.ReadFile(filepath.Join(report.ArtifactDirectory, "report.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	var saved Report
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.RaceEnabled || saved.StressRuns != 20 {
		t.Fatalf("lost requested verification settings: race=%v stress=%d", saved.RaceEnabled, saved.StressRuns)
	}
}
