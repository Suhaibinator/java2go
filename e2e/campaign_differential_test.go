//go:build campaign

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/NickyBoy89/java2go/campaign"
)

// Run explicitly with -tags=campaign. Every discovered fixture is mandatory:
// missing dependencies, unsupported code, known gaps and timeouts all fail.
func TestCampaignDifferential(t *testing.T) {
	root := moduleRoot(t)
	var fixtures []string
	err := filepath.WalkDir(filepath.Join(root, "testfiles/campaign"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "fixture.json" {
			fixtures = append(fixtures, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no campaign fixtures discovered")
	}
	sort.Strings(fixtures)
	for _, fixture := range fixtures {
		relative, _ := filepath.Rel(root, fixture)
		t.Run(relative, func(t *testing.T) {
			report, err := campaign.Run(context.Background(), campaign.Config{Repository: root, Fixture: fixture, Race: true, StressRuns: 20})
			t.Logf("campaign evidence: %s", report.ArtifactDirectory)
			if err != nil {
				t.Fatal(err)
			}
			if !report.Passed || len(report.Observations) != 9 || !report.RaceEnabled || len(report.StressObservations) != 20 {
				t.Fatalf("incomplete campaign run: %+v", report.Failure)
			}
		})
	}
}
