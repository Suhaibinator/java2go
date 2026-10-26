package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Keep compilation and execution separate: every generated package must build
// in strict mode, and an execution timeout or stderr difference cannot be parity.
func runCampaignCompilerStrictProjectOracle(t *testing.T, files map[string]string, mainClass, expected string, parentCompiler ...*campaignCompilerParentCLI) {
	t.Helper()
	runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t, files, mainClass, []campaignCompilerProjectObservation{{name: "default", stdout: expected}}, time.Minute, parentCompiler...)
}

type campaignCompilerProjectObservation struct {
	name   string
	args   []string
	stdout string
	stderr string
}

func runCampaignCompilerStrictProjectObservations(t *testing.T, files map[string]string, mainClass string, observations []campaignCompilerProjectObservation) {
	runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t, files, mainClass, observations, time.Minute)
}

func runCampaignCompilerStrictProjectObservationsWithExecutionLimit(t *testing.T, files map[string]string, mainClass string, observations []campaignCompilerProjectObservation, executionLimit time.Duration, parentCompiler ...*campaignCompilerParentCLI) {
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
	if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
		t.Cleanup(func() {
			if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				name := filepath.Base(path)
				if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".java" && name != "go.mod" && name != "go.sum" {
					return nil
				}
				relative, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				contents, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				return writeProjectFile(filepath.Join(artifactDir, t.Name(), relative), contents)
			}); err != nil {
				t.Errorf("preserve compiler artifacts: %v", err)
			}
		})
	}
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
		want, wantErr := run("JVM "+observation.name, root, executionLimit, java, args...)
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
	if len(parentCompiler) == 0 {
		run("compiler build", repo, 5*time.Minute, "go", "build", "-o", compiler, "./cmd/java2go")
	} else {
		if len(parentCompiler) != 1 || parentCompiler[0] == nil {
			t.Fatal("strict project oracle requires one nonnil parent compiler owner")
		}
		goTool, err := exec.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		var reused bool
		compiler, reused, err = parentCompiler[0].executable(repo, goTool, append(os.Environ(), "GOWORK=off"), func(output string) error {
			run("compiler build", repo, 5*time.Minute, "go", "build", "-o", output, "./cmd/java2go")
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if reused {
			t.Logf("compiler reuse: %s (same parent, source, Go tool and build environment)", compiler)
		}
	}
	generated := filepath.Join(root, "generated")
	run("strict transpilation", repo, 5*time.Minute, compiler, "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	prepareCampaignReadonlyRuntimeModule(t, generated, repo)
	run("all generated packages race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "generated-app")
	run("generated entry point race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	for index, observation := range observations {
		got, gotErr := run("generated Go "+observation.name, root, executionLimit, binary, observation.args...)
		want := oracles[index]
		if !bytes.Equal(got, want.stdout) || !bytes.Equal(gotErr, want.stderr) {
			t.Fatalf("Go observation %s stdout=%q stderr=%q differs from JVM stdout=%q stderr=%q", observation.name, got, gotErr, want.stdout, want.stderr)
		}
	}
}

// This owner is created by one parent test, never a global or environment cache.
// Its directory must outlive all child tests and stay outside the source tree.
type campaignCompilerParentCLI struct {
	mu           sync.Mutex
	directory    string
	built        bool
	contextHash  [32]byte
	compilerHash [32]byte
}

func (owner *campaignCompilerParentCLI) executable(repo, goTool string, env []string, build func(string) error) (string, bool, error) {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	identity, err := campaignCompilerParentCLIContext(repo, goTool, env)
	if err != nil {
		return "", false, err
	}
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return "", false, err
	}
	canonicalDirectory, err := filepath.EvalSymlinks(owner.directory)
	if err != nil {
		return "", false, err
	}
	relative, err := filepath.Rel(canonicalRepo, canonicalDirectory)
	if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return "", false, fmt.Errorf("parent compiler directory must be outside source tree")
	}
	compiler := filepath.Join(owner.directory, "java2go")
	if owner.built {
		if identity != owner.contextHash {
			return "", false, fmt.Errorf("parent compiler source, Go tool or build environment changed")
		}
		data, err := os.ReadFile(compiler)
		if err != nil || sha256.Sum256(data) != owner.compilerHash {
			return "", false, fmt.Errorf("parent compiler binary changed or disappeared: %v", err)
		}
		return compiler, true, nil
	}
	// A failed prior attempt may have left output; only this build may supply it.
	if err := os.Remove(compiler); err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	if err := build(compiler); err != nil {
		return "", false, err
	}
	after, err := campaignCompilerParentCLIContext(repo, goTool, env)
	if err != nil {
		return "", false, err
	}
	if after != identity {
		return "", false, fmt.Errorf("parent compiler inputs changed during build")
	}
	data, err := os.ReadFile(compiler)
	if err != nil {
		return "", false, err
	}
	owner.contextHash, owner.compilerHash = identity, sha256.Sum256(data)
	owner.built = true
	return compiler, false, nil
}

func campaignCompilerParentCLIContext(repo, goTool string, env []string) ([32]byte, error) {
	var result [32]byte
	canonicalRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return result, err
	}
	canonicalTool, err := filepath.EvalSymlinks(goTool)
	if err != nil {
		return result, err
	}
	tool, err := os.ReadFile(canonicalTool)
	if err != nil {
		return result, err
	}
	digest := sha256.New()
	_, _ = fmt.Fprintf(digest, "repo=%q\ngo=%q:%x\nenv=%q\ncommand=%q\n", canonicalRepo, canonicalTool, sha256.Sum256(tool), env, []string{"go", "build", "-o", "<parent-temporary-compiler>", "./cmd/java2go"})
	err = filepath.WalkDir(canonicalRepo, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == filepath.Join(canonicalRepo, ".git") || path == filepath.Join(canonicalRepo, ".campaign") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(canonicalRepo, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(digest, "source=%q:%x\n", relative, sha256.Sum256(data))
		return nil
	})
	if err != nil {
		return result, err
	}
	copy(result[:], digest.Sum(nil))
	return result, nil
}

