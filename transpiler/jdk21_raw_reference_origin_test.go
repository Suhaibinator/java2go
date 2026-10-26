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

func TestGsonOriginalToJson569DeclarationSelection(t *testing.T) {
	root := gsonGenericParentInputRoot(t)
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	diagnostics.mu.Lock()
	items, strict := diagnostics.items, diagnostics.strict
	diagnostics.items, diagnostics.strict = nil, false
	diagnostics.mu.Unlock()
	t.Cleanup(func() {
		symbol.GlobalScope = previous
		diagnostics.mu.Lock()
		diagnostics.items, diagnostics.strict = items, strict
		diagnostics.mu.Unlock()
	})
	var names []string
	for _, sub := range []string{"src/main/java", "dependencies/src"} {
		err := filepath.WalkDir(filepath.Join(root, sub), func(path string, e os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !e.IsDir() && filepath.Ext(path) == ".java" {
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
		t.Fatalf("original114 closure count=%d", len(names))
	}
	var files []parsing.SourceFile
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if filepath.Base(name) == "Gson.java" && fmt.Sprintf("%x", sha256.Sum256(raw)) != "dccb06a118de122605bead739f8c7b309242fabc210802c698649d298954a59f" {
			t.Fatal("original Gson byte pin mismatch")
		}
		f := parsing.SourceFile{Name: name, Source: raw}
		if err := f.ParseAST(); err != nil {
			t.Fatal(err)
		}
		if f.Ast == nil || f.Ast.HasError() {
			t.Fatalf("original source parser failure %s", name)
		}
		symbol.AddSymbolsToPackage(f.ParseSymbols())
		files = append(files, f)
	}
	ResolveFiles(files)
	owner := findQualifiedSourceClass("com.google.gson.Gson")
	if owner == nil {
		t.Fatal("actual Gson declaration missing")
	}
	file := findFileScopeForClassScope(owner)
	if file == nil {
		t.Fatal("actual Gson donor file missing")
	}
	var caller, want, wrong *symbol.Definition
	for _, def := range owner.Methods {
		if def.OriginalName != "toJson" {
			continue
		}
		types := def.OriginalParameterTypes()
		if len(types) == 2 && types[0] == "Object" && types[1] == "Type" {
			caller = def
		}
		if len(types) == 3 && types[0] == "Object" && types[1] == "Type" {
			if types[2] == "Appendable" {
				want = def
			}
			if types[2] == "JsonWriter" {
				wrong = def
			}
		}
	}
	if caller == nil || want == nil || wrong == nil {
		t.Fatal("unchanged original overload declarations unavailable")
	}
	ctx := Ctx{currentClass: owner, currentFile: file, className: owner.Class.Name, localScope: caller, executionContextName: "__gson569ActualExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" && node.StartPoint().Row+1 == 569 {
			if call != nil {
				t.Fatal("unexpected multiple original line569 calls")
			}
			call = node
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(caller.DeclarationNode.ChildByFieldName("body"))
	if call == nil || call.Content(file.Source) != "toJson(src, typeOfSrc, writer)" {
		t.Fatal("exact original569 source call missing")
	}
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	if len(args) != 3 {
		t.Fatal("original call arity changed")
	}
	var receiver inferredJavaTypeOrigin
	recordNominalClassTypeOrigin([]*inferredJavaTypeOrigin{&receiver}, owner)
	t.Logf("implicitReceiverDefinition=%p classScope=%p receiverOrigin=%s caller=%p callerParams=%v selectedWant=%p wrongJsonWriter=%p file=%p sourceImportType=%q writerLocal=%+v", owner.Class, owner, gsonParentDiagnosticOrigin(receiver, nil), caller, caller.OriginalParameterTypes(), want, wrong, file, file.Imports["Type"], caller.FindVariable("writer"))
	actuals := make([]inferredJavaTypeOrigin, len(args))
	for i, arg := range args {
		typ, known := inferExprJavaType(arg, ctx, file.Source, &actuals[i])
		actuals[i].javaType = typ
		var definition *symbol.Definition
		if arg.Type() == "identifier" {
			definition = caller.FindVariable(arg.Content(file.Source))
		}
		t.Logf("actual[%d]=%s definition=%p known=%t origin=%s", i, arg.Content(file.Source), definition, known, gsonParentDiagnosticOrigin(actuals[i], nil))
		if !known || !completeJavaTypeOrigin(actuals[i]) {
			t.Errorf("actual original argument%d origin incomplete", i)
		}
	}
	count := 0
	for _, def := range owner.Methods {
		if def.OriginalName != "toJson" {
			continue
		}
		count++
		arity := methodInvocationArityApplicable(def, len(args))
		t.Logf("candidate=%p owner=%p params=%v return=%s static=%t private=%t public=%t sameOwnerAccessible=true methodBinders=%v arity=%t", def, owner, def.OriginalParameterTypes(), def.OriginalType, def.IsStatic, def.IsPrivate, staticImportMethodHasModifier(def, "public"), def.TypeParameters, arity)
		if !arity {
			continue
		}
		formals := methodFormalOrigins(def, owner, nil, ctx)
		firstRejection := -1
		for i, formal := range formals {
			cost, exact, applies, handled := declaredOriginConversion(actuals[i], formal, ctx)
			t.Logf("candidate=%p arg%d formal=%s conversion=(%d,%t,%t,%t)", def, i, gsonParentDiagnosticOrigin(formal, nil), cost, exact, applies, handled)
			if !applies && firstRejection < 0 {
				firstRejection = i
			}
		}
		bindings, bounds := methodInvocationConstraintBindings(def, owner, args, false, ctx, file.Source)
		score, applies := scoreMethodCandidate(def, owner, methodCandidateTypeParameterNames(owner, def), args, ctx, file.Source)
		t.Logf("candidate=%p score=%+v applicable=%t firstConversionRejection=%d boundSpellings=%v boundsApplicable=%t", def, score, applies, firstRejection, bindings, bounds)
	}
	selected := findBestMethodInHierarchy(owner, "toJson", call.ChildByFieldName("arguments"), true, true, ctx, file.Source)
	if selected == nil {
		t.Log("actual selected source method=<nil>")
	} else {
		t.Logf("actual selected=%p owner=%p params=%v", selected.def, selected.owner, selected.def.OriginalParameterTypes())
	}
	t.Logf("total source overloads=%d", count)
	// These are the exact original Java overloads, rather than inferred expected text.
	if selected == nil || selected.def != want || selected.owner != owner {
		t.Errorf("Java-valid original569 must select exact source Appendable overload")
	}
	if selected != nil && selected.def == wrong {
		t.Error("source JsonWriter overload must remain inapplicable to native StringBuilder")
	}
}

func rawReferenceTestContext(t *testing.T) Ctx {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"p/Use.java":                 `package p; class Value {} class Other {} public class Use<T> { <Appendable> void binder(Appendable value) {} }`,
		"foreign/StringBuilder.java": `package foreign; public class StringBuilder {}`,
		"foreign/Appendable.java":    `package foreign; public class Appendable {}`,
		"foreign/SourceAppend.java": `package foreign; public class SourceAppend implements java.lang.Appendable {
 public SourceAppend append(java.lang.CharSequence s){return this;}
 public SourceAppend append(java.lang.CharSequence s,int start,int end){return this;}
 public SourceAppend append(char c){return this;}
 }`,
	}, "p/Use.java")
	ctx.currentClass = ctx.currentFile.FindClassScope("Use")
	if ctx.currentClass == nil {
		t.Fatal("actual source Use owner missing")
	}
	return ctx
}

func TestJDK21RawNativeExistingAPIConversions(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	origin := func(name string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: name}, ctx)
	}
	for _, name := range []string{"java.lang.StringBuilder", "java.lang.StringBuffer", "java.io.StringWriter", "java.io.Writer", "java.io.PrintStream", "java.nio.CharBuffer"} {
		t.Run(name, func(t *testing.T) {
			actual, formal := origin(name), origin("java.lang.Appendable")
			if actual.unresolved || formal.unresolved {
				t.Fatal("actual canonical JDK origin unavailable")
			}
			cost, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
			t.Logf("%s->%s=(%d,%t,%t,%t)", name, formal.externalName, cost, exact, applies, handled)
			if !handled || !applies || exact || cost < 1 {
				t.Error("actual JDK interface upcast must apply at a non-exact reference cost")
			}
		})
	}
	for _, names := range [][2]string{
		{"java.lang.String", "java.lang.Appendable"}, {"java.lang.reflect.Type", "java.lang.Appendable"}, {"java.io.InputStream", "java.lang.Appendable"},
		{"foreign.StringBuilder", "java.lang.Appendable"}, {"java.lang.StringBuilder", "foreign.Appendable"},
		{"java.util.List<p.Other>", "java.util.Collection<p.Value>"},
	} {
		if _, _, applies, handled := declaredOriginConversion(origin(names[0]), origin(names[1]), ctx); applies || !handled {
			t.Errorf("unrelated/source/invariant %s->%s must remain rejected", names[0], names[1])
		}
	}
	if _, _, applies, handled := declaredOriginConversion(origin("foreign.SourceAppend"), origin("java.lang.Appendable"), ctx); !applies || !handled {
		t.Fatal("actual source interface implementation path changed")
	}
	for _, name := range []string{"java.lang.AbstractStringBuilder", "unknown.StringBuilder"} {
		actual := origin(name)
		if !actual.unresolved {
			t.Errorf("inaccessible/unknown %s must not gain a declaration origin", name)
		}
		if _, _, applies, handled := declaredOriginConversion(actual, origin("java.lang.Appendable"), ctx); applies || !handled {
			t.Error("unknown/inaccessible origin admitted")
		}
	}
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "binder" {
			actual := declaredJavaTypeOrigin(definitionDeclaredJavaType(method.Parameters[0]), ctx)
			if actual.parameter == nil {
				t.Fatal("original caller method binder identity missing")
			}
			if _, _, applies, _ := declaredOriginConversion(actual, origin("java.lang.Appendable"), ctx); applies {
				t.Fatal("same-spelled method binder must not become canonical native interface")
			}
		}
	}
}

