package transpiler

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// This diagnostic loads every unchanged source in the author-frozen closure.
// It records declaration pointers before asserting the Java-valid source call.
func TestGsonImmutableListActualDeclarationOrigins(t *testing.T) {
	root := gsonGenericParentInputRoot(t)
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	diagnostics.mu.Lock()
	oldItems, oldStrict := diagnostics.items, diagnostics.strict
	diagnostics.items, diagnostics.strict = nil, false
	diagnostics.mu.Unlock()
	t.Cleanup(func() {
		symbol.GlobalScope = previous
		diagnostics.mu.Lock()
		diagnostics.items, diagnostics.strict = oldItems, oldStrict
		diagnostics.mu.Unlock()
	})
	var names []string
	for _, sub := range []string{"src/main/java", "dependencies/src"} {
		err := filepath.WalkDir(filepath.Join(root, sub), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && filepath.Ext(path) == ".java" {
				names = append(names, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(names)
	if len(names) != 114 {
		t.Fatalf("actual source closure has %d files; want frozen 114", len(names))
	}
	var files []parsing.SourceFile
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		switch filepath.Base(name) {
		case "Gson.java":
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != "dccb06a118de122605bead739f8c7b309242fabc210802c698649d298954a59f" {
				t.Fatal("Gson original byte pin mismatch")
			}
		case "GsonBuilder.java":
			if fmt.Sprintf("%x", sha256.Sum256(raw)) != "560e2ff54276d4bbc43607e559d2edae4c639f4dbf3b978180a44716ed4b9a49" {
				t.Fatal("GsonBuilder original byte pin mismatch")
			}
		}
		file := parsing.SourceFile{Name: name, Source: raw}
		if err := file.ParseAST(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if file.Ast == nil || file.Ast.HasError() {
			t.Fatalf("%s: actual valid JDK source has parser errors", name)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
		files = append(files, file)
	}
	ResolveFiles(files)
	owner := findQualifiedSourceClass("com.google.gson.GsonBuilder")
	if owner == nil {
		t.Fatal("actual GsonBuilder declaration unavailable")
	}
	var helper *symbol.Definition
	for _, def := range owner.Methods {
		if def.OriginalName == "newImmutableList" {
			if helper != nil {
				t.Fatal("unexpected ambiguous helper")
			}
			helper = def
		}
	}
	if helper == nil || len(helper.TypeParameters) != 1 || len(helper.Parameters) != 1 || !helper.IsStatic {
		t.Fatal("actual helper static generic declaration changed")
	}
	binder := helper.TypeParameters[0].Declaration
	if binder == nil || helper.Parameters[0].TypeParameterBindings["E"] != binder {
		t.Fatal("original helper E declaration identity unavailable")
	}
	describe := func(origin inferredJavaTypeOrigin) string { return gsonParentDiagnosticOrigin(origin, binder) }
	calls := 0
	for _, file := range files {
		if filepath.Base(file.Name) != "Gson.java" && filepath.Base(file.Name) != "GsonBuilder.java" {
			continue
		}
		scope := file.Symbols.BaseClass
		for _, method := range scope.Methods {
			if method.DeclarationNode == nil {
				continue
			}
			ctx := Ctx{currentClass: scope, currentFile: file.Symbols, className: scope.Class.Name, localScope: method, executionContextName: "__originProbeExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil {
					return
				}
				name := node.ChildByFieldName("name")
				if node.Type() == "method_invocation" && name != nil && name.Content(file.Source) == "newImmutableList" {
					calls++
					args := nodeutil.SemanticNamedChildrenOf(node.ChildByFieldName("arguments"))
					if len(args) != 1 {
						t.Fatal("original helper call arity changed")
					}
					var actual inferredJavaTypeOrigin
					actualType, known := inferExprJavaType(args[0], ctx, file.Source, &actual)
					actual.javaType = actualType
					formals := methodFormalOrigins(helper, owner, nil, ctx)
					cost, exact, applies, handled := declaredOriginConversion(actual, formals[0], ctx)
					candidates := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
					gatherMethodTypeOriginBindings(formals[0], actual, map[*symbol.TypeParamDeclaration]bool{binder: true}, candidates)
					gathered := []string{}
					for _, candidate := range candidates[binder] {
						gathered = append(gathered, describe(candidate))
					}
					score, scoreApplies := scoreMethodCandidate(helper, owner, methodCandidateTypeParameterNames(owner, helper), args, ctx, file.Source)
					resolution := resolveStaticImportedMethod(node, ctx, file.Source)
					selected := resolution.source != nil && resolution.source.def == helper && resolution.source.owner == owner
					boundBindings, boundApplies := methodInvocationConstraintBindings(helper, owner, args, false, ctx, file.Source)
					firstGuard := "different-or-other"
					if actual.unresolved || formals[0].unresolved {
						firstGuard = "top-level-unresolved"
					} else if !completeJavaTypeOrigin(actual) || !completeJavaTypeOrigin(formals[0]) {
						firstGuard = "incomplete-deferred"
					}
					t.Logf("file=%s line=%d call=%s callerOwner=%s callerMethod=%p helper=%p helperOwner=%p E=%p accessible=%t imports=%+v lexicalName=%t actualKnown=%t actual=%s formal=%s firstGuard=%s conversion=(%d,%t,%t,%t) gatheredE=%v boundSpellings=%v boundApplicable=%t score=%+v scoreApplicable=%t staticSelected=%t problem=%q intrinsic=%q", file.Name, node.StartPoint().Row+1, node.Content(file.Source), qualifiedSourceClassName(scope), method, helper, owner, binder, staticImportMethodAccessible(helper, owner, ctx), staticMethodImports(ctx), lexicalMethodNamePresent("newImmutableList", ctx), known, describe(actual), describe(formals[0]), firstGuard, cost, exact, applies, handled, gathered, boundBindings, boundApplies, score, scoreApplies, selected, resolution.problem, resolution.intrinsic)
					if !known || !completeJavaTypeOrigin(actual) || !completeJavaTypeOrigin(formals[0]) {
						t.Errorf("original call has unavailable declaration origin: %s", node.Content(file.Source))
					}
					if filepath.Base(file.Name) == "Gson.java" && !selected {
						t.Errorf("Java-valid actual imported call must select exact original static helper: %s", node.Content(file.Source))
					}
					if filepath.Base(file.Name) == "GsonBuilder.java" && !scoreApplies {
						t.Errorf("Java-valid lexical helper must be applicable: %s", node.Content(file.Source))
					}
				}
				for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
					walk(child)
				}
			}
			walk(method.DeclarationNode.ChildByFieldName("body"))
		}
	}
	if calls != 4 {
		t.Fatalf("probe observed %d actual helper calls; want 3 constructor + 1 lexical", calls)
	}
}

func gsonParentDiagnosticOrigin(o inferredJavaTypeOrigin, binder *symbol.TypeParamDeclaration) string {
	nominal, declaring, parameter := "", "", ""
	if o.nominalScope != nil {
		nominal = qualifiedSourceClassName(o.nominalScope)
	}
	if o.declaringOwner != nil {
		declaring = qualifiedSourceClassName(o.declaringOwner)
	}
	if o.parameter != nil {
		parameter = fmt.Sprintf("%s@%p", o.parameter.SourceName, o.parameter)
		if o.parameter == binder {
			parameter = "METHOD_E:" + parameter
		}
	}
	args := make([]string, len(o.arguments))
	for i, arg := range o.arguments {
		args[i] = gsonParentDiagnosticOrigin(arg, binder)
	}
	return fmt.Sprintf("{java:%q declaring:%q source:%q external:%q parameter:%q rank:%d wildcard:%t unresolved:%t complete:%t args:%v}", o.javaType, declaring, nominal, o.externalName, parameter, o.arrayRank, o.wildcard, o.unresolved, completeJavaTypeOrigin(o), args)
}
