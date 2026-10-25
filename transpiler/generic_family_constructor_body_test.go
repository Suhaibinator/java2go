package transpiler

import (
	"go/ast"
	"os"
	"testing"
)

func TestGenericFamilyEnumConstructorBodyCertificate(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_enum_constructor_local/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(data))
	owner := helper.File.Symbols.FindClassScope("EnumBox")
	if owner == nil {
		t.Fatal("missing source owner")
	}
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.genericFamilies = &genericFamilyAnalysis{}
	for _, method := range owner.Methods {
		if method.Constructor {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil {
		t.Fatal("missing exact constructor declaration")
	}
	if constructorBodySupportsCallableErasure(owner, ctx.localScope, owner.OwnTypeParameters()[0].Declaration, ctx) {
		t.Fatal("legacy conservative constructor gate changed")
	}
	plan := canonicalGenericFamily(owner, ctx)
	if plan == nil || plan.constructorBodies[ctx.localScope] == nil {
		t.Fatal("complete family lacks declaration-owned body certificate")
	}
	if got := genericFamilyPhysicalJavaType("E", ctx); got != "E" {
		t.Fatalf("public constructor binder was erased: %s", got)
	}
	body := withGenericFamilyConstructorBody(ctx)
	if got := genericFamilyPhysicalJavaType("E", body); got != "java.lang.Enum" {
		t.Fatalf("body must use JVM bound: %s", got)
	}
	if got := genericFamilyPhysicalJavaType("E", body.Clone()); got != "java.lang.Enum" {
		t.Fatal("body clone lost exact certificate")
	}
	local := findNode(ctx.localScope.DeclarationNode.ChildByFieldName("body"), "local_variable_declaration")
	statement, ok := ParseStmt(local, data, body).(*ast.DeclStmt)
	if !ok {
		t.Fatal("initialized local kept concrete constructor type argument")
	}
	value := statement.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec)
	physical, ok := value.Type.(*ast.SelectorExpr)
	if !ok || physical.Sel.Name != "JavaEnum" {
		t.Fatalf("wrong local physical bound: %#v", value.Type)
	}
	foreign := body.Clone()
	foreign.currentClass = helper.File.Symbols.FindClassScope("Plain")
	if _, accepted := genericFamilyConstructorLocalErasure("E", foreign); accepted {
		t.Fatal("foreign owner borrowed body certificate")
	}
	fresh := body.Clone()
	fresh.genericFamilies = &genericFamilyAnalysis{}
	if _, accepted := genericFamilyConstructorLocalErasure("E", fresh); accepted {
		t.Fatal("fresh discovery borrowed an older certificate")
	}
}

func TestGenericFamilyEnumConstructorBodyRefusals(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"array_local", `class Owner<E extends Enum<E>>{Owner(E x){E[] values=null;}}`},
		{"array_cast", `class Owner<E extends Enum<E>>{Owner(Object x){Object values=(E[])x;}}`},
		{"nested_container", `class Owner<E extends Enum<E>>{Owner(E x){java.util.List<E> values=null;}}`},
		{"ternary", `class Owner<E extends Enum<E>>{Owner(E x,boolean b){Object value=b?x:null;}}`},
		{"delegating_cast", `class Owner<E extends Enum<E>>{Owner(Object x){this((E)x,0);}Owner(E x,int ignored){E local=x;}}`},
		{"local_class", `class Owner<E extends Enum<E>>{Owner(E x){class Local{E value;}E local=x;}}`},
		{"source_Enum", `class Enum<T>{}class Owner<E extends Enum<E>>{Owner(E x){E local=x;}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			if plan, err := planGenericFamily(owner, helper.Ctx); err == nil || plan != nil {
				t.Fatal("unsupported constructor body admitted as complete family")
			}
		})
	}
}

func TestGenericFamilyEnumConstructorBodyUncachedOwner(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_enum_constructor_local/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(data))
	owner := helper.File.Symbols.FindClassScope("EnumBox")
	ctx := classScopeCtx(owner, helper.Ctx)
	for _, m := range owner.Methods {
		if m.Constructor {
			ctx.localScope = m
		}
	}
	if ctx.genericFamilies != nil {
		t.Fatal("fixture must retain the uncached caller")
	}
	body := withGenericFamilyConstructorBody(ctx)
	if ctx.genericFamilies != nil || body.genericFamilies == nil || genericFamilyPhysicalJavaType("E", body) != "java.lang.Enum" {
		t.Fatal("body-local analysis escaped or lost its certificate")
	}
	if a, b := canonicalGenericFamily(owner, ctx), canonicalGenericFamily(owner, ctx); a == nil || b == nil || a == b {
		t.Fatal("uncached caller acquired a render cache")
	}
}

func TestGenericFamilyEnumConstructorVarOriginControls(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_enum_constructor_local/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(data))
	owner := helper.File.Symbols.FindClassScope("EnumBox")
	ctx := classScopeCtx(owner, helper.Ctx)
	for _, m := range owner.Methods {
		if m.Constructor {
			ctx.localScope = m
		}
	}
	ctx = withGenericFamilyConstructorBody(ctx)
	declaration := owner.OwnTypeParameters()[0].Declaration
	origin := inferredJavaTypeOrigin{parameter: declaration, javaType: "E"}
	if got, known := genericFamilyConstructorLocalType("var", "E", origin, ctx); !known || got != "E" {
		t.Fatal("proved inferred class binder lost its local view")
	}
	for _, name := range []string{"foreign_parameter", "nominal", "array", "wildcard", "incomplete"} {
		t.Run(name, func(t *testing.T) {
			other := origin
			switch name {
			case "foreign_parameter":
				other.parameter = helper.File.Symbols.FindClassScope("Plain").OwnTypeParameters()[0].Declaration
			case "nominal":
				other.parameter = nil
				other.nominalScope = owner
			case "array":
				other.arrayRank = 1
			case "wildcard":
				other.wildcard = true
			case "incomplete":
				other.unresolved = true
			}
			if _, known := genericFamilyConstructorLocalType("var", "E", other, ctx); known {
				t.Fatal("unrelated or incomplete origin donated native Enum local storage")
			}
		})
	}
}