func TestJDK21RawNativeArrayAndUnknownGuards(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	origin := func(name string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: name}, ctx)
	}
	actual, formal := origin("java.lang.StringBuilder[]"), origin("java.lang.Appendable[]")
	if _, exact, applies, handled := declaredOriginConversion(actual, formal, ctx); !handled || !applies || exact {
		t.Fatal("existing array component recursion must preserve valid interface covariance")
	}
	if _, _, applies, handled := declaredOriginConversion(actual, origin("java.lang.Appendable"), ctx); applies || !handled {
		t.Fatal("array value cannot become scalar interface")
	}
	value := origin("java.lang.StringBuilder")
	value.unresolved = true
	if _, _, applies, handled := declaredOriginConversion(value, origin("java.lang.Appendable"), ctx); applies || !handled {
		t.Fatal("unknown guard changed")
	}
}

func TestJDK21RawNativeGraphProofGuards(t *testing.T) {
	clone := func(input map[string]jdk21MemberOwner) map[string]jdk21MemberOwner {
		result := map[string]jdk21MemberOwner{}
		for name, record := range input {
			result[name] = record
		}
		return result
	}
	actual, formal := "java.lang.StringBuilder", "java.lang.Appendable"
	if !jdk21RawParentRecordsAssignable(actual, formal, jdk21MemberOwners, jdk21MemberOwners) {
		t.Fatal("actual certified direct/inherited parent diamond must be assignable")
	}
	for _, name := range []string{"missing actual", "missing sibling", "stale certificate", "empty certificate", "altered superclass", "altered interface"} {
		t.Run(name, func(t *testing.T) {
			graph := clone(jdk21MemberOwners)
			switch name {
			case "missing actual":
				delete(graph, actual)
			case "missing sibling":
				delete(graph, "java.lang.Comparable")
			case "stale certificate":
				record := graph[actual]
				record.certificate = "stale"
				graph[actual] = record
			case "empty certificate":
				record := graph[actual]
				record.certificate = ""
				graph[actual] = record
			case "altered superclass":
				record := graph[actual]
				record.superclass = "java.lang.String"
				graph[actual] = record
			case "altered interface":
				record := graph[actual]
				record.interfaces = []string{formal}
				graph[actual] = record
			}
			if jdk21RawParentRecordsAssignable(actual, formal, graph, jdk21MemberOwners) {
				t.Fatal("unknown/stale/tampered raw ancestry admitted")
			}
		})
	}
	// A hypothetical internally attested cycle still has no proof. These injected
	// records test the graph algorithm, not production native name admission.
	cyclic := clone(jdk21MemberOwners)
	record := cyclic[actual]
	record.superclass = actual
	cyclic[actual] = record
	if jdk21RawParentRecordsAssignable(actual, formal, cyclic, cyclic) {
		t.Fatal("cycle admitted despite a direct matching interface")
	}
	unknown := clone(jdk21MemberOwners)
	record = unknown[actual]
	record.superclass = "not.catalogued.Parent"
	unknown[actual] = record
	if jdk21RawParentRecordsAssignable(actual, formal, unknown, unknown) {
		t.Fatal("uncovered ancestry admitted")
	}
	ctx := rawReferenceTestContext(t)
	origin := func(text string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: text}, ctx)
	}
	for _, names := range [][2]string{{"java.util.List<p.Value>", "java.util.Collection<p.Value>"}, {"java.util.List<p.Other>", "java.util.Collection<p.Value>"}, {"java.lang.StringBuilder[]", "java.lang.Appendable[]"}, {"foreign.StringBuilder", "java.lang.Appendable"}, {"java.lang.StringBuilder", "foreign.Appendable"}, {"java.lang.AbstractStringBuilder", "java.lang.Appendable"}} {
		if jdk21RawReferenceOriginAssignable(origin(names[0]), origin(names[1])) {
			t.Errorf("typed/source/array/inaccessible endpoints must not use raw helper: %v", names)
		}
	}
}
