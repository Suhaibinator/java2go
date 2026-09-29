package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Keep compilation and execution separate: every generated package must build
// in strict mode, and an execution timeout or stderr difference cannot be parity.
func runCampaignCompilerStrictProjectOracle(t *testing.T, files map[string]string, mainClass, expected string) {
	t.Helper()
	runCampaignCompilerStrictProjectObservations(t, files, mainClass, []campaignCompilerProjectObservation{{name: "default", stdout: expected}})
}

type campaignCompilerProjectObservation struct {
	name   string
	args   []string
	stdout string
	stderr string
}

func runCampaignCompilerStrictProjectObservations(t *testing.T, files map[string]string, mainClass string, observations []campaignCompilerProjectObservation) {
	t.Helper()
	if len(observations) == 0 {
		t.Fatal("strict project oracle requires at least one observation")
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	run := func(stage, dir string, limit time.Duration, name string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		started := time.Now()
		err := cmd.Run()
		t.Logf("%s: %v (%s)", stage, cmd.Args, time.Since(started))
		if ctx.Err() != nil {
			t.Fatalf("%s timed out: %v\nstdout: %s\nstderr: %s", stage, ctx.Err(), stdout.Bytes(), stderr.Bytes())
		}
		if err != nil {
			t.Fatalf("%s: %v\nstdout: %s\nstderr: %s", stage, err, stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	root := t.TempDir()
	var sources []string
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(name) == ".java" {
			sources = append(sources, path)
		}
	}
	sort.Strings(sources)
	classes := filepath.Join(root, "classes")
	run("javac", root, 5*time.Minute, javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", classes}, sources...)...)
	type oracleOutput struct{ stdout, stderr []byte }
	oracles := make([]oracleOutput, len(observations))
	for index, observation := range observations {
		args := append([]string{"-cp", classes, mainClass}, observation.args...)
		want, wantErr := run("JVM "+observation.name, root, time.Minute, java, args...)
		if string(want) != observation.stdout || string(wantErr) != observation.stderr {
			t.Fatalf("invalid JVM oracle %s: stdout=%q stderr=%q; expected stdout=%q stderr=%q", observation.name, want, wantErr, observation.stdout, observation.stderr)
		}
		t.Logf("JVM oracle %s: stdout=%q stderr=%q", observation.name, want, wantErr)
		oracles[index] = oracleOutput{want, wantErr}
	}
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler build", repo, 5*time.Minute, "go", "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict transpilation", repo, 5*time.Minute, compiler, "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	run("all generated packages race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "generated-app")
	run("generated entry point race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	for index, observation := range observations {
		got, gotErr := run("generated Go "+observation.name, root, time.Minute, binary, observation.args...)
		want := oracles[index]
		if !bytes.Equal(got, want.stdout) || !bytes.Equal(gotErr, want.stderr) {
			t.Fatalf("Go observation %s stdout=%q stderr=%q differs from JVM stdout=%q stderr=%q", observation.name, got, gotErr, want.stdout, want.stderr)
		}
	}
}
