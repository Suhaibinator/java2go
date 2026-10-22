package transpiler

import (
	"crypto/sha256"
	"fmt"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestJDK21SystematicNegativeInvocationFullPipeline(t *testing.T) {
	for _, r := range []struct{ name, setup, clazz, method, actual, formal string }{
		{name: "key", actual: "java.util.HashMap<java.lang.String,java.lang.Integer>", formal: "java.util.Map<java.lang.Integer,java.lang.Integer>"},
		{name: "value", actual: "java.util.HashMap<java.lang.String,java.lang.Integer>", formal: "java.util.Map<java.lang.String,java.lang.Number>"},
		{name: "nested-wildcard", actual: "java.util.HashMap<java.lang.reflect.TypeVariable<?>,java.lang.reflect.Type>", formal: "java.util.Map<java.lang.reflect.TypeVariable<? extends java.lang.Class<?>>,java.lang.reflect.Type>"},
		{name: "source-homonym", setup: "class HashMap<K,V>{}", actual: "HashMap<java.lang.String,java.lang.Integer>", formal: "java.util.Map<java.lang.String,java.lang.Integer>"},
		{name: "method-binder", method: "<HashMap>", actual: "HashMap", formal: "java.util.Map<java.lang.String,java.lang.Integer>"},
		{name: "class-binder", clazz: "<Map>", actual: "Map", formal: "java.util.Map<java.lang.String,java.lang.Integer>"},
		{name: "array-scalar", actual: "java.util.HashMap<java.lang.String,java.lang.Integer>[]", formal: "java.util.Map<java.lang.String,java.lang.Integer>"},
	} {
		t.Run(r.name, func(t *testing.T) {
			source := "package p; " + r.setup + " public class Use" + r.clazz + "{void take(" + r.formal + " value){} " + r.method + " void test(" + r.actual + " value){take(value);}}"
			ctx := systematicNegativeFullPipelineContext(t, source)
			owner := ctx.currentFile.FindClassScope("Use")
			ctx.currentClass = owner
			var caller, target *symbol.Definition
			for _, d := range owner.Methods {
				if d.OriginalName == "test" {
					caller = d
				}
				if d.OriginalName == "take" {
					target = d
				}
			}
			if caller == nil || target == nil {
				t.Fatal("actual method declarations missing")
			}
			ctx.localScope = caller
			var call *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "method_invocation" {
					call = n
				}
				for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
					walk(c)
				}
			}
			walk(caller.DeclarationNode.ChildByFieldName("body"))
			if call == nil {
				t.Fatal("actual source call missing")
			}
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(source))); got != systematicNegativeSourcePins[r.name] {
				t.Fatalf("exact Java counter pin differs: %s", got)
			}
			args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(args[0], ctx, ctx.currentFile.Source, &actual)
			actual.javaType = typ
			formals := methodFormalOrigins(target, owner, nil, ctx)
			formal := formals[0]
			cost, exact, conversion, handled := declaredOriginConversion(actual, formal, ctx)
			t.Logf("registeredOwner=%p ownerDefinition=%p caller=%p target=%p sourceDefinition=%p actualKnown=%t actual=%s actualParameter=%p formal=%s formalParameter=%p conversion=(%d,%t,%t,%t) callerTypeParams=%+v classOwnTypeParams=%+v actualDefinitionBindings=%v", owner, owner.Class, caller, target, caller.Parameters[0], known, gsonParentDiagnosticOrigin(actual, nil), actual.parameter, gsonParentDiagnosticOrigin(formal, nil), formal.parameter, cost, exact, conversion, handled, caller.TypeParameters, owner.OwnTypeParameters(), caller.Parameters[0].TypeParameterBindings)
			selected := findBestMethodInHierarchy(owner, "take", call.ChildByFieldName("arguments"), true, true, ctx, ctx.currentFile.Source)
			if selected == nil {
				t.Log("selected=nil")
			} else {
				t.Logf("selected=%p owner=%p", selected.def, selected.owner)
			}
			score, applies := scoreMethodCandidate(target, owner, methodCandidateTypeParameterNames(owner, target), nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments")), ctx, ctx.currentFile.Source)
			t.Logf("score=%+v applies=%t", score, applies)
			if applies {
				t.Error("JDK-invalid invocation admitted")
			}
		})
	}
}

func systematicNegativeFullPipelineContext(t *testing.T, source string) Ctx {
	t.Helper()
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
	file := parsing.SourceFile{Name: "p/Use.java", Source: []byte(source)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	if file.Ast == nil || file.Ast.HasError() {
		t.Fatal("exact Java parse failed")
	}
	symbol.AddSymbolsToPackage(file.ParseSymbols())
	ResolveFiles([]parsing.SourceFile{file})
	owner := findQualifiedSourceClass("p.Use")
	if owner == nil {
		t.Fatal("registered owner absent after actual resolution")
	}
	f := findFileScopeForClassScope(owner)
	if f == nil {
		t.Fatal("registered source file absent")
	}
	return Ctx{currentFile: f, currentClass: owner, className: owner.Class.Name, importAliases: map[string]string{}, usedImports: map[string]bool{}}
}

var systematicNegativeSourcePins = map[string]string{
	"array-scalar":    "51f6deb55f2dce267819aae105cecb4008909f5720d89709a68be7b9d1a791e1",
	"class-binder":    "cb059a0464a00bd92d27bd5d685be3eaf35c69f72f938ed9b67db23011faab7a",
	"key":             "3ea8c20bac9b027daac72fb60418952fc0fb9ac7d1f0dc1fbefdb81478230e44",
	"method-binder":   "93906ed1d3889660230472cb2bcb0db6f762f3387c55964c0a5160e871ed1a1c",
	"nested-wildcard": "5236586aa5188a7c331f526127b03d470282a342d36cbabd012c471e79979542",
	"source-homonym":  "2ee2309cf3ac9c206b4d2984a2738aad63b53f63c46d502b0bd729a28fe701de",
	"value":           "3cada03d8def1e5fd8f1301789ed67eaded1b143c12625f91a05228e7f74b6bd",
}
