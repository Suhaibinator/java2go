package transpiler

import (
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
)

// Renaming an anonymous field's named type also changes its Go field name.
// Retarget selections by receiver type, never by selector spelling alone: an
// ordinary field or local with the same name must retain its identity.
func projectRetargetEmbeddedFields(set *token.FileSet, files []*projectGoFile, module, runtimeRoot string, destinations, prefixes map[string]string, embeddedNames map[string]string, renames map[*ast.Object]string) {
	if len(embeddedNames) == 0 {
		return
	}
	groups := map[string][]*ast.File{}
	selectors := []*ast.SelectorExpr{}
	composites := []*ast.CompositeLit{}
	origins := map[ast.Node]string{}
	for _, file := range files {
		path := module + "/" + destinations[file.javaPackage]
		groups[path] = append(groups[path], file.file)
		ast.Inspect(file.file, func(node ast.Node) bool {
			if lit, ok := node.(*ast.CompositeLit); ok {
				composites = append(composites, lit)
				origins[lit] = file.javaPackage
			}
			if sel, ok := node.(*ast.SelectorExpr); ok {
				selectors = append(selectors, sel)
				origins[sel] = file.javaPackage
			}
			return true
		})
	}
	// Runtime generic helpers (notably ReferenceCast[T]) carry source receiver
	// types through expressions. Read their declarations so selector binding
	// remains precise without requiring installed Go export data or building
	// the runtime as a side effect of source conversion.
	if runtimeRoot != "" {
		dir := filepath.Join(runtimeRoot, "stdjava")
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			if match, _ := build.Default.MatchFile(dir, entry.Name()); !match {
				continue
			}
			if file, err := parser.ParseFile(set, filepath.Join(dir, entry.Name()), nil, 0); err == nil {
				groups[stdjavaImportPath] = append(groups[stdjavaImportPath], file)
			}
		}
	}
	// A repaired selection can expose the receiver type of the next selection in
	// a chain. Each successful pass repairs at least one previously old name.
	for pass := 0; pass <= len(selectors)+len(composites); pass++ {
		analysis := &projectTypeAnalysis{set: set, files: groups, packages: map[string]*types.Package{}, info: &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}, fallback: importer.Default()}
		for path := range groups {
			_, _ = analysis.Import(path)
		}
		changed := false
		for _, selector := range selectors {
			receiver := analysis.info.TypeOf(selector.X)
			if receiver == nil {
				continue
			}
			origin := origins[selector]
			if !ast.IsExported(selector.Sel.Name) && prefixes[origin] != "" {
				name := projectNamespacedName(prefixes[origin], selector.Sel.Name)
				pkg := analysis.packages[module+"/"+destinations[origin]]
				if member, _, _ := types.LookupFieldOrMethod(receiver, true, pkg, name); member != nil {
					selector.Sel.Name = name
					changed = true
					continue
				}
			}
			if name := projectEmbeddedSelection(receiver, selector.Sel.Name, embeddedNames); name != "" && name != selector.Sel.Name {
				selector.Sel.Name = name
				changed = true
			}
		}
		for _, literal := range composites {
			typ := analysis.info.TypeOf(literal)
			if typ == nil {
				continue
			}
			for _, element := range literal.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok {
					continue
				}
				replacement := ""
				switch actual := typ.Underlying().(type) {
				case *types.Map:
					replacement = renames[key.Obj]
				case *types.Struct:
					origin := origins[literal]
					privateName := ""
					if !ast.IsExported(key.Name) && prefixes[origin] != "" {
						privateName = projectNamespacedName(prefixes[origin], key.Name)
					}
					for i := 0; i < actual.NumFields(); i++ {
						field := actual.Field(i)
						if field.Name() == privateName {
							replacement = privateName
							break
						}
						if field.Embedded() && embeddedNames[field.Name()] == key.Name {
							replacement = field.Name()
							break
						}
					}
				}
				if replacement != "" && replacement != key.Name {
					key.Name = replacement
					changed = true
				}
			}
		}
		if !changed {
			return
		}
	}
}

type projectTypeAnalysis struct {
	set      *token.FileSet
	files    map[string][]*ast.File
	packages map[string]*types.Package
	info     *types.Info
	fallback types.Importer
}

func (a *projectTypeAnalysis) Import(path string) (*types.Package, error) {
	if pkg := a.packages[path]; pkg != nil {
		return pkg, nil
	}
	files := a.files[path]
	if len(files) == 0 {
		pkg, err := a.fallback.Import(path)
		if err != nil {
			pkg = types.NewPackage(path, path[strings.LastIndex(path, "/")+1:])
			pkg.MarkComplete()
		}
		a.packages[path] = pkg
		return pkg, nil
	}
	pkg := types.NewPackage(path, files[0].Name.Name)
	a.packages[path] = pkg
	config := types.Config{Importer: a, Error: func(error) {}, IgnoreFuncBodies: path == stdjavaImportPath}
	// This is binding analysis, not validation: unrelated unsupported runtime
	// signatures must not suppress usable source field/type information.
	_ = types.NewChecker(&config, a.set, pkg, a.info).Files(files)
	return pkg, nil
}

func projectEmbeddedSelection(receiver types.Type, name string, originalNames map[string]string) string {
	frontier := []types.Type{receiver}
	seen := map[types.Type]bool{}
	for len(frontier) > 0 {
		next := []types.Type{}
		matches := 0
		renamed := ""
		for _, typ := range frontier {
			typ = types.Unalias(typ)
			if pointer, ok := typ.(*types.Pointer); ok {
				typ = types.Unalias(pointer.Elem())
			}
			if seen[typ] {
				continue
			}
			seen[typ] = true
			if named, ok := typ.(*types.Named); ok {
				for i := 0; i < named.NumMethods(); i++ {
					if named.Method(i).Name() == name {
						matches++
					}
				}
			}
			structure, ok := typ.Underlying().(*types.Struct)
			if !ok {
				continue
			}
			for i := 0; i < structure.NumFields(); i++ {
				field := structure.Field(i)
				oldName := field.Name()
				if field.Embedded() && originalNames[oldName] != "" {
					oldName = originalNames[oldName]
				}
				if oldName == name {
					matches++
					if field.Embedded() {
						renamed = field.Name()
					}
				}
				if field.Embedded() {
					next = append(next, field.Type())
				}
			}
		}
		if matches > 0 {
			if matches == 1 {
				return renamed
			}
			return ""
		}
		frontier = next
	}
	return ""
}
