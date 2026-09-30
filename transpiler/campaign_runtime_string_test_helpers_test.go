package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

type campaignStringObservation struct {
	stdout, stderr []byte
	exit           int
}

func campaignStringBoundedRun(t *testing.T, stage, directory, program string, timeout time.Duration, args ...string) campaignStringObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, program, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	t.Logf("starting %s", stage)
	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("%s timed out: %v\n%s\n%s", stage, ctx.Err(), stdout.Bytes(), stderr.Bytes())
	}
	exit := 0
	if err != nil {
		if status, ok := err.(*exec.ExitError); ok {
			exit = status.ExitCode()
		} else {
			t.Fatalf("%s could not start: %v", stage, err)
		}
	}
	return campaignStringObservation{stdout.Bytes(), stderr.Bytes(), exit}
}

func campaignStringRequireSuccess(t *testing.T, stage string, observation campaignStringObservation) {
	t.Helper()
	if observation.exit != 0 {
		t.Fatalf("%s exit=%d\nstdout:\n%s\nstderr:\n%s", stage, observation.exit, observation.stdout, observation.stderr)
	}
}
