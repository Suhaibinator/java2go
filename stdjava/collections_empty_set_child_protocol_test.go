package stdjava

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Exercise the actual parent and its observation child through the same test
// executable, including coverage instrumentation when the caller has it enabled.
func TestCollectionsEmptySetDiagnosticIdentityParentProtocol(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestCollectionsEmptySetNativeDiagnosticIdentityJDK21$", "-test.v", "-test.timeout=5s")
	// The observation flag belongs only to the grandchild spawned by the parent.
	command.Env = append(os.Environ(), "JAVA2GO_EMPTY_SET_IDENTITY_CHILD=0")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("identity parent protocol: %v\n%s", err, output)
	}
	const passed = "--- PASS: TestCollectionsEmptySetNativeDiagnosticIdentityJDK21 ("
	if !strings.Contains(string(output), passed) {
		t.Fatalf("identity parent did not execute successfully:\n%s", output)
	}
}
