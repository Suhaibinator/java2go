package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// This supplemental dependency contract uses the unchanged application and
// actual Gson source. Expectations are frozen JDK21 raw captures, never Java
// program text or rewritten generated Go. The source-only Maven path uses the
// current transpiler and runtime and does not need dependency binary JARs.
func TestCampaignDateISOStatefulStrictMavenOriginalPairs(t *testing.T) {
	fixture, err := filepath.Abs(filepath.Join("testdata", "date_iso_stateful"))
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	hash := func(raw []byte) string { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
	var manifest map[string]string
	manifestBytes := read("source-manifest.json")
	if hash(manifestBytes) != "e92f5161c0e6817664310b5795dcbce1e63a7d511f262662c10fe34fb35ad7ce" {
		t.Fatal("frozen source/oracle manifest pin changed")
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	verify := func() {
		t.Helper()
		actual := map[string]string{}
		if err := filepath.WalkDir(fixture, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(fixture, path)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(relative)
			if name != "source-manifest.json" {
				actual[name] = hash(read(name))
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, manifest) {
			t.Fatal("immutable Date/ISO fixture inventory changed")
		}
	}
	verify()
	t.Cleanup(verify)
	var metadata struct {
		MainClass    string `json:"main_class"`
		Project      string `json:"project"`
		Module       string `json:"module"`
		Seeds        []int  `json:"seeds"`
		Repeats      int    `json:"repeats"`
		Dependencies []struct {
			Coordinate string `json:"coordinate"`
			Version    string `json:"version"`
			Project    string `json:"project"`
			Member     string `json:"source_member"`
			SourceSHA  string `json:"source_sha256"`
		} `json:"dependency_sources"`
		Observations []struct {
			Seed      int      `json:"seed"`
			Repeat    int      `json:"repeat"`
			Args      []string `json:"args"`
			Stdout    string   `json:"stdout"`
			Stderr    string   `json:"stderr"`
			StdoutSHA string   `json:"stdout_sha256"`
			StderrSHA string   `json:"stderr_sha256"`
			Rows      int      `json:"rows"`
			ExitCode  int      `json:"exit_code"`
		} `json:"observations"`
	}
	if err := json.Unmarshal(read("build-metadata.json"), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.MainClass != "app.Main" || !reflect.DeepEqual(metadata.Seeds, []int{17, 41, 97}) || metadata.Repeats != 3 || len(metadata.Observations) != 9 || len(metadata.Dependencies) != 1 {
		t.Fatal("frozen Date/ISO scope or observation inventory changed")
	}
	dependency := metadata.Dependencies[0]
	if dependency.Coordinate != "com.google.code.gson:gson" || dependency.Version != "2.14.0" || dependency.SourceSHA != "ae515fc0a17f15c919a152a3004d874a81d1466d39bdf163bd45d468a3faeba5" {
		t.Fatal("upstream dependency coordinate or source pin changed")
	}
	selected := filepath.ToSlash(filepath.Join(dependency.Project, "src/main/java", dependency.Member))
	original := filepath.ToSlash(filepath.Join(metadata.Project, "dependency-sources/gson", dependency.Member))
	if hash(read(selected)) != dependency.SourceSHA || !bytes.Equal(read(selected), read(original)) {
		t.Fatal("selected upstream ISO source changed")
	}
	var captured struct {
		Status  string                                     `json:"status"`
		Repeats int                                        `json:"repeats_per_seed"`
		Seeds   map[string]struct{ Stdout, Stderr string } `json:"seeds"`
	}
	if err := json.Unmarshal(read("captured-oracles.json"), &captured); err != nil {
		t.Fatal(err)
	}
	if captured.Status != "STABLE_JDK21_ORACLES" || captured.Repeats != 3 || len(captured.Seeds) != 3 {
		t.Fatal("actual JDK capture metadata changed")
	}
	for index, observation := range metadata.Observations {
		if observation.Seed != metadata.Seeds[index/3] || observation.Repeat != index%3+1 || !reflect.DeepEqual(observation.Args, []string{strconv.Itoa(observation.Seed)}) || observation.Rows != 19 || observation.ExitCode != 0 {
			t.Fatal("original seed/repeat arguments changed")
		}
		stdout, stderr := read(observation.Stdout), read(observation.Stderr)
		seed := captured.Seeds[strconv.Itoa(observation.Seed)]
		if hash(stdout) != observation.StdoutSHA || hash(stderr) != observation.StderrSHA || !bytes.Equal(stdout, []byte(seed.Stdout)) || !bytes.Equal(stderr, []byte(seed.Stderr)) || bytes.Count(stdout, []byte{'\n'}) != 19 {
			t.Fatal("raw expected streams differ from stable JDK21 captures")
		}
	}
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	generated := filepath.Join(work, "generated")
	args := []string{"-strict", "-maven", filepath.Join(fixture, metadata.Project), "-dependency-source", dependency.Coordinate + "=" + filepath.Join(fixture, dependency.Project), "-main-class", metadata.MainClass, "-runtime", repo, "-module", metadata.Module, "-output", generated}
	var translation bytes.Buffer
	if err := run(args, &translation); err != nil {
		t.Fatalf("current strict source-only Maven translation: %v\n%s", err, translation.Bytes())
	}
	generatedInventory := func() map[string]string {
		t.Helper()
		values := map[string]string{}
		if err := filepath.WalkDir(generated, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(generated, path)
			if err != nil {
				return err
			}
			values[filepath.ToSlash(relative)] = hash(raw)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return values
	}
	generatedBefore := generatedInventory()
	t.Cleanup(func() {
		after := generatedInventory()
		if artifacts := os.Getenv("JAVA2GO_DATE_ISO_ARTIFACTS"); artifacts != "" {
			raw, err := json.MarshalIndent(map[string]any{"before": generatedBefore, "after": after}, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := writeProjectFile(filepath.Join(artifacts, "generated-source-snapshots.json"), append(raw, '\n')); err != nil {
				t.Fatal(err)
			}
		}
		if !reflect.DeepEqual(generatedBefore, after) {
			t.Error("generated Go source was modified after strict translation")
		}
	})
	goTool := os.Getenv("JAVA2GO_DATE_ISO_GO")
	if goTool == "" {
		goTool = "go"
	}
	execute := func(current *testing.T, stage, directory string, bound time.Duration, program string, argv ...string) ([]byte, []byte) {
		current.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		cmd := exec.CommandContext(ctx, program, argv...)
		cmd.Dir = directory
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "TZ=UTC")
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		started := time.Now()
		runErr := cmd.Run()
		exit := 0
		if runErr != nil {
			exit = -1
			if failure, ok := runErr.(*exec.ExitError); ok {
				exit = failure.ExitCode()
			}
		}
		if artifacts := os.Getenv("JAVA2GO_DATE_ISO_ARTIFACTS"); artifacts != "" {
			record, err := json.MarshalIndent(map[string]any{"stage": stage, "command": cmd.Args, "cwd": directory, "bound_seconds": bound.Seconds(), "exit_code": exit, "timed_out": ctx.Err() != nil, "elapsed_seconds": time.Since(started).Seconds(), "stdout_sha256": hash(stdout.Bytes()), "stderr_sha256": hash(stderr.Bytes())}, "", "  ")
			if err != nil {
				current.Fatal(err)
			}
			for name, raw := range map[string][]byte{stage + ".stdout": stdout.Bytes(), stage + ".stderr": stderr.Bytes(), stage + ".json": append(record, '\n')} {
				if err := writeProjectFile(filepath.Join(artifacts, name), raw); err != nil {
					current.Fatal(err)
				}
			}
		}
		if ctx.Err() != nil || runErr != nil {
			current.Fatalf("%s: %v %v\nstdout: %s\nstderr: %s", stage, runErr, ctx.Err(), stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	execute(t, "all-generated-race-build", generated, 5*time.Minute, goTool, "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(work, "generated-app")
	execute(t, "generated-entry-race-build", generated, 5*time.Minute, goTool, "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	for _, observation := range metadata.Observations {
		want, wantErr := read(observation.Stdout), read(observation.Stderr)
		t.Run(fmt.Sprintf("seed%d_repeat%d", observation.Seed, observation.Repeat), func(t *testing.T) {
			stage := fmt.Sprintf("Go-%d-repeat%d", observation.Seed, observation.Repeat)
			got, gotErr := execute(t, stage, work, time.Minute, binary, observation.Args...)
			if !bytes.Equal(got, want) || !bytes.Equal(gotErr, wantErr) {
				t.Fatalf("original JDK21 pair mismatch: stdout=%q stderr=%q; expected stdout=%q stderr=%q", got, gotErr, want, wantErr)
			}
		})
	}
}
