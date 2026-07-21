package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArrayCreation_QualifiesImportedGeneratedElementType(t *testing.T) {
	root := t.TempDir()
	modelDir := filepath.Join(root, "example", "model")
	kernelDir := filepath.Join(root, "example", "kernel")
	for _, directory := range []string{modelDir, kernelDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create source directory: %v", err)
		}
	}

	modelSource := `package example.model; public class Cohort {}`
	kernelSource := `
package example.kernel;
import example.model.Cohort;
public class Kernel {
    public static Cohort[] allocate(int count) {
        return new Cohort[count];
    }
}
`
	if err := os.WriteFile(filepath.Join(modelDir, "Cohort.java"), []byte(modelSource), 0o644); err != nil {
		t.Fatalf("write model source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(kernelDir, "Kernel.java"), []byte(kernelSource), 0o644); err != nil {
		t.Fatalf("write kernel source: %v", err)
	}

	outputs := convertJavaProjectDir(t, root)
	out := outputs["example/kernel/Kernel.go"]
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, `model "example/model"`) {
		t.Fatalf("expected imported model package, got:\n%s", out)
	}
	if !strings.Contains(flat, "make([]*model.Cohort, count)") {
		t.Fatalf("expected qualified imported array element type, got:\n%s", out)
	}
}
