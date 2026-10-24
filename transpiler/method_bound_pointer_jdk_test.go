package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func runMethodBoundPointerProposalProject(t *testing.T, fixture, mainClass string) {
	t.Helper()
	root := filepath.Join("testdata", "method_bound_pointer", fixture)
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>method-bound-pointer</artifactId><version>1</version></project>"}
	if err := filepath.WalkDir(filepath.Join(root, "src"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, mainClass, []campaignCompilerProjectObservation{{name: "repeat1", stdout: string(want)}, {name: "repeat2", stdout: string(want)}, {name: "repeat3", stdout: string(want)}})
}
func TestMethodBoundPointerDispatchOrderJDK21(t *testing.T) {
	runMethodBoundPointerProposalProject(t, "ordinary-dispatch-order", "receiverorder.Main")
}
func TestMethodBoundPointerImportedNominalTransitiveJDK21(t *testing.T) {
	runMethodBoundPointerProposalProject(t, "imported-nominal-and-transitive", "Main")
}
func TestMethodBoundPointerSynchronizedExecutionJDK21(t *testing.T) {
	runMethodBoundPointerProposalProject(t, "synchronized-execution", "receiverlock.Main")
}

func TestMethodBoundPointerOriginalAnimalJDK21(t *testing.T) {
	runMethodBoundPointerProposalProject(t, "original-Animal", "app.Main")
}
