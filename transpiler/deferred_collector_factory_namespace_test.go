package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func deferredCollectorFactoryNamespaceUse(t *testing.T, source string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	if owner == nil {
		t.Fatal("actual Use declaration missing")
	}
	method := owner.FindMethodByName("run", nil)
	if method == nil || method.DeclarationNode == nil {
		t.Fatal("actual run declaration missing")
	}
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	node := findNode(method.DeclarationNode.ChildByFieldName("body"), "method_invocation")
	if node == nil {
		t.Fatal("actual factory use missing")
	}
	return h, ctx, node
}

func TestDeferredCollectorFactoryOriginalCanonicalNamespace(t *testing.T) {
	ordinary := `class Use{Object run(){return Collectors.toCollection(null);}Object other(){return null;}}`
	for _, row := range []struct {
		name, source string
		want         bool
	}{
		{"explicit", `import java.util.stream.Collectors;` + ordinary, true},
		{"on_demand", `import java.util.stream.*;` + ordinary, true},
		{"qualified", `class Use{Object run(){return java.util.stream.Collectors.toCollection(null);}Object other(){return null;}}`, true},
		{"duplicate_same_import", `import java.util.stream.Collectors;import java.util.stream.Collectors;` + ordinary, true},
		{"commented_import", `import /* actual declaration */ java.util.stream.Collectors;` + ordinary, true},
		{"missing_import", ordinary, false},
		{"wrong_import", `import unrelated.Collectors;` + ordinary, false},
		{"source_type", `class Collectors{static Object toCollection(Object supplier){return supplier;}}` + ordinary, false},
		{"source_local_type", `import java.util.stream.Collectors;class Use{Object run(){class Collectors{static Object toCollection(Object supplier){return supplier;}}return Collectors.toCollection(null);}}`, false},
		{"value", `import java.util.stream.Collectors;class Decoy{Object toCollection(Object supplier){return supplier;}}class Use{Object run(Decoy Collectors){return Collectors.toCollection(null);}}`, false},
		{"class_binder", `import java.util.stream.Collectors;class Use<Collectors>{Object run(){return Collectors.toCollection(null);}}`, false},
		{"method_binder", `import java.util.stream.Collectors;class Use{<Collectors>Object run(){return Collectors.toCollection(null);}}`, false},
		{"qualified_value_root", `class Use{Object run(Object java){return java.util.stream.Collectors.toCollection(null);}}`, false},
		{"qualified_binder_root", `class Use<java>{Object run(){return java.util.stream.Collectors.toCollection(null);}}`, false},
		{"unknown_on_demand_competitor", `import java.util.stream.*;import unrelated.*;` + ordinary, false},
		{"conflicting_import_canonical_last", `import unrelated.Collectors;import java.util.stream.Collectors;` + ordinary, false},
		{"conflicting_import_canonical_first", `import java.util.stream.Collectors;import unrelated.Collectors;` + ordinary, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			h, ctx, node := deferredCollectorFactoryNamespaceUse(t, row.source)
			name, known := deferredToCollectionFactoryName(node, h.File.Source, ctx)
			if known != row.want || known && name != "toCollection" {
				t.Fatalf("original factory namespace: name=%q known=%v want=%v", name, known, row.want)
			}
		})
	}

	t.Run("foreign_and_mutated_authority", func(t *testing.T) {
		source := `import java.util.stream.Collectors;` + ordinary
		h, ctx, node := deferredCollectorFactoryNamespaceUse(t, source)
		positive := func() {
			t.Helper()
			if name, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); !known || name != "toCollection" {
				t.Fatal("same original graph did not establish its positive premise")
			}
		}
		positive()
		registered := symbol.GlobalScope
		foreign, other, foreignNode := deferredCollectorFactoryNamespaceUse(t, source)
		if name, known := deferredToCollectionFactoryName(foreignNode, foreign.File.Source, other); !known || name != "toCollection" {
			t.Fatal("foreign graph did not establish its own positive premise")
		}
		symbol.GlobalScope = registered
		positive()
		wrong := ctx.Clone()
		wrong.localScope = ctx.currentClass.FindMethodByName("other", nil)
		for _, bad := range []struct {
			ctx    Ctx
			source []byte
		}{{other, h.File.Source}, {wrong, h.File.Source}, {ctx, append([]byte(nil), h.File.Source...)}, {ctx, nil}} {
			if _, known := deferredToCollectionFactoryName(node, bad.source, bad.ctx); known {
				t.Fatal("foreign executable or copied source donated canonical factory authority")
			}
		}
		if _, known := deferredToCollectionFactoryName(foreignNode, h.File.Source, ctx); known {
			t.Fatal("foreign occurrence donated canonical factory authority")
		}
		originalImport := ctx.currentFile.Imports["Collectors"]
		ctx.currentFile.Imports["Collectors"] = "unrelated"
		if _, known := deferredToCollectionFactoryName(node, h.File.Source, ctx); known {
			t.Fatal("mutated import metadata replaced the original namespace")
		}
		ctx.currentFile.Imports["Collectors"] = originalImport
		positive()
		missing, missingCtx, missingNode := deferredCollectorFactoryNamespaceUse(t, ordinary)
		missingCtx.currentFile.Imports["Collectors"] = "java.util.stream"
		if _, known := deferredToCollectionFactoryName(missingNode, missing.File.Source, missingCtx); known {
			t.Fatal("donated canonical import concealed the missing original import")
		}
		symbol.GlobalScope = registered
		positive()
	})
}
