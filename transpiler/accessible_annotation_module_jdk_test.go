package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAccessibleObjectFreshNamedModuleOracle(t *testing.T) {
	home := os.Getenv("JAVA_HOME")
	if home == "" {
		t.Fatal("JAVA_HOME is required for the JDK21 oracle")
	}
	root := filepath.Join("testdata", "reflection_accessible_module")
	output := t.TempDir()
	run := func(tool string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, filepath.Join(home, "bin", tool), args...)
		var stderr bytes.Buffer
		command.Stderr = &stderr
		stdout, err := command.Output()
		if err != nil {
			t.Fatalf("%s: %v stderr=%s", tool, err, stderr.String())
		}
		t.Logf("JDK21 %s stdout=%q stderr=%q", tool, stdout, stderr.String())
		return stdout
	}
	run("javac", "--release", "21", "-encoding", "UTF-8", "-d", output, filepath.Join(root, "module-info.java"), filepath.Join(root, "access", "fixture", "Target.java"), filepath.Join(root, "access", "observe", "AccessProbe.java"))
	actual := run("java", "--module-path", output, "--module", "access.witness/access.observe.AccessProbe")
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("fresh JDK output changed:\n%s", actual)
	}
}
