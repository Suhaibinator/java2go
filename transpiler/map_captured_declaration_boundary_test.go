package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func parsedMapCapturedScope(t *testing.T, source, superclass string, interfaces []string) (*symbol.ClassScope, Ctx) {
	t.Helper()
	helper := setupParseHelper(t, source)
	owner := helper.File.Symbols.FindClassScope("Outer")
	var method *symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate.OriginalName == "capture" {
			method = candidate
		}
	}
	if method == nil {
		t.Fatal("missing actual executable declaration")
	}
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = method
	local := findNode(method.DeclarationNode, "class_declaration")
	if local == nil {
		t.Fatal("missing actual local declaration")
	}
	captures := collectCapturedLocals(local.ChildByFieldName("body"), []byte(source), ctx)
	plan := planLocalClassTypeParameters(local, captures, []byte(source), ctx)
	scope := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Local", Name: "Local", DeclarationNode: local}, Enclosing: owner, IsInner: true, IsAbstract: true, TypeParameters: plan.carried, DeclaredTypeParameters: plan.declared, Superclass: superclass, ImplementedInterfaces: interfaces}
	return scope, ctx
}

func parsedMapArguments(scope *symbol.ClassScope, args []string, ctx Ctx) ([]string, bool) {
	return mapReferenceScopeArguments(scope, args, "java.util.Map", ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
}

func TestMapCapturedDeclarationArityAndHeaderBoundaries(t *testing.T) {
	for _, control := range []struct {
		name, source, parent string
		interfaces           []string
	}{
		{"direct", `class Outer<T>{<T>void capture(T value){abstract class Local<T> implements java.util.Map<String,T>{Object retained(){return value;}}}}`, "", []string{"java.util.Map<String,T>"}},
		{"inherited", `abstract class Parent<V> implements java.util.Map<String,V>{} class Outer<T>{<T>void capture(T value){abstract class Local<T> extends Parent<T>{Object retained(){return value;}}}}`, "Parent<T>", nil},
		{"source_name_matches_carried_alias", `class Outer<T>{<T>void capture(T value){abstract class Local<T2> implements java.util.Map<String,T2>{Object retained(){return value;}}}}`, "", []string{"java.util.Map<String,T2>"}},
	} {
		t.Run(control.name, func(t *testing.T) {
			scope, ctx := parsedMapCapturedScope(t, control.source, control.parent, control.interfaces)
			if len(scope.TypeParameters) != 3 || len(scope.OwnTypeParameters()) != 1 {
				t.Fatal("actual capture inventory changed")
			}
			for _, args := range [][]string{{"java.lang.Integer"}, {"java.lang.String", "java.lang.Long", "java.lang.Integer"}} {
				values, ok := parsedMapArguments(scope, args, ctx)
				if !ok || len(values) != 2 || values[0] != "java.lang.String" || values[1] != "java.lang.Integer" {
					t.Fatalf("declaration-owned source/physical arguments %v = %v,%v", args, values, ok)
				}
			}
			values, ok := parsedMapArguments(scope, nil, ctx)
			if !ok || len(values) != 0 {
				t.Fatalf("raw own declaration lost raw Map contract: %v,%v", values, ok)
			}
		})
	}
}

func TestMapCapturedHiddenOnlyDeclarationIsNotRaw(t *testing.T) {
	scope, ctx := parsedMapCapturedScope(t, `class Outer<T>{<M>void capture(M value){abstract class Local implements java.util.Map<String,M>{Object retained(){return value;}}}}`, "", []string{"java.util.Map<String,M>"})
	if len(scope.OwnTypeParameters()) != 0 || len(scope.TypeParameters) != 2 {
		t.Fatal("actual hidden-only inventory changed")
	}
	values, ok := parsedMapArguments(scope, nil, ctx)
	if !ok || len(values) != 2 || values[0] != "java.lang.String" || values[1] != ctx.localScope.TypeParameters[0].EmittedName() {
		t.Fatalf("nongeneric local Map contract erased captured declaration: %v,%v", values, ok)
	}
}

func TestMapCapturedDeclarationRefusesMalformedLayouts(t *testing.T) {
	source := `class Outer<T>{<T>void capture(T value){abstract class Local<T> implements java.util.Map<String,T>{Object retained(){return value;}}}<T>void other(T unrelated){}}`
	for _, name := range []string{"partial_arity", "extra_arity", "empty_argument", "missing_identity", "duplicate_identity", "duplicate_alias", "wrong_own_identity", "another_executable", "missing_carried_context"} {
		t.Run(name, func(t *testing.T) {
			scope, ctx := parsedMapCapturedScope(t, source, "", []string{"java.util.Map<String,T>"})
			args := []string{"java.lang.Integer"}
			switch name {
			case "partial_arity":
				args = []string{"java.lang.String", "java.lang.Integer"}
			case "extra_arity":
				args = []string{"java.lang.String", "java.lang.Long", "java.lang.Integer", "java.lang.Integer"}
			case "empty_argument":
				args = []string{""}
			case "missing_identity":
				scope.TypeParameters[0].Declaration = nil
			case "duplicate_identity":
				scope.TypeParameters[1].Declaration = scope.TypeParameters[0].Declaration
			case "duplicate_alias":
				declaration := *scope.TypeParameters[1].Declaration
				declaration.GoName = scope.TypeParameters[0].EmittedName()
				scope.TypeParameters[1].Declaration = &declaration
			case "wrong_own_identity":
				scope.DeclaredTypeParameters = []symbol.TypeParam{scope.TypeParameters[0]}
			case "another_executable":
				for _, method := range ctx.currentClass.Methods {
					if method.OriginalName == "other" {
						ctx.localScope = method
					}
				}
			case "missing_carried_context":
				ctx.localScope = nil
			}
			if values, ok := parsedMapArguments(scope, args, ctx); ok {
				t.Fatalf("malformed declaration admitted as Map: %v", values)
			}
		})
	}
}

func TestMapCapturedDeclarationNominalAndReferenceBoundaries(t *testing.T) {
	helper := setupParseHelper(t, `class Map<K,V>{} class Function<T,R>{} class T{} abstract class Nominal implements java.util.Map<String,T>{} class Bound<X extends java.util.Map<String,Integer>>{}`)
	for _, nominal := range []string{"Map<String,Integer>", "Function<String,Integer>", "Map[]", "java.util.Map<String,Integer>[]"} {
		if values, ok := mapReferenceArguments(nominal, "java.util.Map", helper.Ctx); ok {
			t.Fatalf("nominal/array lookalike admitted: %s = %v", nominal, values)
		}
	}
	values, ok := mapReferenceArguments("Nominal", "java.util.Map", helper.Ctx)
	if !ok || len(values) != 2 || values[1] != "T" {
		t.Fatalf("source nominal T captured a binder: %v,%v", values, ok)
	}
	bound := helper.File.Symbols.FindClassScope("Bound")
	ctx := classScopeCtx(bound, helper.Ctx)
	values, ok = mapReferenceArguments("X", "java.util.Map", ctx)
	if !ok || len(values) != 2 || values[1] != "java.lang.Integer" {
		t.Fatalf("actual declared Map bound lost: %v,%v", values, ok)
	}
	if _, ok := mapReferenceArguments("X[]", "java.util.Map", ctx); ok {
		t.Fatal("array of bound parameters became scalar Map")
	}
}
