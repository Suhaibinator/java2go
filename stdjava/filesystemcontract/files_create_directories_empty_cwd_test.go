//go:build darwin || linux

package directoryoracle_test

import (
	"bytes"
	"context"
	"fmt"
	j "github.com/NickyBoy89/java2go/stdjava"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// directoryChildAPIObservations removes exactly one recognized Go testing
// success footer. The caller still checks exit/status/stderr and compares every
// remaining API byte to the immutable JDK oracle.
func directoryChildAPIObservations(output []byte) ([]byte, bool) {
	if api, ok := bytes.CutSuffix(output, []byte("PASS\ncoverage: [no statements]\n")); ok {
		return api, true
	}
	return bytes.CutSuffix(output, []byte("PASS\n"))
}

// A separate copy of this same test binary owns and removes only its child cwd.
// The parent never changes its cwd or permissions. The exact child operation
// observations are compared to the three-fresh-JDK stdout fixture.
func TestFilesCreateDirectoriesEmptyPathRemovedCwd(t *testing.T) {
	if os.Getenv("JAVA2GO_EMPTY_CWD_CHILD") != "" {
		directoryEmptyPathChild(t)
		return
	}
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/empty_path_removed_cwd_jdk21.stdout")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	if err := os.Mkdir(cwd, 0700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	argv := []string{"-test.run=^TestFilesCreateDirectoriesEmptyPathRemovedCwd$", "-test.count=1", "-test.timeout=8s"}
	command := exec.CommandContext(ctx, executable, argv...)
	command.Dir = cwd
	command.WaitDelay = time.Second
	command.Env = append(os.Environ(), "JAVA2GO_EMPTY_CWD_CHILD="+cwd)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	pid := command.Process.Pid
	runErr := command.Wait()
	t.Logf("child pid=%d argv=%q exit=%d stdout=%q stderr=%q", pid, argv, command.ProcessState.ExitCode(), stdout.String(), stderr.String())
	if runErr != nil || ctx.Err() != nil || stderr.Len() != 0 {
		t.Fatalf("child setup/framework/timeout failure: %v context=%v stderr=%q", runErr, ctx.Err(), stderr.String())
	}
	got, frameworkOK := directoryChildAPIObservations(stdout.Bytes())
	if !frameworkOK {
		t.Fatalf("child framework footer invalid: %q", stdout.String())
	}
	if !bytes.Equal(got, want) {
		t.Errorf("child API observations differ\nGOT:\n%sWANT:\n%s", got, want)
	}
	after, err := os.Getwd()
	if err != nil || before != after {
		t.Fatalf("parent cwd changed: %q ->%q err=%v", before, after, err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("owned cwd cleanup entries=%d err=%v", len(entries), err)
	}
}

func directoryEmptyPathChild(t *testing.T) {
	own := os.Getenv("JAVA2GO_EMPTY_CWD_CHILD")
	if own == "" {
		t.Fatal("missing isolated child marker")
	}
	actual, err := os.Getwd()
	if err != nil || actual != own || filepath.Base(own) != "cwd" {
		t.Fatalf("invalid owned child cwd actual=%q own=%q err=%v", actual, own, err)
	}
	exists := func(path string) bool { _, err := os.Lstat(path); return err == nil }
	fmt.Printf("BEFORE\tcwd-exists=%t\n", exists(own))
	if err := os.Remove(own); err != nil {
		t.Fatalf("remove own cwd setup: %v", err)
	}
	fmt.Printf("REMOVED\tcwd-exists=%t\n", exists(own))
	path := j.NewJavaPath("")
	var result *j.JavaPath
	var failure any
	func() { defer func() { failure = recover() }(); result = j.FilesCreateDirectories(path) }()
	if failure == nil {
		fmt.Printf("RETURN\tidentity=%t\tlexical-empty=%t\tdirectory=%t\n", result == path, result != nil && result.ToString() == "", j.FilesIsDirectory(result))
	} else {
		failure = j.NormalizePanic(failure)
		throwable, ok := failure.(j.Throwable)
		if !ok {
			t.Fatalf("non-Java panic: %T", failure)
		}
		fmt.Printf("THROW\t%s\n", j.BuiltinThrowableTypeID(throwable.ThrowableTypeName()))
		message := directoryMessageUnits(strings.ReplaceAll(j.GetMessage(failure), own, "<ROOT>"))
		fmt.Printf("MESSAGE_UTF16\t%s\n", message)
		cause := j.GetCause(failure)
		depth := 0
		for cause != nil {
			if depth >= 32 {
				t.Fatal("cause bound")
			}
			c, ok := cause.(j.Throwable)
			if !ok {
				t.Fatalf("non-Java cause %T", cause)
			}
			fmt.Printf("CAUSE%d\t%s\t%s\n", depth, j.BuiltinThrowableTypeID(c.ThrowableTypeName()), directoryMessageUnits(strings.ReplaceAll(j.GetMessage(cause), own, "<ROOT>")))
			cause = j.GetCause(cause)
			depth++
		}
		fmt.Printf("CAUSE%d\t<null>\n", depth)
	}
	fmt.Printf("AFTER\tcwd-exists=%t\tlocal-directory=%t\n", exists(own), j.FilesIsDirectory(j.NewJavaPath(".")))
}
