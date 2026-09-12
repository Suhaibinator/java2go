// Package project discovers a deliberately bounded, offline Maven source graph.
// It does not execute build plugins or treat binary JAR signatures as source.
package project

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Dependency struct {
	GroupID    string       `xml:"groupId"`
	ArtifactID string       `xml:"artifactId"`
	Version    string       `xml:"version"`
	Scope      string       `xml:"scope"`
	Type       string       `xml:"type"`
	Classifier string       `xml:"classifier"`
	Exclusions []Dependency `xml:"exclusions>exclusion"`
}

func (d Dependency) key() string { return d.GroupID + ":" + d.ArtifactID }

type property struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}
type resource struct {
	Directory  string   `xml:"directory"`
	TargetPath string   `xml:"targetPath"`
	Filtering  bool     `xml:"filtering"`
	Includes   []string `xml:"includes>include"`
	Excludes   []string `xml:"excludes>exclude"`
}
type pom struct {
	XMLName    xml.Name `xml:"project"`
	GroupID    string   `xml:"groupId"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Packaging  string   `xml:"packaging"`
	Parent     *struct {
		GroupID      string  `xml:"groupId"`
		ArtifactID   string  `xml:"artifactId"`
		Version      string  `xml:"version"`
		RelativePath *string `xml:"relativePath"`
	} `xml:"parent"`
	RawProperties struct {
		Values []property `xml:",any"`
	} `xml:"properties"`
	Modules      []string     `xml:"modules>module"`
	Dependencies []Dependency `xml:"dependencies>dependency"`
	Managed      []Dependency `xml:"dependencyManagement>dependencies>dependency"`
	Profiles     []struct{}   `xml:"profiles>profile"`
	Build        struct {
		SourceDirectory string     `xml:"sourceDirectory"`
		Resources       []resource `xml:"resources>resource"`
		Plugins         []struct{} `xml:"plugins>plugin"`
		Extensions      []struct{} `xml:"extensions>extension"`
	} `xml:"build"`
}

type Resource struct{ Directory, Target string }
type Module struct {
	Directory, GroupID, ArtifactID, Version, SourceDirectory string
	Resources                                                []Resource
	dependencies                                             []Dependency
	rawDependencies, rawManaged                              []Dependency
	managed                                                  map[string]Dependency
	properties                                               map[string]string
	model                                                    pom
}
type Plan struct{ Modules []*Module }
type resolver struct {
	cache       map[string]*Module
	loading     map[string]bool
	coordinates map[string]*Module
	visited     map[string]bool
}

// Discover reads a reactor and explicit groupId:artifactId=source-project mappings.
// Every compile/runtime/provided dependency must resolve to the exact source
// version. Conflicts are rejected instead of approximating Maven mediation.
func Discover(root string, mappings []string) (*Plan, error) {
	r := &resolver{cache: map[string]*Module{}, loading: map[string]bool{}, coordinates: map[string]*Module{}, visited: map[string]bool{}}
	rootModule, err := r.read(root)
	if err != nil {
		return nil, err
	}
	if err = r.reactor(rootModule); err != nil {
		return nil, err
	}
	for _, mapping := range mappings {
		key, path, ok := strings.Cut(mapping, "=")
		if !ok || strings.Count(key, ":") != 1 || path == "" {
			return nil, fmt.Errorf("invalid dependency source %q; use groupId:artifactId=/path/to/project", mapping)
		}
		module, err := r.read(path)
		if err != nil {
			return nil, err
		}
		if key != module.GroupID+":"+module.ArtifactID {
			return nil, fmt.Errorf("dependency source %s declares %s:%s", key, module.GroupID, module.ArtifactID)
		}
		if err = r.reactor(module); err != nil {
			return nil, err
		}
	}
	// Only the root reactor and reachable mapped projects are compiled.
	reachable := map[string]*Module{}
	var visit func(*Module) error
	visit = func(m *Module) error {
		if reachable[m.Directory] != nil {
			return nil
		}
		reachable[m.Directory] = m
		for _, name := range m.model.Modules {
			path, err := expand(name, m.properties)
			if err != nil {
				return err
			}
			child, err := r.read(filepath.Join(m.Directory, path))
			if err != nil {
				return err
			}
			if err = visit(child); err != nil {
				return err
			}
		}
		for _, dep := range m.dependencies {
			if dep.Scope == "test" {
				continue
			}
			if dep.Scope != "" && dep.Scope != "compile" && dep.Scope != "runtime" && dep.Scope != "provided" {
				return fmt.Errorf("%s: unsupported dependency scope %q for %s", m.Directory, dep.Scope, dep.key())
			}
			if dep.Type != "" && dep.Type != "jar" || dep.Classifier != "" || len(dep.Exclusions) > 0 {
				return fmt.Errorf("%s: dependency %s uses unsupported type, classifier, or exclusions", m.Directory, dep.key())
			}
			target := r.coordinates[dep.key()]
			if target == nil {
				return fmt.Errorf("%s: dependency %s:%s has no implementation source; supply -dependency-source %s=/path/to/Maven/project (binary JARs are insufficient)", m.Directory, dep.key(), dep.Version, dep.key())
			}
			if dep.Version == "" || dep.Version != target.Version {
				return fmt.Errorf("%s: dependency %s requires version %q, but source project provides %q; supply exact source versions", m.Directory, dep.key(), dep.Version, target.Version)
			}
			if err := visit(target); err != nil {
				return err
			}
		}
		return nil
	}
	if err = visit(rootModule); err != nil {
		return nil, err
	}
	plan := &Plan{}
	for _, m := range reachable {
		plan.Modules = append(plan.Modules, m)
	}
	sort.Slice(plan.Modules, func(i, j int) bool { return plan.Modules[i].Directory < plan.Modules[j].Directory })
	return plan, nil
}

func expand(value string, properties map[string]string) (string, error) {
	for n := 0; n < 32; n++ {
		start := strings.Index(value, "${")
		if start < 0 {
			return strings.TrimSpace(value), nil
		}
		end := strings.Index(value[start:], "}")
		if end < 0 {
			break
		}
		key := value[start+2 : start+end]
		replacement, ok := properties[key]
		if !ok {
			return "", fmt.Errorf("unresolved Maven property ${%s}", key)
		}
		value = value[:start] + replacement + value[start+end+1:]
	}
	return "", fmt.Errorf("recursive or malformed Maven property in %q", value)
}
func (r *resolver) read(path string) (*Module, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if filepath.Base(path) == "pom.xml" {
		path = filepath.Dir(path)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if m := r.cache[path]; m != nil {
		return m, nil
	}
	if r.loading[path] {
		return nil, fmt.Errorf("maven parent cycle at %s", path)
	}
	r.loading[path] = true
	defer delete(r.loading, path)
	data, err := os.ReadFile(filepath.Join(path, "pom.xml"))
	if err != nil {
		return nil, fmt.Errorf("read Maven project %s: %w", path, err)
	}
	m := &Module{Directory: path, properties: map[string]string{}, managed: map[string]Dependency{}}
	if err = xml.Unmarshal(data, &m.model); err != nil {
		return nil, fmt.Errorf("parse %s/pom.xml: %w", path, err)
	}
	p := &m.model
	if len(p.Profiles) > 0 || len(p.Build.Plugins) > 0 || len(p.Build.Extensions) > 0 {
		return nil, fmt.Errorf("%s: Maven profiles, build plugins and extensions are unsupported; supply a source-only POM with generated sources already materialized", path)
	}
	if p.Parent != nil {
		relative := "../pom.xml"
		if p.Parent.RelativePath != nil {
			relative = *p.Parent.RelativePath
		}
		if relative == "" {
			return nil, fmt.Errorf("%s: external Maven parent requires a local relativePath", path)
		}
		parent, err := r.read(filepath.Join(path, relative))
		if err != nil {
			return nil, fmt.Errorf("resolve local Maven parent of %s: %w", path, err)
		}
		if parent.GroupID != p.Parent.GroupID || parent.ArtifactID != p.Parent.ArtifactID || parent.Version != p.Parent.Version {
			return nil, fmt.Errorf("%s: local parent coordinates do not match declared parent", path)
		}
		for key, value := range parent.properties {
			m.properties[key] = value
		}
		m.rawManaged = append(m.rawManaged, parent.rawManaged...)
		m.rawDependencies = append(m.rawDependencies, parent.rawDependencies...)
		m.GroupID = parent.GroupID
		m.Version = parent.Version
		if parent.model.Build.SourceDirectory != "" || len(parent.model.Build.Resources) > 0 {
			return nil, fmt.Errorf("%s: inherited custom build paths are unsupported; declare paths in each child POM", path)
		}
	}
	for _, prop := range p.RawProperties.Values {
		m.properties[prop.XMLName.Local] = strings.TrimSpace(prop.Value)
	}
	if p.GroupID != "" {
		m.GroupID = p.GroupID
	}
	if p.Version != "" {
		m.Version = p.Version
	}
	m.ArtifactID = p.ArtifactID
	m.properties["project.basedir"] = path
	m.properties["basedir"] = path
	m.GroupID, err = expand(m.GroupID, m.properties)
	if err != nil {
		return nil, err
	}
	m.Version, err = expand(m.Version, m.properties)
	if err != nil {
		return nil, err
	}
	if m.GroupID == "" || m.ArtifactID == "" || m.Version == "" {
		return nil, fmt.Errorf("%s: Maven groupId, artifactId and version are required", path)
	}
	m.properties["project.groupId"] = m.GroupID
	m.properties["project.artifactId"] = m.ArtifactID
	m.properties["project.version"] = m.Version
	if p.Packaging != "" && p.Packaging != "jar" && p.Packaging != "pom" {
		return nil, fmt.Errorf("%s: unsupported packaging %q (supported: jar, pom)", path, p.Packaging)
	}
	m.rawManaged = append(m.rawManaged, p.Managed...)
	m.rawDependencies = append(m.rawDependencies, p.Dependencies...)
	// Interpolate only after child properties are known. Reusing the parent's
	// expanded model would freeze inherited versions before child overrides.
	for _, dep := range m.rawManaged {
		dep, err = resolveDependency(dep, m.properties)
		if err != nil {
			return nil, err
		}
		if dep.Type == "pom" || dep.Scope == "import" {
			return nil, fmt.Errorf("%s: imported BOMs are unsupported; declare managed dependency versions locally", path)
		}
		if dep.Scope != "" || dep.Type != "" && dep.Type != "jar" || dep.Classifier != "" || len(dep.Exclusions) > 0 {
			return nil, fmt.Errorf("%s: dependencyManagement supports version declarations only", path)
		}
		m.managed[dep.key()] = dep
	}
	inherited := map[string]int{}
	for _, dep := range m.rawDependencies {
		dep, err = resolveDependency(dep, m.properties)
		if err != nil {
			return nil, err
		}
		if dep.Version == "" {
			dep.Version = m.managed[dep.key()].Version
		}
		if i, ok := inherited[dep.key()]; ok {
			m.dependencies[i] = dep
		} else {
			inherited[dep.key()] = len(m.dependencies)
			m.dependencies = append(m.dependencies, dep)
		}
	}
	source := p.Build.SourceDirectory
	if source == "" {
		source = "src/main/java"
	}
	m.SourceDirectory, err = localPath(path, source, m.properties)
	if err != nil {
		return nil, err
	}
	resources := p.Build.Resources
	if resources == nil {
		resources = []resource{{Directory: "src/main/resources"}}
	}
	for _, res := range resources {
		if res.Filtering || len(res.Includes) > 0 || len(res.Excludes) > 0 {
			return nil, fmt.Errorf("%s: filtered resources and include/exclude patterns are unsupported", path)
		}
		directory, err := localPath(path, res.Directory, m.properties)
		if err != nil {
			return nil, err
		}
		target, err := expand(res.TargetPath, m.properties)
		if err != nil {
			return nil, err
		}
		if filepath.IsAbs(target) || target == ".." || strings.HasPrefix(filepath.Clean(target), ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s: resource targetPath must stay within resources", path)
		}
		m.Resources = append(m.Resources, Resource{directory, target})
	}
	r.cache[path] = m
	return m, nil
}
func localPath(root, value string, properties map[string]string) (string, error) {
	value, err := expand(value, properties)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("empty Maven source/resource directory")
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	return filepath.Join(root, value), nil
}
func resolveDependency(d Dependency, p map[string]string) (Dependency, error) {
	for _, field := range []*string{&d.GroupID, &d.ArtifactID, &d.Version, &d.Scope, &d.Type, &d.Classifier} {
		value, err := expand(*field, p)
		if err != nil {
			return d, err
		}
		*field = value
	}
	return d, nil
}
func (r *resolver) reactor(m *Module) error {
	if r.visited[m.Directory] {
		return nil
	}
	r.visited[m.Directory] = true
	key := m.GroupID + ":" + m.ArtifactID
	if previous := r.coordinates[key]; previous != nil && previous.Directory != m.Directory {
		return fmt.Errorf("conflicting source projects for %s: %s and %s", key, previous.Directory, m.Directory)
	}
	r.coordinates[key] = m
	for _, name := range m.model.Modules {
		value, err := expand(name, m.properties)
		if err != nil {
			return err
		}
		child, err := r.read(filepath.Join(m.Directory, value))
		if err != nil {
			return err
		}
		if err = r.reactor(child); err != nil {
			return err
		}
	}
	return nil
}
