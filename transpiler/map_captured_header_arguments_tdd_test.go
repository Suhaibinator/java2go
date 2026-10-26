package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

// Source-only prepared admission control. All three T binders come from actual
// parsed Java declarations and the real local capture planner.
func TestMapReferenceParsedCapturedHeaderArguments(t *testing.T) {
	source := `class CaptureOuter<T>{<T>void capture(T methodValue){abstract class Local<T> extends java.util.AbstractMap<String,T> implements java.util.function.Function<String,T>{Object retained(){return methodValue;}}}}`
	helper := setupParseHelper(t, source)
	owner := helper.File.Symbols.FindClassScope("CaptureOuter")
	var method *symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate.OriginalName == "capture" {
			method = candidate
		}
	}
	if method == nil {
		t.Fatal("missing parsed method")
	}
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = method
	methodNode := findNode(helper.File.Ast, "method_declaration")
	localNode := findNode(methodNode, "class_declaration")
	if localNode == nil {
		t.Fatal("missing local declaration")
	}
	captures := collectCapturedLocals(localNode.ChildByFieldName("body"), []byte(source), ctx)
	plan := planLocalClassTypeParameters(localNode, captures, []byte(source), ctx)
	if len(plan.carried) != 3 || len(plan.declared) != 1 {
		t.Fatalf("actual carried/declaration binders=%v/%v", symbol.GoTypeParamNames(plan.carried), symbol.GoTypeParamNames(plan.declared))
	}
	identities := map[*symbol.TypeParamDeclaration]bool{}
	names := map[string]bool{}
	for _, parameter := range plan.carried {
		if parameter.Name != "T" || parameter.Declaration == nil || identities[parameter.Declaration] || names[parameter.EmittedName()] {
			t.Fatal("binder identity collapsed")
		}
		identities[parameter.Declaration] = true
		names[parameter.EmittedName()] = true
	}
	scope := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Local", Name: "Local", DeclarationNode: localNode}, Enclosing: owner, IsInner: true, IsAbstract: true, TypeParameters: plan.carried, DeclaredTypeParameters: plan.declared, Superclass: "java.util.AbstractMap<String,T>", ImplementedInterfaces: []string{"java.util.function.Function<String,T>"}}
	actual := []string{"java.lang.String", "java.lang.Long", "java.lang.Integer"}
	slots, ok := mapReferenceScopeArguments(scope, actual, "java.util.Map", ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
	if !ok || len(slots) != 2 || slots[0] != "java.lang.String" || slots[1] != "java.lang.Integer" {
		t.Fatalf("local header mapped to captured owner/method binder: %v,%v", slots, ok)
	}
	function := sourceFunctionContract(scope, ctx)
	if len(function) != 2 || function[0] != "java.lang.String" || function[1] != plan.declared[0].EmittedName() {
		t.Fatalf("Function header declaration lost: %v", function)
	}
	t.Run("source_declared_arity", func(t *testing.T) {
		slots, ok := mapReferenceScopeArguments(scope, []string{"java.lang.Integer"}, "java.util.Map", ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
		if !ok || len(slots) != 2 || slots[0] != "java.lang.String" || slots[1] != "java.lang.Integer" {
			t.Fatalf("Local<Integer> source argument bound to captured prefix instead of declared local T: %v,%v", slots, ok)
		}
	})
	t.Run("physical_subobject_self_role", func(t *testing.T) {
		physicalCtx := classScopeCtx(scope, ctx)
		self := classSubobjectDeclarationPointerType(scope, physicalCtx)
		var output bytes.Buffer
		if err := format.Node(&output, token.NewFileSet(), self); err != nil {
			t.Fatal(err)
		}
		want := "*Local[" + strings.Join(scope.GoTypeParameterNames(), ", ") + "]"
		if output.String() != want {
			t.Fatalf("already emitted self binders re-resolved by source spelling: got %s want %s", output.String(), want)
		}
	})
}
