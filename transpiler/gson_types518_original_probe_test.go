package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestGsonTypes518OriginalSourceSelection(t *testing.T) {
	root := gsonGenericParentInputRoot(t)
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
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
		t.Fatal("original complete114 missing")
	}
	var files []parsing.SourceFile
	for _, name := range names {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f := parsing.SourceFile{Name: name, Source: raw}
		if err := f.ParseAST(); err != nil {
			t.Fatal(err)
		}
		if f.Ast == nil || f.Ast.HasError() {
			t.Fatalf("original parser failed: %s", name)
		}
		symbol.AddSymbolsToPackage(f.ParseSymbols())
		files = append(files, f)
	}
	ResolveFiles(files)
	outer := findQualifiedSourceClass("com.google.gson.internal.GsonTypes")
	if outer == nil {
		t.Fatal("actual original outer declaration missing")
	}
	var inner *symbol.ClassScope
	for _, scope := range outer.Subclasses {
		if scope.Class.OriginalName == "ParameterizedTypeImpl" {
			inner = scope
		}
	}
	if inner == nil {
		t.Fatal("actual original constructor owner missing")
	}
	var ctor, target *symbol.Definition
	for _, def := range inner.Methods {
		if def.Constructor && len(def.Parameters) == 3 {
			ctor = def
		}
	}
	for _, def := range outer.Methods {
		if def.OriginalName == "requiresOwnerType" {
			target = def
		}
	}
	if ctor == nil || target == nil || !target.IsStatic || len(target.Parameters) != 1 {
		t.Fatal("actual original caller or target missing")
	}
	file := findFileScopeForClassScope(outer)
	if file == nil {
		t.Fatal("actual original file missing")
	}
	ctx := Ctx{currentClass: inner, currentFile: file, className: inner.Class.Name, localScope: ctor, executionContextName: "__original518Execution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" && n.StartPoint().Row+1 == 518 {
			if call != nil {
				t.Fatal("duplicate actual518")
			}
			call = n
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
			walk(child)
		}
	}
	walk(ctor.DeclarationNode)
	if call == nil || call.Content(file.Source) != "requiresOwnerType(rawType)" {
		t.Fatal("unchanged original518 call missing")
	}
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	if len(args) != 1 {
		t.Fatal("actual518 arity changed")
	}
	var actual inferredJavaTypeOrigin
	typ, known := inferExprJavaType(args[0], ctx, file.Source, &actual)
	actual.javaType = typ
	formals := methodFormalOrigins(target, outer, nil, ctx)
	if len(formals) != 1 {
		t.Fatal("actual source formal missing")
	}
	formal := formals[0]
	t.Logf("original owner=%p enclosing=%p outer=%p ctor=%p target=%p actual=%+v formal=%+v", inner, inner.Enclosing, outer, ctor, target, actual, formal)
	if inner.Enclosing != outer || !known || actual.externalName != "java.lang.Class" || actual.nominalScope != nil || len(actual.arguments) != 1 || !actual.arguments[0].wildcard || formal.externalName != "java.lang.reflect.Type" || formal.nominalScope != nil || formal.unresolved {
		t.Fatal("original declaration/origin premise failed; no conversion attribution")
	}
	_, ap := declaredArgumentProof(actual, actual.javaType, ctx, 0)
	_, fp := declaredArgumentProof(formal, formal.javaType, ctx, 0)
	relation := jdk21DeclaredBoundSubtype(actual, formal)
	_, project := projectJDK21GenericParentOrigin(actual, formal.externalName)
	raw := jdk21RawReferenceOriginAssignable(actual, formal)
	cost, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
	score, selectedApplicable := scoreMethodCandidate(target, outer, methodCandidateTypeParameterNames(outer, target), args, ctx, file.Source)
	selected := findEnclosingStaticMethod("requiresOwnerType", call.ChildByFieldName("arguments"), ctx, file.Source)
	t.Logf("proof actual=%v formal=%v subtype=%v genericProjection=%v raw=%v conversion=%d/%v/%v/%v score=%+v applies=%v selected=%+v", ap, fp, relation, project, raw, cost, exact, applies, handled, score, selectedApplicable, selected)
	if !ap || !fp || relation != declaredBoundYes {
		t.Fatal("existing certified declaration/argument/raw header proof unavailable; no new admission proposal")
	}
	if selected == nil || selected.def != target || selected.owner != outer {
		t.Fatal("JDK-valid unchanged original Class<?>→Type invocation must select exact source requiresOwnerType declaration")
	}
}
