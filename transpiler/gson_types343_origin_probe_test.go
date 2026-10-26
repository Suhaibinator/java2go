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

func TestGsonTypes343OriginalDeclarationSelection(t *testing.T) {
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
	owner := findQualifiedSourceClass("com.google.gson.internal.GsonTypes")
	if owner == nil {
		t.Fatal("actual Gson declaration missing")
	}
	file := findFileScopeForClassScope(owner)
	if file == nil {
		t.Fatal("actual Gson donor file missing")
	}

	var caller, want *symbol.Definition
	for _, def := range owner.Methods {
		if def.OriginalName != "resolve" {
			continue
		}
		switch len(def.OriginalParameterTypes()) {
		case 3:
			caller = def
		case 4:
			want = def
		}
	}
	if caller == nil || want == nil {
		t.Fatal("exact original resolve declarations missing")
	}
	ctx := Ctx{currentClass: owner, currentFile: file, className: owner.Class.Name, localScope: caller, executionContextName: "__gsonTypes343ActualExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" && node.StartPoint().Row+1 == 343 {
			if call != nil {
				t.Fatal("duplicate343")
			}
			call = node
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(caller.DeclarationNode.ChildByFieldName("body"))
	if call == nil || call.Content(file.Source) != "resolve(context, contextRawType, toResolve, new HashMap<TypeVariable<?>, Type>())" {
		t.Fatal("unchanged original343 call missing")
	}
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	if len(args) != 4 {
		t.Fatal("unchanged original343 arity differs")
	}
	var receiver inferredJavaTypeOrigin
	recordNominalClassTypeOrigin([]*inferredJavaTypeOrigin{&receiver}, owner)
	t.Logf("implicitReceiverDefinition=%p owner=%p file=%p caller=%p want=%p receiver=%s imports=%v", owner.Class, owner, file, caller, want, gsonParentDiagnosticOrigin(receiver, nil), file.Imports)
	actuals := make([]inferredJavaTypeOrigin, len(args))
	for i, arg := range args {
		typ, known := inferExprJavaType(arg, ctx, file.Source, &actuals[i])
		actuals[i].javaType = typ
		t.Logf("actual[%d] node=%p kind=%s text=%s known=%t origin=%s", i, arg, arg.Type(), arg.Content(file.Source), known, gsonParentDiagnosticOrigin(actuals[i], nil))
		if !known || !completeJavaTypeOrigin(actuals[i]) {
			t.Errorf("actual%d incomplete", i)
		}
	}
	count := 0
	for _, def := range owner.Methods {
		if def.OriginalName != "resolve" {
			continue
		}
		count++
		arity := methodInvocationArityApplicable(def, len(args))
		t.Logf("candidate=%p declaration=%p owner=%p parameterTypes=%v return=%s static=%t private=%t sameOwnerAccessible=true methodBinders=%v arity=%t", def, def.DeclarationNode, owner, def.OriginalParameterTypes(), def.OriginalType, def.IsStatic, def.IsPrivate, def.TypeParameters, arity)
		if !arity {
			continue
		}
		formals := methodFormalOrigins(def, owner, nil, ctx)
		first := -1
		methodParams := map[*symbol.TypeParamDeclaration]bool{}
		for _, p := range def.TypeParameters {
			methodParams[p.Declaration] = true
		}
		gathered := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
		for i, formal := range formals {
			cost, exact, applies, handled := declaredOriginConversion(actuals[i], formal, ctx)
			projected, proved := projectJDK21GenericParentOrigin(actuals[i], formal.externalName)
			gatherMethodTypeOriginBindings(formal, actuals[i], methodParams, gathered)
			t.Logf("candidate=%p arg%d parameterDefinition=%p formal=%s sameOrigin=%t conversion=(%d,%t,%t,%t) typedProject=%t projected=%s rawNativeAncestry=%t", def, i, def.Parameters[i], gsonParentDiagnosticOrigin(formal, nil), sameJavaTypeOrigin(actuals[i], formal), cost, exact, applies, handled, proved, gsonParentDiagnosticOrigin(projected, nil), jdk21RawReferenceOriginAssignable(actuals[i], formal))
			if !applies && first < 0 {
				first = i
			}
		}
		bindings, bounds := methodInvocationConstraintBindings(def, owner, args, false, ctx, file.Source)
		score, applies := scoreMethodCandidate(def, owner, methodCandidateTypeParameterNames(owner, def), args, ctx, file.Source)
		t.Logf("candidate=%p firstRejectedArg=%d score=%+v applies=%t spellingBindings=%v bounds=%t actualMethodParamCount=%d gathered=%v", def, first, score, applies, bindings, bounds, len(methodParams), gathered)
	}
	selected := findBestMethodInHierarchy(owner, "resolve", call.ChildByFieldName("arguments"), true, true, ctx, file.Source)
	if selected == nil {
		t.Log("selected=<nil>")
	} else {
		t.Logf("selected=%p owner=%p parameters=%v", selected.def, selected.owner, selected.def.OriginalParameterTypes())
	}
	t.Logf("actualOverloads=%d", count)
	if selected == nil || selected.def != want || selected.owner != owner {
		t.Error("Java-valid original343 must select exact original private static four-argument resolve declaration")
	}
}

func TestGsonTypes343ExistingAPIParentConversions(t *testing.T) {
	ctx := rawReferenceTestContext(t)
	origin := func(s string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: s}, ctx)
	}
	for _, pair := range [][2]string{
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "java.util.Map<java.lang.String,java.lang.Integer>"},
		{"java.util.HashMap<java.lang.reflect.TypeVariable<?>,java.lang.reflect.Type>", "java.util.Map<java.lang.reflect.TypeVariable<?>,java.lang.reflect.Type>"},
	} {
		t.Run(pair[0], func(t *testing.T) {
			a, f := origin(pair[0]), origin(pair[1])
			c, e, ok, h := declaredOriginConversion(a, f, ctx)
			p, proved := projectJDK21GenericParentOrigin(a, f.externalName)
			t.Logf("actual=%s formal=%s conversion=(%d,%t,%t,%t) project=%t projected=%s", gsonParentDiagnosticOrigin(a, nil), gsonParentDiagnosticOrigin(f, nil), c, e, ok, h, proved, gsonParentDiagnosticOrigin(p, nil))
			if !h || !ok || e || c < 1 {
				t.Error("actual JDK HashMap typed Map upcast must apply non-exactly")
			}
		})
	}
	for _, pair := range [][2]string{
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "java.util.Map<java.lang.Integer,java.lang.Integer>"},
		{"java.util.HashMap<java.lang.String,java.lang.Integer>", "java.util.Map<java.lang.String,java.lang.Number>"},
		{"java.util.HashMap<java.lang.String,java.lang.Integer>[]", "java.util.Map<java.lang.String,java.lang.Integer>"},
	} {
		if _, _, ok, _ := declaredOriginConversion(origin(pair[0]), origin(pair[1]), ctx); ok {
			t.Errorf("invariant/array refusal changed: %v", pair)
		}
	}
}
