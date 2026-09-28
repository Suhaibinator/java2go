package transpiler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
)

type projectGoFile struct {
	path, javaPackage string
	file              *ast.File
}

// Java permits package cycles. Collapse their strongly connected components at
// the Go boundary, after Java name resolution and metadata generation. Original
// package identity remains in descriptor literals; only Go declarations move.
func lowerProjectPackages(root, module, runtimeRoot string, packages map[string]string, mainPackage, entry string) (string, string, error) {
	set := token.NewFileSet()
	graph := map[string]map[string]bool{}
	files := []*projectGoFile{}
	groups := map[string][]*ast.File{}
	paths := make([]string, 0, len(packages))
	for path := range packages {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, pkg := range paths {
		graph[pkg] = map[string]bool{}
		dir := filepath.Join(root, filepath.FromSlash(projectPackagePath(pkg)))
		entries, err := os.ReadDir(dir)
		if err != nil {
			return "", "", err
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(set, path, nil, parser.ParseComments|parser.SkipObjectResolution)
			if err != nil {
				return "", "", fmt.Errorf("parse generated Go %s: %w", path, err)
			}
			groups[pkg] = append(groups[pkg], file)
			files = append(files, &projectGoFile{path, pkg, file})
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				if _, ok := packages[imported]; ok {
					graph[pkg][imported] = true
				}
			}
		}
	}
	// Original Java packages may cycle. Check each package's lexical scopes
	// independently, with complete empty import namespaces for project imports.
	// Sharing an unfinished checker's real scope can expose TypeNames whose
	// types are still nil and violate go/types invariants. Actual cross-package
	// types are analyzed only after the SCCs have been merged below.
	bindings := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Scopes: map[ast.Node]*types.Scope{}}
	imports := &projectTypeAnalysis{packages: map[string]*types.Package{}, fallback: importer.Default()}
	for _, path := range paths {
		name := sanitizeGoIdent(filepath.Base(path))
		if len(groups[path]) > 0 {
			name = groups[path][0].Name.Name
		}
		pkg := types.NewPackage(path, name)
		pkg.MarkComplete()
		imports.packages[path] = pkg
	}
	boundPackages := map[string]*types.Package{}
	for _, path := range paths {
		pkg := types.NewPackage(path, imports.packages[path].Name())
		config := types.Config{Importer: imports, Error: func(error) {}}
		_ = types.NewChecker(&config, set, pkg, bindings).Files(groups[path])
		boundPackages[path] = pkg
	}

	destinations, prefixes := projectPackageComponents(graph)
	renames := map[types.Object]string{}
	embeddedNames := map[string]string{}
	keyRenames := map[*ast.Ident]string{}
	for _, path := range paths {
		prefix := prefixes[path]
		if prefix == "" {
			continue
		}
		scope := boundPackages[path].Scope()
		for _, name := range scope.Names() {
			if name == "init" || name == "_" {
				continue
			}
			obj := scope.Lookup(name)
			renames[obj] = projectNamespacedName(prefix, name)
			if _, ok := obj.(*types.TypeName); ok {
				embeddedNames[renames[obj]] = name
			}
		}
	}
	for _, source := range files {
		pkg, file := source.javaPackage, source.file
		objects, keys := projectLexicalBindings(file, bindings)
		for key := range keys {
			if name := renames[objects[key]]; name != "" {
				keyRenames[key] = name
			}
		}
		file.Name.Name = sanitizeGoIdent(filepath.Base(destinations[pkg]))
		// Rewrite qualified project references before removing internal imports.
		projectRewriteAST(reflect.ValueOf(file), func(node ast.Node) ast.Node {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return node
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok {
				return node
			}
			importedPackage, ok := objects[ident].(*types.PkgName)
			if !ok {
				return node
			}
			imported := importedPackage.Imported().Path()
			if _, ok := packages[imported]; !ok {
				return node
			}
			name := projectNamespacedName(prefixes[imported], selector.Sel.Name)
			if destinations[pkg] == destinations[imported] {
				return &ast.Ident{NamePos: selector.Pos(), Name: name}
			}
			selector.Sel.Name = name
			return node
		})
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok {
				if name := renames[objects[ident]]; name != "" && !keys[ident] {
					ident.Name = name
				}
			}
			return true
		})
		projectNamespacePrivateMembers(file, prefixes[pkg])
		imports := []*ast.ImportSpec{}
		declarations := []ast.Decl{}
		for _, decl := range file.Decls {
			group, ok := decl.(*ast.GenDecl)
			if !ok || group.Tok != token.IMPORT {
				declarations = append(declarations, decl)
				continue
			}
			specs := []ast.Spec{}
			for _, raw := range group.Specs {
				spec := raw.(*ast.ImportSpec)
				imported, _ := strconv.Unquote(spec.Path.Value)
				if _, ok := packages[imported]; ok {
					if destinations[pkg] == destinations[imported] {
						continue
					}
					spec.Path.Value = strconv.Quote(module + "/" + destinations[imported])
				}
				specs = append(specs, spec)
				imports = append(imports, spec)
			}
			if len(specs) > 0 {
				group.Specs = specs
				declarations = append(declarations, group)
			}
		}
		file.Decls, file.Imports = declarations, imports
	}
	projectRetargetEmbeddedFields(set, files, module, runtimeRoot, destinations, prefixes, embeddedNames, keyRenames)
	for _, source := range files {
		var data bytes.Buffer
		if err := format.Node(&data, set, source.file); err != nil {
			return "", "", err
		}
		target := source.path
		if prefixes[source.javaPackage] != "" {
			target = filepath.Join(root, destinations[source.javaPackage], hex.EncodeToString([]byte(source.javaPackage))+"_"+filepath.Base(source.path))
		}
		if err := writeProjectFile(target, data.Bytes()); err != nil {
			return "", "", err
		}
		if target != source.path {
			if err := os.Remove(source.path); err != nil {
				return "", "", err
			}
		}
	}
	return module + "/" + destinations[mainPackage], projectNamespacedName(prefixes[mainPackage], entry), nil
}

