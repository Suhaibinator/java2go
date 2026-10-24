package transpiler

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestCallableBodyClassCastExistingRefusals(t *testing.T) {
	for _, source := range []string{
		`class Class<X>{} class Owner<T>{Owner(Object v){Object c=(Class<T>)v;}}`,
		`class Holder<X>{} class Owner<T>{Owner(Object v){Object c=(Holder<T>)v;}}`,
		`class Holder<X>{} class Owner<T>{Owner(){Object c=(Class<T>)new Holder<T>();}}`,
		`class Owner<T>{Owner(boolean b,Object v){Object c=b?(Class<T>)v:null;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			for _, method := range owner.Methods {
				if method.Constructor && constructorBodySupportsCallableErasure(owner, method, owner.TypeParameters[0].Declaration, helper.Ctx) {
					t.Fatal("unsupported constructor body admitted")
				}
			}
		})
	}
}

func callableClassCastFixture(t *testing.T, source string) (Ctx, *symbol.ClassScope, *symbol.Definition, *sitter.Node) {
	t.Helper()
	helper := setupParseHelper(t, source)
	owner := helper.File.Symbols.FindClassScope("Owner")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("missing owner")
	}
	for _, m := range owner.Methods {
		if m.Constructor || m.OriginalName == "read" {
			cast := findNode(m.DeclarationNode, "cast_expression")
			if cast != nil {
				return helper.Ctx, owner, m, cast
			}
		}
	}
	t.Fatal("missing cast")
	return Ctx{}, nil, nil, nil
}

func TestCallableBodyClassCastOwnedConstructorAndMethod(t *testing.T) {
	for _, source := range []string{
		`class Owner<T>{Owner(Object value){Object c=(Class<? super T>)value;}}`,
		`class Owner<T>{Owner(Object value){Object c=(java.lang.Class<? extends T>[])value;}}`,
		`class Owner<T>{Object read(Object value){return (Class<T>)value;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			ctx, owner, m, cast := callableClassCastFixture(t, source)
			if !callableBodyCanonicalClassCast(owner, m, cast, owner.TypeParameters[0].Declaration, ctx) || !methodBodyTypeSyntaxSupportsCallableErasure(owner, m, owner.TypeParameters[0].Declaration, false, ctx) {
				t.Fatal("owned canonical cast type rejected")
			}
		})
	}
}

func TestCallableBodyClassCastRejectsForeignIdentity(t *testing.T) {
	const source = `class Owner<T>{Owner(Object value){Object c=(Class<? super T>)value;}}`
	for _, which := range []string{"foreignCast", "foreignConstructor", "foreignOwner", "detachedDefinition", "wrongDeclaration", "missingRoot", "badFormalType", "badFormalBindings"} {
		t.Run(which, func(t *testing.T) {
			ctx, owner, m, cast := callableClassCastFixture(t, source)
			requested := owner.TypeParameters[0].Declaration
			foreign := parsing.SourceFile{Name: "Foreign.java", Source: []byte(source)}
			if err := foreign.ParseAST(); err != nil {
				t.Fatal(err)
			}
			switch which {
			case "foreignCast":
				cast = findNode(foreign.Ast, "cast_expression")
			case "foreignConstructor":
				m.DeclarationNode = findNode(foreign.Ast, "constructor_declaration")
				cast = findNode(m.DeclarationNode, "cast_expression")
			case "foreignOwner":
				owner.Class.DeclarationNode = findNode(foreign.Ast, "class_declaration")
			case "detachedDefinition":
				copy := *m
				m = &copy
			case "wrongDeclaration":
				requested = &symbol.TypeParamDeclaration{SourceName: "T", GoName: "T"}
			case "missingRoot":
				owner.Class.DeclarationNode = nil
			case "badFormalType":
				m.Parameters[0].OriginalType = "String"
			case "badFormalBindings":
				m.Parameters[0].TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"T": {SourceName: "T", GoName: "T"}}
			}
			if callableBodyCanonicalClassCast(owner, m, cast, requested, ctx) {
				t.Fatal("unowned cast acquired certificate")
			}
		})
	}
}

func TestCallableBodyClassCastRejectsNestedReboundAndMethodShadow(t *testing.T) {
	for _, source := range []string{
		`class Owner<T>{class Inner<T>{Inner(Object value){Object c=(Class<T>)value;}}}`,
		`class Owner<T>{<T>Owner(Object value){Object c=(Class<T>)value;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			var method *symbol.Definition
			if len(owner.Subclasses) > 0 {
				method = owner.Subclasses[0].Methods[0]
				owner.Methods = append(owner.Methods, method)
			} else {
				method = owner.Methods[0]
			}
			cast := findNode(method.DeclarationNode, "cast_expression")
			if callableBodyCanonicalClassCast(owner, method, cast, owner.TypeParameters[0].Declaration, helper.Ctx) {
				t.Fatal("other declaration became owned class binder")
			}
		})
	}
}

func TestCallableBodyClassCastNamespaceAndOperandRefusals(t *testing.T) {
	for _, source := range []string{
		`class Class<X>{}class Owner<T>{Owner(Object value){Object c=(Class<T>)value;}}`,
		`class Owner<T,Class>{Owner(Object value){Object c=(Class<T>)value;}}`,
		`import vendor.Class;class Owner<T>{Owner(Object value){Object c=(Class<T>)value;}}`,
		`class Owner<T>{Owner(Object value){Object c=(Class<T,T>)value;}}`,
		`class Holder<X>{}class Owner<T>{Owner(Object value){Object c=(Holder<T>)value;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			for _, m := range owner.Methods {
				if m.Constructor {
					cast := findNode(m.DeclarationNode, "cast_expression")
					if callableBodyCanonicalClassCast(owner, m, cast, owner.TypeParameters[0].Declaration, helper.Ctx) {
						t.Fatal("noncanonical target admitted")
					}
				}
			}
		})
	}
	ctx, owner, m, cast := callableClassCastFixture(t, `class Holder<X>{}class Owner<T>{Owner(){Object c=(Class<T>)new Holder<T>();}}`)
	if !callableBodyCanonicalClassCast(owner, m, cast, owner.TypeParameters[0].Declaration, ctx) {
		t.Fatal("owned target premise missing")
	}
	if methodBodyTypeSyntaxSupportsCallableErasure(owner, m, owner.TypeParameters[0].Declaration, false, ctx) {
		t.Fatal("cast target skip swallowed operand")
	}
}

func TestCallableBodyClassCastCertificateIsPure(t *testing.T) {
	ctx, owner, m, cast := callableClassCastFixture(t, `class Owner<T>{Owner(Object value){Object c=(Class<T>)value;}}`)
	ctx.importAliases = map[string]string{"keep": "alias"}
	ctx.usedImports = map[string]bool{"keep": true}
	counter := 17
	ctx.anonClassCounter = &counter
	graph := symbol.GlobalScope
	name := m.Name
	binding := m.Parameters[0].TypeParameterBindings["T"]
	for i := 0; i < 25; i++ {
		if !callableBodyCanonicalClassCast(owner, m, cast, owner.TypeParameters[0].Declaration, ctx) {
			t.Fatal("repeated certificate changed")
		}
	}
	if graph != symbol.GlobalScope || counter != 17 || m.Name != name || m.Parameters[0].TypeParameterBindings["T"] != binding || !reflect.DeepEqual(ctx.importAliases, map[string]string{"keep": "alias"}) || !reflect.DeepEqual(ctx.usedImports, map[string]bool{"keep": true}) {
		t.Fatal("certificate mutated context/registry/bindings/allocator")
	}
}
