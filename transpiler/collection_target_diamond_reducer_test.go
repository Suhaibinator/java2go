package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Existing renderer API; the fixture is a separate reducer of the preserved
// original all-packages build failure, not a replacement acceptance fixture.
func TestCollectionTargetDiamondReducer(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "collection_target_diamond_v1", "CollectionDiamondReducer.java"))
	if err != nil {
		t.Fatal(err)
	}
	out := renderGoFileFromJava(t, string(source))
	if !strings.Contains(out, "stdjava.NewArrayList[*stdjava.JavaString]()") {
		t.Fatalf("Collection<String> target lost canonical ArrayList diamond argument; generated source:\n%s", out)
	}
}

// This distinct control proves an explicit argument already reaches the same
// canonical constructor. It cannot stand in for the diamond acceptance case.
func TestCollectionTargetExplicitArgumentControl(t *testing.T) {
	const source = `class ExplicitControl{static void run(){java.util.Collection<String> values=new java.util.ArrayList<String>();values.add("control");}}`
	out := renderGoFileFromJava(t, source)
	if !strings.Contains(out, "stdjava.NewArrayList[*stdjava.JavaString]()") {
		t.Fatalf("explicit constructor argument did not reach existing intrinsic: %s", out)
	}
}