// Pure filesystem/build-result controls: these invoke no Go compiler or JVM.
func TestCampaignCompilerParentCLIReuseControls(t *testing.T) {
	setup := func(t *testing.T) (*campaignCompilerParentCLI, string, string, []string) {
		t.Helper()
		repo, tool := t.TempDir(), filepath.Join(t.TempDir(), "go")
		if err := os.WriteFile(filepath.Join(repo, "source.go"), []byte("package probe"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tool, []byte("pinned tool"), 0600); err != nil {
			t.Fatal(err)
		}
		return &campaignCompilerParentCLI{directory: t.TempDir()}, repo, tool, []string{"GOFLAGS=-p=1 -race -mod=readonly", "GOWORK=off"}
	}
	t.Run("successfulParentBuildSurvivesChildrenAndReuses", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		builds := 0
		build := func(path string) error {
			builds++
			return os.WriteFile(path, []byte("compiler bytes"), 0700)
		}
		for index := 0; index < 2; index++ {
			t.Run(fmt.Sprint(index), func(t *testing.T) {
				path, reused, err := owner.executable(repo, tool, env, build)
				if err != nil || reused != (index == 1) || path != filepath.Join(owner.directory, "java2go") {
					t.Fatalf("path=%q reused=%v err=%v", path, reused, err)
				}
			})
		}
		if builds != 1 {
			t.Fatalf("want one parent build, got %d", builds)
		}
	})
	t.Run("failedBuildCannotBecomeCachedSuccess", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		_, reused, err := owner.executable(repo, tool, env, func(string) error { return fmt.Errorf("real build failure") })
		if err == nil || reused || owner.built {
			t.Fatal("failed build accepted or cached")
		}
		_, reused, err = owner.executable(repo, tool, env, func(path string) error { return os.WriteFile(path, []byte("successful retry"), 0700) })
		if err != nil || reused || !owner.built {
			t.Fatalf("retry reused=%v err=%v built=%v", reused, err, owner.built)
		}
	})
	t.Run("changedSourceDuringBuildCannotBeCached", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		_, _, err := owner.executable(repo, tool, env, func(path string) error {
			if err := os.WriteFile(path, []byte("compiler bytes"), 0700); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(repo, "source.go"), []byte("changed during build"), 0600)
		})
		if err == nil || owner.built {
			t.Fatal("build with changed inputs accepted or cached")
		}
	})
	t.Run("missingBuildArtifactCannotBeCached", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		_, _, err := owner.executable(repo, tool, env, func(string) error { return nil })
		if err == nil || owner.built {
			t.Fatal("missing compiler accepted or cached")
		}
	})
	t.Run("sourceTreeCannotOwnCompilerLifetime", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		owner.directory = repo
		called := false
		_, _, err := owner.executable(repo, tool, env, func(string) error { called = true; return nil })
		if err == nil || called || owner.built {
			t.Fatal("compiler directory inside source accepted")
		}
	})
	t.Run("symlinkIntoSourceRefusesBeforeBuildOrSourceWrite", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		inside := filepath.Join(repo, "inside")
		if err := os.Mkdir(inside, 0700); err != nil {
			t.Fatal(err)
		}
		alias := filepath.Join(t.TempDir(), "source-alias")
		if err := os.Symlink(inside, alias); err != nil {
			t.Fatal(err)
		}
		owner.directory = alias
		called := false
		_, _, err := owner.executable(repo, tool, env, func(path string) error {
			called = true
			return os.WriteFile(path, []byte("compiler bytes"), 0700)
		})
		_, outputErr := os.Stat(filepath.Join(inside, "java2go"))
		if err == nil || called || owner.built || !os.IsNotExist(outputErr) {
			t.Fatalf("symlink source directory accepted: err=%v called=%v built=%v outputErr=%v", err, called, owner.built, outputErr)
		}
	})
	t.Run("failedBuildArtifactCannotSatisfyLaterMissingOutput", func(t *testing.T) {
		owner, repo, tool, env := setup(t)
		_, _, err := owner.executable(repo, tool, env, func(path string) error {
			if err := os.WriteFile(path, []byte("failed build leftover"), 0700); err != nil {
				return err
			}
			return fmt.Errorf("real build failure after output")
		})
		if err == nil || owner.built {
			t.Fatal("failed build with leftover accepted or cached")
		}
		_, reused, err := owner.executable(repo, tool, env, func(string) error { return nil })
		if err == nil || reused || owner.built {
			t.Fatalf("stale output accepted after no-artifact success: err=%v reused=%v built=%v", err, reused, owner.built)
		}
	})
	for _, change := range []string{"source", "environment", "tool", "repository", "binary", "missingBinary"} {
		t.Run("changed"+change+"RefusesReuse", func(t *testing.T) {
			owner, repo, tool, env := setup(t)
			path, _, err := owner.executable(repo, tool, env, func(path string) error { return os.WriteFile(path, []byte("compiler bytes"), 0700) })
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "source":
				err = os.WriteFile(filepath.Join(repo, "source.go"), []byte("changed source"), 0600)
			case "environment":
				env = append(env, "GOOS=linux")
			case "tool":
				err = os.WriteFile(tool, []byte("changed tool"), 0600)
			case "repository":
				repo = t.TempDir()
			case "binary":
				err = os.WriteFile(path, []byte("changed binary"), 0700)
			case "missingBinary":
				err = os.Remove(path)
			}
			if err != nil {
				t.Fatal(err)
			}
			buildCalled := false
			_, _, err = owner.executable(repo, tool, env, func(string) error { buildCalled = true; return nil })
			if err == nil || buildCalled {
				t.Fatalf("changed inputs accepted: err=%v buildCalled=%v", err, buildCalled)
			}
		})
	}
}
