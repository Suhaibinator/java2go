package campaign

import (
	"context"
	"testing"
	"time"
)

func TestGoProgramDisablesAmbientAssertions(t *testing.T) {
	t.Setenv("JAVA2GO_ASSERTIONS", "true")
	result := execute(context.Background(), time.Second, t.TempDir(), goProgramCommand("sh", []string{"-c", `printf '%s' "$JAVA2GO_ASSERTIONS"`}))
	if result.ExitCode != 0 || result.Stdout != "false" || result.Stderr != "" || result.TimedOut || result.Error != "" {
		t.Fatalf("assertion mode not pinned: %#v", result)
	}
}
