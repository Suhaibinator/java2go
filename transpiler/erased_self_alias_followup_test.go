package transpiler

import (
	"bytes"
	sitter "github.com/smacker/go-tree-sitter"
	"go/printer"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestErasedSelfAliasQualifiedNominalIsNotBinder(t *testing.T) {
	helper := setupParseHelper(t, `class Root {} class B {} class T {} class Owner<B extends Root,T extends B> {}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	ctx := classScopeCtx(owner, helper.Ctx)
	b, child := owner.TypeParameters[0], owner.TypeParameters[1]
	if b.Declaration == nil || child.Declaration == nil {
		t.Fatal("missing declarations")
	}
	for _, pair := range [][2]string{
		{"foreign." + child.EmittedName(), b.EmittedName()},
		{child.EmittedName(), "foreign." + b.EmittedName()},
		{"foreign.T", "foreign.B"},
		{child.EmittedName() + "<String>", b.EmittedName()},
		{child.EmittedName() + "[]", b.EmittedName()},
	} {
		if javaDependentTypeParameterAssignable(pair[0], pair[1], ctx) {
			t.Errorf("nominal/rank/parameterized type stolen by binder: %s -> %s", pair[0], pair[1])
		}
	}
}

func TestErasedSelfAliasLegacyShadowPrecedence(t *testing.T) {
	outerB := symbol.TypeParam{Name: "B", Bounds: []symbol.JavaType{{Original: "OuterRoot"}}}
	outerT := symbol.TypeParam{Name: "T", Bounds: []symbol.JavaType{{Original: "B"}}}
	methodB := symbol.TypeParam{Name: "B", Bounds: []symbol.JavaType{{Original: "InnerRoot"}}}
	ctx := Ctx{currentClass: &symbol.ClassScope{TypeParameters: []symbol.TypeParam{outerB, outerT}}, localScope: &symbol.Definition{TypeParameters: []symbol.TypeParam{methodB}}}
	if !javaDependentTypeParameterAssignable("T", "B", ctx) {
		t.Fatal("legacy nearest B fallback lost")
	}
	if javaDependentTypeParameterAssignable("B", "T", ctx) {
		t.Fatal("legacy reverse relation invented")
	}
	if javaDependentTypeParameterAssignable("B", "B", ctx) {
		t.Fatal("legacy self relation widened")
	}
	// A new implementation must retain the inner nil-declaration bounds rather
	// than taking referenceTypeParameterBindings' outer-first legacy B.
	innerX := symbol.TypeParam{Name: "X", Bounds: []symbol.JavaType{{Original: "T"}}}
	outerX := symbol.TypeParam{Name: "X", Bounds: []symbol.JavaType{{Original: "OuterRoot"}}}
	ctx.currentClass.TypeParameters = append(ctx.currentClass.TypeParameters, outerX)
	ctx.localScope.TypeParameters = append(ctx.localScope.TypeParameters, innerX)
	if !javaDependentTypeParameterAssignable("X", "T", ctx) {
		t.Fatal("inner legacy X bounds replaced by outer X")
	}
}

func TestErasedSelfAliasCapturedDeclaredBoundStillWins(t *testing.T) {
	helper := setupParseHelper(t, `class Root {} class Other {}
class Outer<B extends Root,T extends B> { <B extends Other> void method(T value,B unrelated) {} }`)
	owner := helper.File.Symbols.FindClassScope("Outer")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = owner.FindMethodByName("method", nil)
	outerB, child, methodB := owner.TypeParameters[0], owner.TypeParameters[1], ctx.localScope.TypeParameters[0]
	if child.Bounds[0].TypeParameterBindings[outerB.Name] != outerB.Declaration {
		t.Fatal("control did not capture outer bound identity")
	}
	if !javaDependentTypeParameterAssignable(child.EmittedName(), outerB.EmittedName(), ctx) {
		t.Fatal("captured outer bound invisible")
	}
	if javaDependentTypeParameterAssignable(child.EmittedName(), methodB.EmittedName(), ctx) {
		t.Fatal("captured bound rebound to method B")
	}
}

func TestErasedSelfAliasFieldReadOrderJDK21(t *testing.T) {
	runErasedSelfAliasFullProject(t, "field-read-order", "fieldread.Main")
}

func TestErasedSelfAliasExplicitFieldResultUsesNominalOrigin(t *testing.T) {
	source, err := os.ReadFile("testdata/erased_self_alias/full-control/src/main/java/erasedalias/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(source))
	owner := helper.File.Symbols.FindClassScope("Main")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = owner.FindMethodByName("main", nil)
	ctx.executionContextName = "callerExecution"
	tokenScope := helper.File.Symbols.FindClassScope("Token")
	found := false
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "field_access" && n.Content(source) == "box.get().id" {
			found = true
			obj := n.ChildByFieldName("object")
			target, res := resolvedInstanceInvocation(obj, ctx, source)
			view, erasure, admitted := directOwnerMethodResultView(obj, ctx, source)
			if !admitted || target == nil || res == nil || res.def.DirectTypeParameter == nil || view != "Token" || erasure != "Object" {
				t.Fatalf("source result physical proof missing: %s %s %v", view, erasure, admitted)
			}
			var origin inferredJavaTypeOrigin
			recordMethodTypeOrigin([]*inferredJavaTypeOrigin{&origin}, res, target, obj, ctx, source, view)
			if origin.unresolved || origin.parameter != nil || origin.nominalScope != tokenScope {
				t.Fatal("nominal Token argument rebound to Box.Token")
			}
			expr := ParseExpr(n, source, ctx)
			var output bytes.Buffer
			if err := printer.Fprint(&output, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			rendered := output.String()
			if strings.Count(rendered, "ObjectView") != 1 || strings.Count(rendered, "getJava2goExecution") != 1 || !strings.HasSuffix(rendered, ".id") {
				t.Fatalf("getter not projected exactly once before field selection: %s", rendered)
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(helper.File.Ast)
	if !found {
		t.Fatal("full immutable consumer missing")
	}
}

func TestErasedSelfAliasShadowedCallableFormalUsesAllocatedIdentity(t *testing.T) {
	helper := setupParseHelper(t, `class Root {} class Other {}
 class Outer<B extends Root,T extends B> { <B extends Other,U extends B> B widen(U value) {return value;} }`)
	owner := helper.File.Symbols.FindClassScope("Outer")
	ctx := classScopeCtx(owner, helper.Ctx)
	method := owner.FindMethodByName("widen", nil)
	ctx.localScope = method
	formal := qualifyDeclaredReferenceType(definitionDeclaredJavaType(method), ctx)
	if formal != method.TypeParameters[0].EmittedName() {
		t.Fatalf("source B formal lost its method declaration: %s", formal)
	}
	actual := methodParameterReferenceType(method.Parameters[0], method, owner, ctx)
	if !javaDependentTypeParameterAssignable(actual, formal, ctx) {
		t.Fatal("method bound cannot widen to its own allocated formal")
	}
	if javaDependentTypeParameterAssignable(owner.TypeParameters[1].EmittedName(), formal, ctx) {
		t.Fatal("outer captured bound stolen by source method B")
	}
}

func TestErasedSelfAliasSourceReturnRootKeepsMethodDeclaration(t *testing.T) {
	helper := setupParseHelper(t, `class Root {}
 class Outer<B extends Root> { <B extends Root,U extends B> B related(U value) {return value;} }`)
	owner := helper.File.Symbols.FindClassScope("Outer")
	ctx := classScopeCtx(owner, helper.Ctx)
	method := owner.FindMethodByName("related", nil)
	ctx.localScope = method
	ret := findNode(method.DeclarationNode, "return_statement")
	ctx.expectedType = method.OriginalType
	ctx.expectedTypeRoot = ret.NamedChild(0)
	actual := methodParameterReferenceType(method.Parameters[0], method, owner, ctx)
	if !javaDependentTypeParameterAssignable(actual, ctx.expectedType, ctx) {
		t.Fatal("raw source return B lost its exact method declaration")
	}
	if javaDependentTypeParameterAssignable(method.TypeParameters[0].EmittedName(), ctx.expectedType, ctx) {
		t.Fatal("same method return slot widened")
	}
}

func TestErasedSelfAliasShadowedMethodReturnJDK21(t *testing.T) {
	runErasedSelfAliasFullProject(t, "method-return", "returnslot.Main")
}
