package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const lateSourceTwoGenericRegistriesJava = `import java.time.LocalDate;

class Owner<ClassValue> {
  private static <Result> Result required(Result value) { return value; }

  <FirstMethod> LocalDate first(ClassValue owner, FirstMethod captured, LocalDate value) {
    class First<FirstOwn> {
      ClassValue first = owner;
      FirstMethod second = captured;
      FirstOwn third;
      First(FirstOwn value) { third = value; }
      LocalDate read(LocalDate input) {
        ClassValue ownerUse = first;
        FirstMethod methodUse = second;
        FirstOwn ownUse = third;
        return required(input);
      }
    }
    return new First<String>("first").read(value);
  }

  <SecondMethod> LocalDate second(ClassValue owner, SecondMethod captured, LocalDate value) {
    class Second<SecondOwn> {
      ClassValue first = owner;
      SecondMethod second = captured;
      SecondOwn third;
      Second(SecondOwn value) { third = value; }
      LocalDate read(LocalDate input) {
        ClassValue ownerUse = first;
        SecondMethod methodUse = second;
        SecondOwn ownUse = third;
        return required(input);
      }
    }
    return new Second<String>("second").read(value);
  }
}
`

// The observation pass uses the same actual source producers as the accepted
// API fixture. It inspects produced declaration identities only; it never
// reconstructs a second block registry or donates a successful caller context.
func TestJDK21LateSourceOwnershipTwoGenericLocalRegistries(t *testing.T) {
	oldGraph, oldIndex, oldImports, oldFrame := symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = nil, nil, nil
	t.Cleanup(func() {
		symbol.GlobalScope, activeResolutionFiles, activeImportSourceInventory, activeLateSourceOwnership = oldGraph, oldIndex, oldImports, oldFrame
		setStrictMode(false)
		resetDiagnostics()
	})
	file := parsing.SourceFile{Name: "TwoGenericRegistries.java", Source: []byte(lateSourceTwoGenericRegistriesJava)}
	if err := file.ParseAST(); err != nil || file.Ast == nil || file.Ast.HasError() {
		t.Fatal("generic two-registry source parse failed", err)
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	ResolveFiles([]parsing.SourceFile{file})
	var hoisted []ast.Decl
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, genericFamilies: &genericFamilyAnalysis{}, callableSubclasses: &callableSubclassSourceInventory{}, runtimeLocationProfile: sourceRuntimeLocationClasspath, anonymousClasses: make(map[anonymousClassKey]*anonymousClassInfo), localClasses: make(map[string]*localClassInfo), hoistedDecls: &hoisted, anonClassCounter: new(int)}
	activeResolutionFiles = sourceOwnershipIndex(ctx)
	activeImportSourceInventory = resolvedSourceInventory(ctx)
	frame := newLateSourceOwnershipFrame(file, ctx)
	ctx.lateSourceOwnership, activeLateSourceOwnership = frame, frame
	defer frame.close()
	if !frame.active {
		t.Fatal("generic two-registry actual source frame unavailable")
	}
	resetDiagnostics()
	setStrictMode(true)
	// Only the real source strict refusal is caught. Setup or runtime panic remains
	// a test failure and cannot match the semantic RED decoder.
	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(strictModeError); !ok {
					panic(r)
				}
			}
		}()
		ParseNode(file.Ast, file.Source, ctx)
	}()
	setStrictMode(false)
	seen := map[string]bool{}
	for owner, registration := range frame.late {
		name := owner.Class.OriginalName
		methodName, methodBinder, ownBinder := "", "", ""
		switch name {
		case "First":
			methodName, methodBinder, ownBinder = "first", "FirstMethod", "FirstOwn"
		case "Second":
			methodName, methodBinder, ownBinder = "second", "SecondMethod", "SecondOwn"
		default:
			t.Fatal("unexpected generic late caller", name)
		}
		if seen[name] || registration.local == nil || registration.local.scope != owner || registration.name != name || registration.producer != file.Symbols.BaseClass || registration.executable == nil || registration.executable.OriginalName != methodName {
			t.Fatal("generic caller did not retain its original source producer", name)
		}
		seen[name] = true
		var read *symbol.Definition
		for _, method := range owner.Methods {
			if method != nil && method.OriginalName == "read" {
				if read != nil {
					t.Fatal("duplicate actual read", name)
				}
				read = method
			}
		}
		query := ctx.Clone()
		query.currentClass, query.localScope = owner, read
		if read == nil || findFileScopeForClassScope(owner, query) != nil || len(inScopeTypeParameters(query)) == 0 || len(owner.TypeParameters) != 3 {
			t.Fatal("ordinary or binder-free route could satisfy generic source control", name)
		}
		for _, origin := range []struct {
			name       string
			parameters []symbol.TypeParam
			class      *symbol.ClassScope
			method     *symbol.Definition
		}{
			{"ClassValue", registration.producer.OwnTypeParameters(), registration.producer, nil},
			{methodBinder, registration.executable.TypeParameters, registration.producer, registration.executable},
			{ownBinder, owner.OwnTypeParameters(), owner, nil},
		} {
			var declaration *symbol.TypeParamDeclaration
			for _, parameter := range origin.parameters {
				if parameter.Name == origin.name {
					if declaration != nil {
						t.Fatal("duplicate source binder slot", name, origin.name)
					}
					declaration = parameter.Declaration
				}
			}
			anchor, present := frame.binders[declaration]
			if declaration == nil || visibleTypeParameterDeclarationForJavaType(origin.name, query) != declaration || !present || anchor.class != origin.class || anchor.method != origin.method || anchor.node == nil || !threadSourceNodeWithin(anchor.node, frame.root) {
				t.Fatal("generic caller lost an exact source binder identity", name, origin.name)
			}
		}
		if visibleTypeParameterDeclarationForJavaType("LocalDate", query) != nil {
			t.Fatal("generic caller shadows original LocalDate", name)
		}
		withoutLate := query.Clone()
		withoutLate.lateSourceOwnership = nil
		if jdk21NominalDeclaringUseOwned(withoutLate, "LocalDate") {
			t.Fatal("generic nominal ownership bypassed the late source branch", name)
		}
	}
	if len(seen) != 2 || !seen["First"] || !seen["Second"] {
		t.Fatal("both actual generic caller declarations must be observed")
	}
	// Run the actual source converter independently with a fresh producer frame.
	// This is the semantic observation; structural assertions above are setup.
	frame.close()
	activeLateSourceOwnership = nil
	resetDiagnostics()
	setStrictMode(true)
	// Reparse the exact immutable source so observation lowering cannot donate
	// mutated graph or declaration state to the actual conversion observation.
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	activeResolutionFiles, activeImportSourceInventory = nil, nil
	actual := parsing.SourceFile{Name: file.Name, Source: []byte(lateSourceTwoGenericRegistriesJava)}
	if err := actual.ParseAST(); err != nil || actual.Ast == nil || actual.Ast.HasError() {
		t.Fatal("generic semantic source reparse failed", err)
	}
	symbol.AddSymbolsToPackage(actual.ParseSymbols())
	ResolveFiles([]parsing.SourceFile{actual})
	base := Ctx{currentFile: actual.Symbols, currentClass: actual.Symbols.BaseClass, genericFamilies: &genericFamilyAnalysis{}, callableSubclasses: &callableSubclassSourceInventory{}, runtimeLocationProfile: sourceRuntimeLocationClasspath}
	node, err := convertFileNode(actual, base)
	setStrictMode(false)
	if err != nil {
		t.Logf("two-generic-local-registry actual conversion error: %s", err)
		t.Fatal("both genuine generic block-local callers must lower their original helper calls")
	}
	if node == nil || len(collectedDiagnostics()) != 0 {
		t.Fatal("generic two-registry source retained unsupported diagnostics")
	}
}
