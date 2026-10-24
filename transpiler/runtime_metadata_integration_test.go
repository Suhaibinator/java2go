package transpiler

import (
	"os"
	"testing"
)

func TestRuntimeMetadataWiringApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/runtime_metadata_wiring/src/RuntimeMetadataWiring.java")
	if err != nil {
		t.Fatal(err)
	}
	out := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, out, `package metadatawiring
import (
    "testing"
    "unicode/utf16"
)
func TestWiring(t *testing.T) {
    want := "parity.metadatawiring.RuntimeMetadataWiring$Greeting|parity.metadatawiring.RuntimeMetadataWiring$Service|true|true|hello|hello world"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := string(utf16.Decode(reference.UTF16Copy())); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}`)
}

func TestRuntimeMetadataLifecycleApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/runtime_metadata_lifecycle/src/RuntimeMetadataLifecycle.java")
	if err != nil {
		t.Fatal(err)
	}
	out := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, out, `package metadatalifecycle
import (
    "testing"
    "unicode/utf16"
)
func TestLifecycle(t *testing.T) {
    want := "/I/IC/true/base/child/9/12/2/true/true/true/plugin/target/missing/field/method/final"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := string(utf16.Decode(reference.UTF16Copy())); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}`)
}

func TestRuntimeMetadataArgumentsApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/runtime_metadata_arguments/src/RuntimeMetadataArguments.java")
	if err != nil {
		t.Fatal(err)
	}
	out := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, out, `package metadataarguments
import (
    "testing"
    "unicode/utf16"
)
func TestArguments(t *testing.T) {
    want := "plugin/plugin/arity/constructor"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := string(utf16.Decode(reference.UTF16Copy())); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}`)
}

func TestRuntimeMetadataAnnotationsApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/runtime_metadata_annotations/src/RuntimeMetadataAnnotations.java")
	if err != nil {
		t.Fatal(err)
	}
	out := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, out, `package metadataannotations
import (
    "testing"
    "unicode/utf16"
)
func TestAnnotations(t *testing.T) {
    want := "hello component/false/false/true/false/true"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := string(utf16.Decode(reference.UTF16Copy())); got != want { t.Fatalf("Run() = %q, want %q", got, want) }
}`)
}
