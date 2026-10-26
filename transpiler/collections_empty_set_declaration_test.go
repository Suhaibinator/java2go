package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
	"testing"
)

func TestCollectionsEmptySetCanonicalFactories(t *testing.T) {
	for _, source := range []string{
		`import java.util.Collections;import java.util.Set;class Probe {Set<String> run(){return Collections.emptySet();}}`,
		`import java.util.*;class Probe {Set<String> run(){return Collections.<String>emptySet();}}`,
		`class Probe {java.util.Set<int[]> run(){return java.util.Collections.<int[]>emptySet();}}`,
		`import static java.util.Collections.emptySet;class Probe {java.util.Set<String> run(){return emptySet();}}`,
		`import static java.util.Collections.*;class Probe {java.util.Set<String[]> run(){return emptySet();}}`,
	} {
		generated := renderGoFileFromJava(t, source)
		if !strings.Contains(generated, "stdjava.EmptySet()") {
			t.Errorf("canonical immutable factory missing for %s\n%s", source, generated)
		}
		if strings.Contains(generated, "stdjava.EmptySet[") || strings.Contains(generated, "stdjava.NewSet[") {
			t.Errorf("factory gained generic allocation/mutable storage\n%s", generated)
		}
	}
}

func TestCollectionsEmptySetSourceAndForeignGuards(t *testing.T) {
	for _, source := range []string{
		`class Collections {static Object emptySet(){return null;}}class Probe {Object run(){return Collections.emptySet();}}`,
		`import foreign.Collections;class Probe {Object run(){return Collections.emptySet();}}`,
		`class Service {Object emptySet(){return null;}}class Probe {Service Collections;Object run(){return Collections.emptySet();}}`,
		`class Probe<Collections,Set>{Collections owner;Set value;Collections run(){return owner;}}`,
		`class Probe {<Collections>Collections run(Collections value){return value;}}`,
	} {
		generated := renderGoFileFromJava(t, source)
		if strings.Contains(generated, "stdjava.EmptySet(") || strings.Contains(generated, "stdjava.EmptySet[") {
			t.Errorf("source/value/foreign declaration borrowed factory: %s\n%s", source, generated)
		}
	}
}

func TestCollectionsEmptySetDeclaredResultAndArrayRank(t *testing.T) {
	cases := []struct{ source, want string }{
		{`class Probe {java.util.Set<String> run(){return java.util.Collections.emptySet();}}`, "java.util.Set<java.lang.String>"},
		{`class Probe {java.util.Set<int[]> run(){return java.util.Collections.emptySet();}}`, "java.util.Set<int[]>"},
		{`class Probe {java.util.Set<String[]> run(){return java.util.Collections.emptySet();}}`, "java.util.Set<java.lang.String[]>"},
		{`class Object {}class Probe {java.lang.Object run(){return java.util.Collections.emptySet();}}`, "java.util.Set<java.lang.Object>"},
		{`class Set<T>{}class Probe {Set<String> run(){return java.util.Collections.emptySet();}}`, "java.util.Set<java.lang.Object>"},
	}
	for _, c := range cases {
		helper := setupParseHelper(t, c.source)
		owner := helper.File.Symbols.FindClassScope("Probe")
		var method *symbol.Definition
		for _, candidate := range owner.Methods {
			if candidate.OriginalName == "run" {
				method = candidate
			}
		}
		if method == nil {
			t.Fatal("parsed return owner missing")
		}
		ctx := classScopeCtx(owner, helper.Ctx)
		ctx.localScope = method
		invocation := findNode(method.DeclarationNode, "method_invocation")
		if invocation == nil {
			t.Fatal("parsed invocation missing")
		}
		ctx.expectedType = method.OriginalType
		ctx.expectedTypeRoot = invocation
		got, known := inferExprJavaType(invocation, ctx, helper.File.Source)
		if !known || got != c.want {
			t.Errorf("declared result got %q/%v want %q from %s", got, known, c.want, c.source)
		}
		ctx.expectedType = "java.util.Set<java.lang.String>[]"
		got, known = inferExprJavaType(invocation, ctx, helper.File.Source)
		if !known || got != "java.util.Set<java.lang.Object>" {
			t.Errorf("whole-array target leaked element: %q/%v", got, known)
		}
		ctx.expectedType = "java.util.Set<java.lang.String>"
		ctx.expectedTypeRoot = method.DeclarationNode
		got, known = inferExprJavaType(invocation, ctx, helper.File.Source)
		if !known || got != "java.util.Set<java.lang.Object>" {
			t.Errorf("enclosing expected root leaked element: %q/%v", got, known)
		}
	}
}

func TestCollectionsEmptySetExplicitArityRefusals(t *testing.T) {
	for _, source := range []string{
		`class Probe {Object run(){return java.util.Collections.emptySet(1);}}`,
		`class Probe {Object run(){return java.util.Collections.<int>emptySet();}}`,
		`class Probe {Object run(){return java.util.Collections.<void>emptySet();}}`,
		`class Probe {Object run(){return java.util.Collections.<String,Integer>emptySet();}}`,
		`class Probe {Object run(){return java.util.Collections.<?>emptySet();}}`,
	} {
		helper := setupParseHelper(t, source)
		node := findNode(helper.File.Ast, "method_invocation")
		if node == nil {
			t.Fatal("missing actual malformed invocation")
		}
		if value, mapped := tryStaticIntrinsicInvocation(node, "Collections", "emptySet", helper.File.Source, helper.Ctx); mapped || value != nil {
			t.Errorf("invalid invocation gained factory: %s", source)
		}
	}
}

func TestCollectionsEmptySetActualMethodBinderOwnership(t *testing.T) {
	source := `class Probe<T>{<T>java.util.Set<T> run(){return java.util.Collections.emptySet();}}`
	helper := setupParseHelper(t, source)
	owner := helper.File.Symbols.FindClassScope("Probe")
	method := owner.FindMethodByName("run", nil)
	if method == nil || len(method.TypeParameters) != 1 || method.TypeParameters[0].Declaration == nil || owner.TypeParameters[0].Declaration == method.TypeParameters[0].Declaration {
		t.Fatal("actual declarations collapsed")
	}
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = method
	node := findNode(method.DeclarationNode, "method_invocation")
	ctx.expectedType = method.OriginalType
	ctx.expectedTypeRoot = node
	want := "java.util.Set<" + method.TypeParameters[0].EmittedName() + ">"
	got, known := inferExprJavaType(node, ctx, helper.File.Source)
	if !known || got != want {
		t.Errorf("method return binder lost: %q/%v want %q", got, known, want)
	}
}
