package campaign

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Implementation struct {
	Revision string            `json:"revision"`
	SHA256   string            `json:"sha256"`
	Files    map[string]string `json:"files"`
}

func implementationFingerprint(root string) (Implementation, error) {
	result := Implementation{Files: map[string]string{}}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	b, e := cmd.Output()
	if e != nil {
		return result, e
	}
	result.Revision = strings.TrimSpace(string(b))
	for _, name := range []string{"api.go", "go.mod", "go.sum", "astutil", "nodeutil", "parsing", "project", "symbol", "stdjava", "transpiler", "cmd/java2go", "campaign", "cmd/javacampaign"} {
		path := filepath.Join(root, name)
		e = filepath.WalkDir(path, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if !strings.HasSuffix(path, ".go") && filepath.Base(path) != "go.mod" && filepath.Base(path) != "go.sum" && !strings.HasSuffix(path, ".lock.json") {
				return nil
			}
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			rel, e := filepath.Rel(root, path)
			if e != nil {
				return e
			}
			result.Files[filepath.ToSlash(rel)] = hash(b)
			return nil
		})
		if e != nil {
			return result, e
		}
	}
	var names []string
	for name := range result.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	var builder strings.Builder
	builder.WriteString(result.Revision + "\n")
	for _, name := range names {
		builder.WriteString(name + "\x00" + result.Files[name] + "\n")
	}
	result.SHA256 = hash([]byte(builder.String()))
	return result, nil
}
func resolveJDK(ctx context.Context, explicit string) (string, Execution, error) {
	var candidates []string
	if explicit != "" {
		candidates = []string{explicit}
	} else if home := os.Getenv("JAVA_HOME"); home != "" {
		candidates = []string{home}
	} else {
		if java, e := exec.LookPath("java"); e == nil {
			if actual, e := filepath.EvalSymlinks(java); e == nil {
				candidates = append(candidates, filepath.Dir(filepath.Dir(actual)))
			}
		}
		candidates = append(candidates, "/Library/Java/JavaVirtualMachines/jdk-21.jdk/Contents/Home", "/usr/lib/jvm/java-21-openjdk-amd64", "/usr/lib/jvm/java-21-openjdk-arm64")
	}
	var last Execution
	for _, home := range candidates {
		home, e := filepath.Abs(home)
		if e != nil {
			continue
		}
		if _, e = os.Stat(filepath.Join(home, "bin/javac")); e != nil {
			continue
		}
		last = execute(ctx, 10*time.Second, home, []string{filepath.Join(home, "bin/java"), "-XshowSettings:properties", "-version"})
		if last.ExitCode == 0 && !last.TimedOut && strings.Contains(last.Stderr, "java.specification.version = 21") {
			return home, last, nil
		}
	}
	return "", last, fmt.Errorf("a working JDK 21 is required; set -jdk or JAVA_HOME (searched %v)", candidates)
}
func verifyMavenLock(root string) (string, error) {
	var lock struct {
		SchemaVersion int `json:"schema_version"`
		Artifacts     []struct {
			File   string `json:"file"`
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
		} `json:"artifacts"`
	}
	path := filepath.Join(root, "campaign/maven.lock.json")
	if e := decode(path, &lock); e != nil {
		return "", e
	}
	if lock.SchemaVersion != 1 {
		return "", fmt.Errorf("unsupported Maven lock schema")
	}
	for _, a := range lock.Artifacts {
		if !safeRelative(a.File) {
			return "", fmt.Errorf("unsafe Maven lock path")
		}
		b, e := os.ReadFile(filepath.Join(root, ".campaign/m2", a.File))
		if e != nil {
			return "", e
		}
		if hash(b) != a.SHA256 {
			return "", fmt.Errorf("maven artifact checksum mismatch: %s", a.File)
		}
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return "", e
	}
	return hash(b), nil
}
func copyMavenProject(fixture, dest string) error {
	return filepath.WalkDir(fixture, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if d.Name() == "target" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture symlink %s", path)
		}
		rel, e := filepath.Rel(fixture, path)
		if e != nil {
			return e
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		return write(filepath.Join(dest, rel), b)
	})
}
