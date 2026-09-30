package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/campaign"
)

// Re-enter the real CLI without a nested Go build or a Java/Maven dependency.
func TestCampaignSignalHelper(t *testing.T) {
	if os.Getenv("JAVA2GO_SIGNAL_HELPER") != "1" {
		return
	}
	flag.CommandLine = flag.NewFlagSet("javacampaign", flag.ExitOnError)
	os.Args = []string{"javacampaign", "-repository", os.Getenv("JAVA2GO_SIGNAL_REPOSITORY"), "-fixture", os.Getenv("JAVA2GO_SIGNAL_FIXTURE"), "-artifacts", os.Getenv("JAVA2GO_SIGNAL_ARTIFACTS"), "-jdk", os.Getenv("JAVA2GO_SIGNAL_JDK")}
	main()
	os.Exit(0)
}

func TestCampaignSignalsCancelStageTreeAndPreserveReport(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			dir := t.TempDir()
			jdk := filepath.Join(dir, "jdk")
			bin := filepath.Join(jdk, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				t.Fatal(err)
			}
			// Both shell and sleep belong to execute's distinct process group.
			// The PID file is written only after the grandchild has started.
			script := "#!/bin/sh\nsleep 120 &\nprintf '%s %s\\n' \"$$\" \"$!\" > pids\nwait\n"
			for name, contents := range map[string]string{"java": script, "javac": "#!/bin/sh\nexit 0\n"} {
				if err := os.WriteFile(filepath.Join(bin, name), []byte(contents), 0755); err != nil {
					t.Fatal(err)
				}
			}
			repository, err := filepath.Abs("../..")
			if err != nil {
				t.Fatal(err)
			}
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(executable, "-test.run=^TestCampaignSignalHelper$")
			cmd.Env = append(os.Environ(), "JAVA2GO_SIGNAL_HELPER=1", "JAVA2GO_SIGNAL_REPOSITORY="+repository, "JAVA2GO_SIGNAL_FIXTURE="+dir, "JAVA2GO_SIGNAL_ARTIFACTS="+filepath.Join(dir, "runs"), "JAVA2GO_SIGNAL_JDK="+jdk)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			var stagePID int
			finished := false
			t.Cleanup(func() {
				if stagePID != 0 {
					if err := syscall.Kill(-stagePID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
						t.Errorf("stage cleanup: %v", err)
					}
				}
				if !finished {
					if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
						t.Errorf("CLI cleanup: %v", err)
					}
					<-done
				}
			})
			var pids []int
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				data, err := os.ReadFile(filepath.Join(jdk, "pids"))
				fields := strings.Fields(string(data))
				if err == nil && len(fields) == 2 {
					for _, field := range fields {
						pid, err := strconv.Atoi(field)
						if err != nil || pid <= 0 {
							t.Fatalf("invalid stage PID %q", field)
						}
						pids = append(pids, pid)
					}
					stagePID = pids[0]
					break
				}
				select {
				case err := <-done:
					finished = true
					t.Fatalf("CLI exited before stage started: %v; %s", err, stderr.String())
				default:
				}
				time.Sleep(10 * time.Millisecond)
			}
			if len(pids) != 2 {
				t.Fatal("stage did not publish child/grandchild PIDs")
			}
			if group, err := syscall.Getpgid(stagePID); err != nil || group != stagePID || group == cmd.Process.Pid {
				t.Fatalf("stage must have its own group: group=%d err=%v", group, err)
			}
			for _, pid := range pids {
				if !signalProcessAlive(t, pid) {
					t.Fatalf("stage descendant %d was not alive before cancellation", pid)
				}
			}
			// Signal the CLI group as an external runner does, not the stage.
			if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				finished = true
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 {
					t.Fatalf("CLI must exit 1 after preserving failure: %v; %s", err, stderr.String())
				}
			case <-time.After(3 * time.Second):
				t.Fatal("CLI did not cancel before the 10-second JDK stage timeout")
			}
			for _, pid := range pids {
				deadline := time.Now().Add(time.Second)
				for signalProcessAlive(t, pid) && time.Now().Before(deadline) {
					time.Sleep(10 * time.Millisecond)
				}
				if signalProcessAlive(t, pid) {
					t.Errorf("stage descendant %d survived CLI cancellation", pid)
				}
			}
			var report campaign.Report
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
				t.Fatalf("missing JSON failure report: %v; stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			if report.Passed || report.Failure == nil || report.Failure.Stage != "java-toolchain" {
				t.Fatalf("unexpected cancellation report: %+v", report)
			}
			data, err := os.ReadFile(filepath.Join(report.ArtifactDirectory, "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(bytes.TrimSpace(data), bytes.TrimSpace(stdout.Bytes())) {
				t.Fatal("saved failure report differs from CLI JSON")
			}
		})
	}
}

func signalProcessAlive(t *testing.T, pid int) bool {
	t.Helper()
	output, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return false
		}
		t.Fatalf("inspect descendant: %v", err)
	}
	state := strings.TrimSpace(string(output))
	return state != "" && !strings.HasPrefix(state, "Z")
}
