package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const lateSourceOwnershipGenericJava = `
import java.time.LocalDate;
interface DateView { LocalDate read(LocalDate value); }
class Owner<ClassValue> {
  <MethodValue> DateView create(ClassValue owner, MethodValue captured) {
    return new DateView() {
      ClassValue first = owner;
      MethodValue second = captured;
      public LocalDate read(LocalDate value) { return value; }
      LocalDate other(LocalDate value) { return value; }
    };
  }
}`

// These positives use the public production conversion boundary rather than
// asking the new ownership leaf to manufacture a successful selection.
func TestJDK21LateSourceOwnershipStrictSourcePipeline(t *testing.T) {
	for _, local := range []bool{false, true} {
		name := "anonymous"
		source := strings.Replace(lateSourceOwnershipGenericJava, "return value;", "return required(value);", 1)
		source = strings.Replace(source, "class Owner<ClassValue> {", "class Owner<ClassValue> { private static <Result> Result required(Result value) { return value; }", 1)
		if local {
			name = "local"
			source = `import java.time.LocalDate;
class Owner<ClassValue> {
  private static <Result> Result required(Result value) { return value; }
  <MethodValue> Object create(ClassValue owner, MethodValue captured) {
    class Local<OwnValue> {
      ClassValue first = owner; MethodValue second = captured; OwnValue third;
      Local(OwnValue value) { third = value; }
      LocalDate read(LocalDate value) { return required(value); }
    }
    return new Local<String>("value");
  }
}`
		}
		t.Run(name, func(t *testing.T) {
			oldGraph, oldIndex, oldImports := symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
			activeResolutionFiles, activeImportSourceInventory = nil, nil
			t.Cleanup(func() {
				symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory = oldGraph, oldIndex, oldImports
				setStrictMode(false)
				resetDiagnostics()
			})
			file := parsing.SourceFile{Name: "StrictLate.java", Source: []byte(source)}
			if err := file.ParseAST(); err != nil || file.Ast.HasError() {
				t.Fatal("actual source parse failed", err)
			}
			symbol.AddSymbolsToPackage(file.ParseSymbols())
			ResolveFiles([]parsing.SourceFile{file})
			ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, genericFamilies: &genericFamilyAnalysis{}, callableSubclasses: &callableSubclassSourceInventory{}, runtimeLocationProfile: sourceRuntimeLocationClasspath}
			resetDiagnostics()
			setStrictMode(true)
			node, err := convertFileNode(file, ctx)
			setStrictMode(false)
			if err != nil || node == nil || len(collectedDiagnostics()) != 0 {
				t.Fatal("actual strict generic late-source pipeline rejected helper invocation", err, collectedDiagnostics())
			}
		})
	}
}
