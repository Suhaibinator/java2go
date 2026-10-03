package transpiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// Every ordinary field/method goes through this predicate. Rejecting a spelling
// outside the fixed generated-selector vocabulary must not scan the complete
// hierarchy and source AST; that work scales with the unrelated source graph.
func TestResolutionSelectorGuard_OrdinaryNamesAvoidHierarchyAnalysis(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	var source strings.Builder
	source.WriteString("package selector.fixture;\n")
	for index := 0; index < 24; index++ {
		fmt.Fprintf(&source, "class Leaf%d { int ordinaryField; int ordinaryMethod() { return ordinaryField + %d; } }\n", index, index)
	}
	source.WriteString(`class Factory { Object make() { return new Leaf0() {}; } }
class Metadata { Class<?> inspect() { return getClass(); } }
enum Tag { ONE }
`)
	file := parsing.SourceFile{Name: "SelectorGuard.java", Source: []byte(source.String())}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	leaf := file.Symbols.FindClassScope("Leaf23")
	if leaf == nil {
		t.Fatal("fixture lost ordinary leaf")
	}
	t.Run("ordinary-names", func(t *testing.T) {
		for _, name := range []string{"ordinaryField", "ordinaryMethod", "JavaObjectInfoSuffix"} {
			allocations := testing.AllocsPerRun(1, func() {
				if sourceReferenceReservedSelector(leaf, name, Ctx{}) {
					t.Fatalf("ordinary source selector %q was reserved", name)
				}
			})
			t.Logf("ordinary selector %s allocations=%.0f", name, allocations)
			if allocations > 8 {
				t.Errorf("ORDINARY_SELECTOR_GLOBAL_SCAN: %s allocated %.0f objects, want <=8 independent of source graph", name, allocations)
			}
		}
	})
	t.Run("reserved-selectors", func(t *testing.T) {
		target := file.Symbols.FindClassScope("Leaf0")
		if !classHasSyntheticSubclass(target, Ctx{}) {
			t.Fatal("anonymous subclass hierarchy precondition missing")
		}
		for _, name := range []string{"ObjectInfo", "JavaObjectInfo", "JavaDynamicTypeID", generatedDynamicTypeMethod, generatedObjectViewMethod} {
			if !sourceReferenceReservedSelector(target, name, Ctx{}) {
				t.Fatalf("generated identity selector %q was not reserved", name)
			}
		}
		if !sourceReferenceReservedSelector(file.Symbols.FindClassScope("Tag"), "JavaEnumMetadata", Ctx{}) {
			t.Fatal("enum metadata selector was not reserved")
		}
	})
}
