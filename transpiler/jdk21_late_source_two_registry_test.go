package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const lateSourceTwoRegistriesJava = `import java.time.LocalDate;
class Owner {
  private static <Value> Value required(Value value) { return value; }
  LocalDate first(LocalDate value) {
    class First {
      LocalDate read(LocalDate input) { return required(input); }
    }
    return new First().read(value);
  }
  LocalDate second(LocalDate value) {
    class Second {
      LocalDate read(LocalDate input) { return required(input); }
    }
    return new Second().read(value);
  }
}`

// Both calls use the real source parser/resolver and converter in one render.
// Their different executable blocks own different local-class name maps.
func TestJDK21LateSourceOwnershipTwoLocalRegistries(t *testing.T) {
	oldGraph, oldIndex, oldImports, oldFrame := symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = nil, nil, nil
	t.Cleanup(func() {
		symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = oldGraph, oldIndex, oldImports, oldFrame
		setStrictMode(false)
		resetDiagnostics()
	})
	file := parsing.SourceFile{Name: "TwoLocalScopes.java", Source: []byte(lateSourceTwoRegistriesJava)}
	if err := file.ParseAST(); err != nil || file.Ast.HasError() {
		t.Fatal("two-local-registry valid source parse failed", err)
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	ResolveFiles([]parsing.SourceFile{file})
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, genericFamilies: &genericFamilyAnalysis{}, callableSubclasses: &callableSubclassSourceInventory{}, runtimeLocationProfile: sourceRuntimeLocationClasspath}
	resetDiagnostics()
	setStrictMode(true)
	node, err := convertFileNode(file, ctx)
	setStrictMode(false)
	if err != nil {
		t.Logf("two-local-registry actual conversion error: %s", err)
		t.Fatal("both genuine block-local registries must lower their original helper calls")
	}
	if node == nil || len(collectedDiagnostics()) != 0 {
		t.Fatal("two-local-registry source retained unsupported diagnostics")
	}
}
