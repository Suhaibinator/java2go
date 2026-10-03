package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
)

type Execution struct {
	Command    []string          `json:"command,omitempty"`
	Directory  string            `json:"directory,omitempty"`
	Stdout     string            `json:"stdout"`
	Stderr     string            `json:"stderr"`
	ExitCode   int               `json:"exit_code"`
	TimedOut   bool              `json:"timed_out"`
	Error      string            `json:"error,omitempty"`
	DurationMS int64             `json:"duration_ms"`
	Files      map[string]string `json:"files,omitempty"`
}

func execute(ctx context.Context, timeout time.Duration, dir string, command []string) Execution {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var out, errout bytes.Buffer
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stdout = &out
	cmd.Stderr = &errout
	cmd.Env = append(os.Environ(), "TZ=UTC", "LC_ALL=C", "LANG=C", "GOWORK=off")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = time.Second
	start := time.Now()
	err := cmd.Run()
	r := Execution{Command: command, Directory: dir, Stdout: out.String(), Stderr: errout.String(), DurationMS: time.Since(start).Milliseconds()}
	if err != nil {
		r.ExitCode = -1
		if cmd.ProcessState != nil {
			r.ExitCode = cmd.ProcessState.ExitCode()
		}
		if _, ok := err.(*exec.ExitError); !ok {
			r.Error = err.Error()
		}
	}
	r.TimedOut = ctx.Err() != nil
	return r
}
func Compare(want, got Execution) string {
	if want.TimedOut || got.TimedOut {
		return "execution timed out"
	}
	if want.Error != "" || got.Error != "" {
		return fmt.Sprintf("execution error: Java=%q Go=%q", want.Error, got.Error)
	}
	if want.ExitCode != got.ExitCode {
		return fmt.Sprintf("exit code differs: Java=%d Go=%d", want.ExitCode, got.ExitCode)
	}
	if want.Stdout != got.Stdout {
		return "stdout bytes differ"
	}
	if want.Stderr != got.Stderr {
		return "stderr bytes differ"
	}
	if !reflect.DeepEqual(want.Files, got.Files) {
		return "output file names or bytes differ"
	}
	return ""
}
func writeJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return write(path, append(b, '\n'))
}
func write(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	return os.WriteFile(path, b, 0644)
}

// Raw files are authoritative even if stdout/stderr contain non-UTF-8 bytes.
func saveExecution(prefix string, r Execution) error {
	if e := write(prefix+".stdout", []byte(r.Stdout)); e != nil {
		return e
	}
	if e := write(prefix+".stderr", []byte(r.Stderr)); e != nil {
		return e
	}
	return writeJSON(prefix+".execution.json", r)
}
func classifyMavenFailure(r Execution) string {
	if r.TimedOut {
		return "maven-timeout"
	}
	combined := r.Stdout + "\n" + r.Stderr
	for _, marker := range []string{"could not be resolved", "Cannot access central", "has not been downloaded", "Could not resolve dependencies", "PluginResolutionException", "DependencyResolutionException"} {
		if strings.Contains(combined, marker) {
			return "maven-resolution"
		}
	}
	return "maven-build"
}
