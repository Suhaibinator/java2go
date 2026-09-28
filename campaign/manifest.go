// Package campaign runs immutable, dependency-aware Java/Go differential fixtures.
// Unsupported constructs, dependency gaps and timeouts are failures, never skips.
package campaign

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Resource struct {
	Source string `json:"source"`
	Target string `json:"target"`
}
type Manifest struct {
	Name              string              `json:"name"`
	MainClass         string              `json:"main_class"`
	SourceRoots       []string            `json:"source_roots"`
	POM               string              `json:"pom"`
	Dependencies      []string            `json:"dependencies"`
	DependencySources map[string][]string `json:"dependency_sources"`
	Resources         []Resource          `json:"resources"`
	Args              []string            `json:"args"`
	Seeds             []int               `json:"seeds"`
	Repeats           int                 `json:"repeats"`
	OutputFiles       []string            `json:"output_files"`
}

func safeRelative(path string) bool {
	return path != "" && !filepath.IsAbs(path) && filepath.Clean(path) != ".." && !strings.HasPrefix(filepath.Clean(path), ".."+string(filepath.Separator))
}
func (m Manifest) Validate() error {
	if m.Name == "" || strings.ContainsAny(m.Name, "/\\") || m.MainClass == "" || len(m.SourceRoots) == 0 || !safeRelative(m.POM) {
		return fmt.Errorf("name, qualified main_class, source_roots and relative pom are required")
	}
	if !reflect.DeepEqual(m.Seeds, []int{17, 41, 97}) || m.Repeats != 3 {
		return fmt.Errorf("campaign requires seeds [17,41,97], each repeated 3 times")
	}
	for _, p := range append(append([]string{}, m.SourceRoots...), m.OutputFiles...) {
		if !safeRelative(p) {
			return fmt.Errorf("unsafe relative path %q", p)
		}
	}
	seen := map[string]bool{}
	for _, dep := range m.Dependencies {
		if seen[dep] {
			return fmt.Errorf("duplicate dependency %s", dep)
		}
		seen[dep] = true
		if len(m.DependencySources[dep]) == 0 {
			return fmt.Errorf("dependency %s requires complete implementation source closure", dep)
		}
	}
	for dep, paths := range m.DependencySources {
		if !seen[dep] {
			return fmt.Errorf("undeclared dependency source %s", dep)
		}
		for _, p := range paths {
			if p != "**" && (!safeRelative(p) || !strings.HasSuffix(p, ".java")) {
				return fmt.Errorf("invalid implementation source %q", p)
			}
		}
	}
	for _, r := range m.Resources {
		if !safeRelative(r.Source) || !safeRelative(r.Target) {
			return fmt.Errorf("unsafe resource path")
		}
	}
	return nil
}
func decode(path string, into any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(into); e != nil {
		return e
	}
	if e = d.Decode(new(any)); e != io.EOF {
		return fmt.Errorf("trailing JSON in %s", path)
	}
	return nil
}
func LoadManifest(root string) (Manifest, error) {
	var m Manifest
	e := decode(filepath.Join(root, "fixture.json"), &m)
	if e == nil {
		e = m.Validate()
	}
	return m, e
}

type Artifact struct {
	ID      string `json:"id"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	File    string `json:"file"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}
type Lock struct {
	SchemaVersion int        `json:"schema_version"`
	Artifacts     []Artifact `json:"artifacts"`
}

func LoadLock(root string) (Lock, error) {
	var l Lock
	e := decode(filepath.Join(root, "campaign/dependencies.lock.json"), &l)
	if e == nil {
		e = l.Verify(root)
	}
	return l, e
}
func hash(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func (l Lock) Verify(root string) error {
	if l.SchemaVersion != 1 {
		return fmt.Errorf("unsupported dependency lock schema")
	}
	for _, a := range l.Artifacts {
		if !safeRelative(a.File) {
			return fmt.Errorf("unsafe artifact path")
		}
		b, e := os.ReadFile(filepath.Join(root, ".campaign/cache", a.File))
		if e != nil {
			return e
		}
		if hash(b) != a.SHA256 {
			return fmt.Errorf("artifact checksum mismatch: %s", a.File)
		}
	}
	return nil
}
func (l Lock) Artifact(id, kind string) (Artifact, error) {
	for _, a := range l.Artifacts {
		if a.ID == id && a.Kind == kind {
			return a, nil
		}
	}
	return Artifact{}, fmt.Errorf("missing locked %s artifact for %s", kind, id)
}
