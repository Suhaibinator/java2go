package campaign

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise Run's actual stage/retention boundary with tiny explicit tool stubs.
// No real Git, JDK, Maven, compiler, application, or dependency download runs.
// A directory at an artifact filename fails reliably even under a privileged UID.
func TestRunRejectsUnretainedStageArtifacts(t *testing.T) {
	for _, target := range []string{"json", "stdout", "stderr", "writable_control"} {
		t.Run(target, func(t *testing.T) {
			root := t.TempDir()
			repo, fixture := filepath.Join(root, "repo"), filepath.Join(root, "fixture")
			tools, jdk := filepath.Join(root, "tools"), filepath.Join(root, "jdk")
			marker := filepath.Join(root, "next-stage-ran")
			processLedger := filepath.Join(root, "controlled-tool-processes")
			install := func(path, data string, mode os.FileMode) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), mode); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"astutil", "nodeutil", "parsing", "project", "symbol", "stdjava", "transpiler", "cmd/java2go", "campaign", "cmd/javacampaign"} {
				if err := os.MkdirAll(filepath.Join(repo, name), 0755); err != nil {
					t.Fatal(err)
				}
			}
			install(filepath.Join(repo, "api.go"), "package fixture\n", 0644)
			install(filepath.Join(repo, "go.mod"), "module fixture\n\ngo 1.27.0\n", 0644)
			install(filepath.Join(repo, "go.sum"), "", 0644)
			install(filepath.Join(repo, "campaign/dependencies.lock.json"), `{"schema_version":1,"artifacts":[]}`+"\n", 0644)
			install(filepath.Join(repo, "campaign/maven.lock.json"), `{"schema_version":1,"artifacts":[]}`+"\n", 0644)
			install(filepath.Join(tools, "git"), "#!/bin/sh\nprintf 'git %s\\n' \"$$\" >> \"$CAMPAIGN_ARTIFACT_PROCESS_LEDGER\"\n[ \"$1\" = rev-parse ] && [ \"$2\" = HEAD ] || exit 91\nprintf '%s\\n' 0000000000000000000000000000000000000000\n", 0755)
			install(filepath.Join(jdk, "bin/java"), "#!/bin/sh\nprintf 'java %s\\n' \"$$\" >> \"$CAMPAIGN_ARTIFACT_PROCESS_LEDGER\"\nprintf '%s\\n' 'java.specification.version = 21' >&2\n", 0755)
			install(filepath.Join(jdk, "bin/javac"), "#!/bin/sh\nprintf 'javac %s\\n' \"$$\" >> \"$CAMPAIGN_ARTIFACT_PROCESS_LEDGER\"\nprintf '%s\\n' reached > \"$CAMPAIGN_ARTIFACT_NEXT_STAGE\"\nexit 19\n", 0755)
			install(filepath.Join(repo, ".campaign/tools/apache-maven-3.9.16/bin/mvn"), `#!/bin/sh
printf 'maven %s\n' "$$" >> "$CAMPAIGN_ARTIFACT_PROCESS_LEDGER"
case "$CAMPAIGN_ARTIFACT_CONFLICT" in
  json|stdout|stderr) mkdir -p "../../stages/maven-build.$CAMPAIGN_ARTIFACT_CONFLICT" || exit 92 ;;
  writable_control) ;;
  *) exit 93 ;;
esac
printf 'maven-sentinel\n'
`, 0755)
			t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("CAMPAIGN_ARTIFACT_CONFLICT", target)
			t.Setenv("CAMPAIGN_ARTIFACT_NEXT_STAGE", marker)
			t.Setenv("CAMPAIGN_ARTIFACT_PROCESS_LEDGER", processLedger)
			install(filepath.Join(fixture, "src/app/Main.java"), "package app; public class Main {}\n", 0644)
			install(filepath.Join(fixture, "pom.xml"), "<project><modelVersion>4.0.0</modelVersion><groupId>fixture</groupId><artifactId>artifact</artifactId><version>1</version></project>\n", 0644)
			manifest := Manifest{Name: "artifact-retention", MainClass: "app.Main", SourceRoots: []string{"src"}, POM: "pom.xml", Seeds: []int{17, 41, 97}, Repeats: 3}
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			install(filepath.Join(fixture, "fixture.json"), string(data), 0644)
			for _, seed := range []string{"17", "41", "97"} {
				install(filepath.Join(fixture, "expected.seed-"+seed+".stdout"), "", 0644)
			}
			report, runErr := Run(context.Background(), Config{Repository: repo, Fixture: fixture, Artifacts: filepath.Join(root, "artifacts"), JDK: jdk, BuildTimeout: 5 * time.Second, RunTimeout: time.Second})
			instances, ledgerErr := os.ReadFile(processLedger)
			if ledgerErr != nil {
				t.Fatal(ledgerErr)
			}
			for _, instance := range strings.Split(strings.TrimSpace(string(instances)), "\n") {
				t.Logf("controlled fixture tool: %s", instance)
			}
			stage, exists := report.Stages["maven-build"]
			if !exists || stage.ExitCode != 0 || stage.Stdout != "maven-sentinel\n" || stage.Stderr != "" || stage.TimedOut || stage.Error != "" {
				t.Fatalf("fixture did not reach a successful controlled stage: %+v", report)
			}
			if runErr == nil || report.Passed || report.Failure == nil {
				t.Fatal("controlled failing fixture unexpectedly passed")
			}
			_, nextErr := os.Stat(marker)
			if target == "writable_control" {
				if nextErr != nil || report.Failure.Stage != "dependency-source-compile" {
					t.Fatalf("writable artifacts did not preserve ordinary continuation: %+v marker=%v", report.Failure, nextErr)
				}
				for _, suffix := range []string{"json", "stdout", "stderr"} {
					if info, err := os.Stat(filepath.Join(report.ArtifactDirectory, "stages/maven-build."+suffix)); err != nil || !info.Mode().IsRegular() {
						t.Fatalf("missing writable stage artifact %s: %v", suffix, err)
					}
				}
				return
			}
			conflict := filepath.Join(report.ArtifactDirectory, "stages/maven-build."+target)
			if info, err := os.Stat(conflict); err != nil || !info.IsDir() {
				t.Fatalf("controlled artifact collision missing: %s: %v", conflict, err)
			}
			if !os.IsNotExist(nextErr) || !strings.Contains(runErr.Error(), "stage artifact write failed") || !strings.Contains(report.Failure.Detail, conflict) {
				t.Fatal("successful command escaped artifact retention failure")
			}
			saved, err := os.ReadFile(filepath.Join(report.ArtifactDirectory, "report.json"))
			if err != nil {
				t.Fatal(err)
			}
			var durable Report
			if err := json.Unmarshal(saved, &durable); err != nil {
				t.Fatal(err)
			}
			if durable.Passed || durable.Failure == nil || !strings.Contains(durable.Failure.Detail, conflict) {
				t.Fatal("artifact failure missing from durable report")
			}
		})
	}
}
