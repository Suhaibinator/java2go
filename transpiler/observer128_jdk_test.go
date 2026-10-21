package transpiler

import (
	"encoding/json"
	"os"
	"testing"
)

func observer128ProjectOracle(t *testing.T, name string) {
	t.Helper()
	raw, err := os.ReadFile("testdata/observer128_programs.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Programs map[string]struct {
			Files                 map[string]string `json:"files"`
			Entry, Stdout, Stderr string
		}
	}
	if err = json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	p, ok := data.Programs[name]
	if !ok || p.Stderr != "" {
		t.Fatal("missing/faulty frozen oracle")
	}
	runCampaignCompilerStrictProjectOracle(t, p.Files, p.Entry, p.Stdout)
}
func TestObserver128CanonicalSuperJDK(t *testing.T) { observer128ProjectOracle(t, "CanonicalSuper") }
func TestObserver128GSEImportedBindersJDK(t *testing.T) {
	observer128ProjectOracle(t, "GSEImportedBinders")
}
func TestObserver128GSEMinimalJDK(t *testing.T) { observer128ProjectOracle(t, "GSEMinimal") }
func TestObserver128SourceParentSuperJDK(t *testing.T) {
	observer128ProjectOracle(t, "SourceParentSuper")
}
func TestObserver128URIImportedChildJDK(t *testing.T) {
	observer128ProjectOracle(t, "URIImportedChild")
}
func TestObserver128URIMinimalJDK(t *testing.T) { observer128ProjectOracle(t, "URIMinimal") }