func projectNamespacedName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	if ast.IsExported(name) {
		return "J" + prefix + "_" + name
	}
	return "j" + prefix + "_" + name
}

// The destination and identifier encodings are injective. Project package paths
// always start with j_, reserving java2go_scc_ for synthetic components.
func projectPackageComponents(graph map[string]map[string]bool) (map[string]string, map[string]string) {
	destinations, prefixes := map[string]string{}, map[string]string{}
	indices, low := map[string]int{}, map[string]int{}
	stack := []string{}
	active := map[string]bool{}
	next := 0
	var visit func(string)
	visit = func(node string) {
		next++
		indices[node], low[node] = next, next
		stack = append(stack, node)
		active[node] = true
		edges := []string{}
		for edge := range graph[node] {
			edges = append(edges, edge)
		}
		sort.Strings(edges)
		for _, edge := range edges {
			if indices[edge] == 0 {
				visit(edge)
				if low[edge] < low[node] {
					low[node] = low[edge]
				}
			} else if active[edge] && indices[edge] < low[node] {
				low[node] = indices[edge]
			}
		}
		if low[node] != indices[node] {
			return
		}
		component := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			active[last] = false
			component = append(component, last)
			if last == node {
				break
			}
		}
		sort.Strings(component)
		cyclic := len(component) > 1 || graph[node][node]
		for _, pkg := range component {
			destinations[pkg] = projectPackagePath(pkg)
			if cyclic {
				destinations[pkg] = "java2go_scc_" + hex.EncodeToString([]byte(component[0]))
				prefixes[pkg] = hex.EncodeToString([]byte(pkg))
			}
		}
	}
	nodes := []string{}
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if indices[node] == 0 {
			visit(node)
		}
	}
	return destinations, prefixes
}

// Walk only AST edges: Object/Scope links form cycles and are bindings, not
// syntax. Reflection keeps expression replacement complete for generic syntax
// and future Go AST expression nodes without text-based substitutions.
func projectRewriteAST(value reflect.Value, rewrite func(ast.Node) ast.Node) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		if node, ok := value.Interface().(ast.Node); ok {
			value.Set(reflect.ValueOf(rewrite(node)))
		}
		projectRewriteAST(value.Elem(), rewrite)
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		if _, ok := value.Interface().(ast.Node); !ok {
			return
		}
		projectRewriteAST(value.Elem(), rewrite)
		return
	}
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i).Name
			if field != "Obj" && field != "Scope" && field != "Unresolved" {
				projectRewriteAST(value.Field(i), rewrite)
			}
		}
	}
	if value.Kind() == reflect.Slice {
		for i := 0; i < value.Len(); i++ {
			projectRewriteAST(value.Index(i), rewrite)
		}
	}
}

// Go's unexported member identity includes its package. Preserve that identity
// when Java packages cohabit one SCC package, so an unrelated package-private
// method cannot become an override merely because its spelling matches.
func projectNamespacePrivateMembers(file *ast.File, prefix string) {
	if prefix == "" {
		return
	}
	rename := func(ident *ast.Ident) {
		if ident.Name != "_" && !ast.IsExported(ident.Name) {
			ident.Name = projectNamespacedName(prefix, ident.Name)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.FuncDecl:
			if value.Recv != nil {
				rename(value.Name)
			}
		case *ast.StructType:
			for _, field := range value.Fields.List {
				for _, name := range field.Names {
					rename(name)
				}
			}
		case *ast.InterfaceType:
			for _, field := range value.Methods.List {
				for _, name := range field.Names {
					rename(name)
				}
			}
		}
		return true
	})
}
