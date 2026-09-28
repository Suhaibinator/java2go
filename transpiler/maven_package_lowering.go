package transpiler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
		packageFiles := map[string]*ast.File{}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
			if err != nil {
				return "", "", fmt.Errorf("parse generated Go %s: %w", path, err)
			}
			packageFiles[path] = file
			files = append(files, &projectGoFile{path, pkg, file})
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				if _, ok := packages[imported]; ok {
					graph[pkg][imported] = true
				}
			}
		}
		// Resolve package declarations across files as well as lexical locals. The
		// import objects let us distinguish aliases from shadowing local variables.
		_, _ = ast.NewPackage(set, packageFiles, func(_ map[string]*ast.Object, path string) (*ast.Object, error) {
			obj := ast.NewObj(ast.Pkg, filepath.Base(path))
			obj.Data = ast.NewScope(nil)
			return obj, nil
		}, nil)
	}
	destinations, prefixes := projectPackageComponents(graph)
	renames := map[*ast.Object]string{}
	embeddedNames := map[string]string{}
	for _, file := range files {
		prefix := prefixes[file.javaPackage]
		if prefix == "" {
			continue
		}
		for name, obj := range file.file.Scope.Objects {
			if name == "init" || name == "_" {
				continue
			}
			renames[obj] = projectNamespacedName(prefix, name)
			if obj.Kind == ast.Typ {
				embeddedNames[renames[obj]] = name
			}
		}
	}
	for _, source := range files {
		pkg, file := source.javaPackage, source.file
		file.Name.Name = sanitizeGoIdent(filepath.Base(destinations[pkg]))
		// Rewrite qualified project references before removing internal imports.
		projectRewriteAST(reflect.ValueOf(file), func(node ast.Node) ast.Node {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return node
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || ident.Obj == nil || ident.Obj.Kind != ast.Pkg {
				return node
			}
			spec, ok := ident.Obj.Decl.(*ast.ImportSpec)
			if !ok {
				return node
			}
			imported, _ := strconv.Unquote(spec.Path.Value)
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
		keys := map[*ast.Ident]bool{}
		ast.Inspect(file, func(node ast.Node) bool {
			if pair, ok := node.(*ast.KeyValueExpr); ok {
				if ident, ok := pair.Key.(*ast.Ident); ok {
					keys[ident] = true
				}
			}
			return true
		})
		ast.Inspect(file, func(node ast.Node) bool {
			if ident, ok := node.(*ast.Ident); ok {
				if name, ok := renames[ident.Obj]; ok && !keys[ident] {
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
	projectRetargetEmbeddedFields(set, files, module, runtimeRoot, destinations, prefixes, embeddedNames, renames)
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
