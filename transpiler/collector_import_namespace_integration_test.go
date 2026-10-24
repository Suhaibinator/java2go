package transpiler

import "testing"

func TestCollectorImportNamespaceStructuralComments(t *testing.T) {
	ordinary := `class Use{Object run(){return Collectors.toCollection(null);}Object other(){return null;}}`
	for _, row := range []struct{ name, imports string }{
		{"plain", `import java.util.stream.Collectors;`},
		{"after_keyword_block", `import /* ordinary extra */ java.util.stream.Collectors;`},
		{"after_keyword_line", "import // ordinary extra\njava.util.stream.Collectors;"},
		{"after_first_dot", `import java./* ordinary extra */util.stream.Collectors;`},
		{"before_last_dot", `import java.util.stream/* ordinary extra */.Collectors;`},
		{"before_terminal", `import java.util.stream./* ordinary extra */Collectors;`},
		{"before_semicolon", `import java.util.stream.Collectors/* ordinary extra */;`},
		{"leading_comment", `/* ordinary extra */ import java.util.stream.Collectors;`},
		{"duplicate_same_import", `import java.util.stream.Collectors;import /* ordinary extra */ java.util.stream.Collectors;`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h, ctx, node := deferredCollectorFactoryNamespaceUse(t, row.imports+ordinary)
			if !anonymousFunctionalNamespace(ctx.currentFile) {
				t.Fatal("actual structural import did not attest the retained namespace")
			}
			if _, anonymous := anonymousFunctionalLookup(ctx.currentClass, ctx); anonymous {
				t.Fatal("actual named declaration was routed as an anonymous declaration")
			}
			if declaring, owned := supplierSourceDeclarationCtx(ctx.currentClass, ctx); !owned || declaring.currentFile != ctx.currentFile {
				t.Fatal("actual named source owner lost its declaration context")
			}
			if !sourceArrayListNativeSourceContext(node, ctx, h.File.Source) {
				t.Fatal("actual occurrence lost native source ownership")
			}
			if name, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); !known || name != "toCollection" {
				t.Fatal("canonical factory lost original structural import authority")
			}
		})
	}
}

func TestCollectorImportNamespaceRefusesDonatedAuthority(t *testing.T) {
	ordinary := `class Use{Object run(){return Collectors.toCollection(null);}Object other(){return null;}}`
	canonical := `import /* ordinary extra */ java.util.stream.Collectors;`
	for _, name := range []string{"metadata_changed", "metadata_missing", "metadata_extra", "registered_type_census_missing"} {
		t.Run(name, func(t *testing.T) {
			h, ctx, node := deferredCollectorFactoryNamespaceUse(t, canonical+ordinary)
			positive := func() {
				t.Helper()
				if !anonymousFunctionalNamespace(ctx.currentFile) {
					t.Fatal("original positive namespace premise missing")
				}
				if selected, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); !known || selected != "toCollection" {
					t.Fatal("original positive factory premise missing")
				}
			}
			positive()
			file := ctx.currentFile
			originalImports, originalClasses := file.Imports, file.TopLevelClasses
			copied := make(map[string]string, len(file.Imports))
			for key, value := range file.Imports {
				copied[key] = value
			}
			file.Imports = copied
			switch name {
			case "metadata_changed":
				file.Imports["Collectors"] = "unrelated"
			case "metadata_missing":
				delete(file.Imports, "Collectors")
			case "metadata_extra":
				file.Imports["Injected"] = "unrelated"
			case "registered_type_census_missing":
				file.TopLevelClasses = nil
			}
			if anonymousFunctionalNamespace(file) {
				t.Fatal("mutated retained metadata donated original namespace authority")
			}
			if _, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); known {
				t.Fatal("mutated retained metadata donated canonical factory authority")
			}
			file.Imports, file.TopLevelClasses = originalImports, originalClasses
			positive()
		})
	}
	for _, row := range []struct {
		name, imports      string
		namespace, factory bool
	}{
		{"conflict_canonical_first", canonical + `import unrelated.Collectors;`, false, false},
		{"conflict_canonical_last", `import unrelated.Collectors;` + canonical, false, false},
		{"static_method_not_type", `import static /* ordinary extra */ java.util.stream.Collectors.toCollection;`, true, false},
		{"static_star_not_type", `import static /* ordinary extra */ java.util.stream.Collectors.*;`, true, false},
		{"on_demand_retains_ordinary_namespace", `import java.util.stream.*;`, true, true},
	} {
		t.Run(row.name, func(t *testing.T) {
			h, ctx, node := deferredCollectorFactoryNamespaceUse(t, row.imports+ordinary)
			if got := anonymousFunctionalNamespace(ctx.currentFile); got != row.namespace {
				t.Fatalf("original namespace identity: got=%v want=%v", got, row.namespace)
			}
			if selected, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); known != row.factory || known && selected != "toCollection" {
				t.Fatalf("original import kind/duplicate authority: name=%q known=%v want=%v", selected, known, row.factory)
			}
		})
	}
}
