package transpiler

import (
	"bytes"
	"fmt"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/project"
	"github.com/NickyBoy89/java2go/symbol"
)

type repeatedFlag []string

func (f *repeatedFlag) String() string         { return strings.Join(*f, ", ") }
func (f *repeatedFlag) Set(value string) error { *f = append(*f, value); return nil }

// Project conversion stages a complete module before publishing it. Failed
// dependency resolution, conversion, or cycle checks leave the output untouched.
func runMavenProject(root string, mappings []string, mainClass, runtimeRoot, output, module, excluded string, stdout io.Writer) error {
	if mainClass == "" || runtimeRoot == "" {
		return fmt.Errorf("-maven requires -main-class fully.qualified.Class and -runtime /path/to/java2go")
	}
	if !validProjectModule(module) {
		return fmt.Errorf("invalid Go module path %q", module)
	}
	runtimeRoot, err := filepath.Abs(runtimeRoot)
	if err != nil {
		return err
	}
	runtimeMod, err := os.ReadFile(filepath.Join(runtimeRoot, "go.mod"))
	if err != nil {
		return fmt.Errorf("read java2go runtime module: %w", err)
	}
	if !strings.HasPrefix(string(runtimeMod), "module github.com/NickyBoy89/java2go\n") {
		return fmt.Errorf("-runtime must identify the java2go repository")
	}
	if info, err := os.Stat(filepath.Join(runtimeRoot, "stdjava")); err != nil || !info.IsDir() {
		return fmt.Errorf("-runtime is missing stdjava")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(output); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("project output %s must be a directory, not a file or symlink", output)
	}
	if entries, err := os.ReadDir(output); err == nil && len(entries) > 0 {
		return fmt.Errorf("project output %s must be absent or empty", output)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	plan, err := project.Discover(root, mappings)
	if err != nil {
		return err
	}
	for _, m := range plan.Modules {
		roots := []string{m.SourceDirectory}
		for _, resource := range m.Resources {
			roots = append(roots, resource.Directory)
		}
		for _, root := range roots {
			rel, err := filepath.Rel(root, output)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return fmt.Errorf("project output must be outside production source/resource directory %s", root)
			}
		}
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(output), ".java2go-project-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	inputs := filepath.Join(staging, "sources")
	generated := filepath.Join(staging, "output")
	if err = os.MkdirAll(inputs, 0755); err != nil {
		return err
	}
	if err = os.MkdirAll(generated, 0755); err != nil {
		return err
	}
	packages := map[string]string{}
	classes := map[string]string{}
	sourceCount := 0
	for _, m := range plan.Modules {
		if _, err = os.Stat(m.SourceDirectory); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		err = filepath.WalkDir(m.SourceDirectory, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("symlink in source directory is unsupported: %s", path)
			}
			if d.IsDir() || filepath.Ext(path) != ".java" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			file := parsing.SourceFile{Name: path, Source: source}
			if err = file.ParseAST(); err != nil {
				return err
			}
			if file.Ast.HasError() {
				return fmt.Errorf("java parse error in %s", path)
			}
			if filepath.Base(path) == "module-info.java" {
				return fmt.Errorf("JPMS module descriptors are unsupported: %s", path)
			}
			// Package annotations have no executable declarations to convert.
			if filepath.Base(path) == "package-info.java" {
				return nil
			}
			symbols := file.ParseSymbols()
			if symbols.BaseClass == nil {
				return fmt.Errorf("no Java type declaration in %s", path)
			}
			pkg := symbols.Package
			if pkg == "" {
				return fmt.Errorf("maven source %s requires a named Java package", path)
			}
			for _, class := range symbols.TopLevelClasses {
				key := pkg + "." + class.Class.OriginalName
				if previous := classes[key]; previous != "" {
					return fmt.Errorf("duplicate Java type %s in %s and %s", key, previous, path)
				}
				classes[key] = path
			}
			packagePath := strings.ReplaceAll(pkg, ".", "/")
			if standard, err := build.Default.Import(packagePath, "", build.FindOnly); err == nil && standard.Goroot {
				return fmt.Errorf("java package %s conflicts with a Go standard-library import; use a different qualified Java package", pkg)
			}
			packages[packagePath] = pkg
			target := filepath.Join(inputs, filepath.FromSlash(projectPackagePath(packagePath)), filepath.Base(path))
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("duplicate source filename in Java package %s: %s", pkg, path)
			}
			sourceCount++
			return writeProjectFile(target, source)
		})
		if err != nil {
			return err
		}
	}
	if sourceCount == 0 {
		return fmt.Errorf("maven project has no production Java sources")
	}
	if classes[mainClass] == "" {
		return fmt.Errorf("main class %s is not in the resolved production sources", mainClass)
	}
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	defer func() { symbol.GlobalScope = previous }()
	args := []string{"-w", "-strict", "-sync", "-output", generated, "-module", "", "-exclude-annotations", excluded, inputs}
	if err = runInternal(args, stdout, true); err != nil {
		return err
	}
	split := strings.LastIndex(mainClass, ".")
	mainPackage := mainClass[:split]
	mainType := mainClass[split+1:]
	scope := symbol.GlobalScope.FindPackage(mainPackage)
	entry := ""
	if scope != nil {
		for _, file := range scope.Files {
			class := file.FindClassScope(mainType)
			if class == nil {
				continue
			}
			for _, method := range class.Methods {
				if projectMain(method) {
					entry = method.Name
				}
			}
		}
	}
	if entry == "" {
		return fmt.Errorf("%s must declare public static void main(String[])", mainClass)
	}
	graph := map[string]map[string]bool{}
	paths := make([]string, 0, len(packages))
	for path := range packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, packagePath := range paths {
		graph[packagePath] = map[string]bool{}
		err = filepath.WalkDir(filepath.Join(generated, filepath.FromSlash(projectPackagePath(packagePath))), func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if path != filepath.Join(generated, filepath.FromSlash(projectPackagePath(packagePath))) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("parse generated Go %s: %w", path, err)
			}
			name := packagePath[strings.LastIndex(packagePath, "/")+1:]
			file.Name.Name = "j_" + sanitizeGoIdent(name)
			for _, spec := range file.Imports {
				importPath, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				if _, ok := packages[importPath]; ok {
					graph[packagePath][importPath] = true
					spec.Path.Value = strconv.Quote(module + "/" + projectPackagePath(importPath))
				}
			}
			var data bytes.Buffer
			if err = format.Node(&data, set, file); err != nil {
				return err
			}
			return os.WriteFile(path, data.Bytes(), 0644)
		})
		if err != nil {
			return err
		}
	}
	if cycle := projectImportCycle(graph); len(cycle) > 0 {
		return fmt.Errorf("java package cycle cannot be represented as Go imports: %s; move mutually dependent classes into one Java package before conversion", strings.Join(cycle, " -> "))
	}
	launcher := fmt.Sprintf(`package main
import (
 app %q
 "os"
 stdjava "github.com/NickyBoy89/java2go/stdjava"
)
func main() {
 args := stdjava.NewReferenceArrayOf[string](len(os.Args)-1, stdjava.StringTypeID)
 for i, value := range os.Args[1:] { stdjava.ReferenceArraySet(args, i, value) }
 app.%s(args)
}
`, module+"/"+projectPackagePath(strings.ReplaceAll(mainPackage, ".", "/")), entry)
	data, err := format.Source([]byte(launcher))
	if err != nil {
		return err
	}
	if err = writeProjectFile(filepath.Join(generated, "cmd/app/main.go"), data); err != nil {
		return err
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => %s\n", module, strconv.Quote(filepath.ToSlash(runtimeRoot)))
	if err = writeProjectFile(filepath.Join(generated, "go.mod"), []byte(goMod)); err != nil {
		return err
	}
	for _, m := range plan.Modules {
		for _, resource := range m.Resources {
			if err = copyProjectResources(resource, filepath.Join(generated, "resources")); err != nil {
				return err
			}
		}
	}
	// Refuse to replace any files produced by another writer while converting.
	if err = os.Remove(output); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("publish project output: %w", err)
	}
	if err = os.Rename(generated, output); err != nil {
		return fmt.Errorf("publish project output: %w", err)
	}
	return nil
}
func validProjectModule(module string) bool {
	if module == "" || strings.HasPrefix(module, "/") || strings.HasSuffix(module, "/") {
		return false
	}
	for _, part := range strings.Split(module, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		for _, r := range part {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && !strings.ContainsRune(".-_~", r) {
				return false
			}
		}
	}
	return true
}
func writeProjectFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
func projectMain(d *symbol.Definition) bool {
	if d == nil || d.OriginalName != "main" || !d.IsStatic || d.OriginalType != "void" || len(d.Parameters) != 1 {
		return false
	}
	parameter := strings.ReplaceAll(d.Parameters[0].OriginalType, " ", "")
	if parameter != "String[]" && parameter != "java.lang.String[]" {
		return false
	}
	if d.DeclarationNode == nil {
		return false
	}
	modifiers := d.DeclarationNode.NamedChild(0)
	if modifiers == nil || modifiers.Type() != "modifiers" {
		return false
	}
	for i := 0; i < int(modifiers.ChildCount()); i++ {
		if modifiers.Child(i).Type() == "public" {
			return true
		}
	}
	return false
}
func prepareProjectEntrypoints(files []parsing.SourceFile) {
	used := map[string]map[string]bool{}
	for _, file := range files {
		if used[file.Symbols.Package] == nil {
			used[file.Symbols.Package] = map[string]bool{}
		}
		names := used[file.Symbols.Package]
		for _, class := range file.Symbols.TopLevelClasses {
			names[class.Class.Name] = true
			for _, method := range class.Methods {
				names[method.Name] = true
			}
			for _, field := range class.Fields {
				names[field.Name] = true
			}
		}
	}
	for _, file := range files {
		for _, class := range file.Symbols.TopLevelClasses {
			for _, method := range class.Methods {
				if method.OriginalName != "main" || !method.IsStatic {
					continue
				}
				base := "Java2goEntry" + class.Class.Name
				name := base
				for i := 1; used[file.Symbols.Package][name]; i++ {
					name = fmt.Sprintf("%s%d", base, i)
				}
				used[file.Symbols.Package][name] = true
				method.Name = name
			}
		}
	}
}
func copyProjectResources(resource project.Resource, root string) error {
	if _, err := os.Stat(resource.Directory); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return filepath.WalkDir(resource.Directory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink resource is unsupported: %s", path)
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(resource.Directory, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, resource.Target, rel)
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("duplicate production resource %s", filepath.Join(resource.Target, rel))
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeProjectFile(target, data)
	})
}
func projectImportCycle(graph map[string]map[string]bool) []string {
	states := map[string]int{}
	stack := []string{}
	var visit func(string) []string
	visit = func(node string) []string {
		if states[node] == 1 {
			for i, n := range stack {
				if n == node {
					return append(append([]string{}, stack[i:]...), node)
				}
			}
		}
		if states[node] == 2 {
			return nil
		}
		states[node] = 1
		stack = append(stack, node)
		edges := []string{}
		for edge := range graph[node] {
			edges = append(edges, edge)
		}
		sort.Strings(edges)
		for _, edge := range edges {
			if cycle := visit(edge); cycle != nil {
				return cycle
			}
		}
		stack = stack[:len(stack)-1]
		states[node] = 2
		return nil
	}
	nodes := []string{}
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if cycle := visit(node); cycle != nil {
			return cycle
		}
	}
	return nil
}

// Prefix every Java package component so vendor, internal, testdata, leading
// underscores and Go tooling's other special directory names remain ordinary
// generated packages. The mapping is deterministic and injective.
func projectPackagePath(javaPath string) string {
	parts := strings.Split(javaPath, "/")
	for i, part := range parts {
		parts[i] = "j_" + part
	}
	return strings.Join(parts, "/")
}
