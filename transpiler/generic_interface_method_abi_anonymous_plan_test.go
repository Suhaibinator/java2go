package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenericInterfaceABIAnonymousPlanScalarAndLifetime(t *testing.T) {
	h := setupParseHelper(t, `class Object{}class Box<X>{}interface Poly{<T>T id(T value);}class Main<C>{Poly make(){return new Poly(){public<U>U id(U value){return value;}};}}`)
	owner := h.File.Symbols.FindClassScope("Poly")
	method := owner.FindMethodByName("id", nil)
	entry := planGenericMethodEntryABI(owner, method, classScopeCtx(owner, h.Ctx))
	creation := findNode(h.File.Ast, "object_creation_expression")
	implementationNode := anonymousClassMethods(findNode(creation, "class_body"))[0]
	implementation := scopeForAnonymousMethod(implementationNode, method, h.File.Source)
	ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
	ctx.localScope = implementation
	ctx.erasedGenericMethodBody = implementation
	plan := planGenericInterfaceMethodBodyABI(entry, implementation, implementationNode, ctx)
	if plan == nil {
		t.Fatal("exact alpha-renamed implementation lacks body plan")
	}
	ctx.genericInterfaceMethodBody = plan
	if !plan.scalar("U", ctx) {
		t.Fatal("actual method declaration not selected")
	}
	render := func(expr ast.Expr) string {
		var b bytes.Buffer
		if err := format.Node(&b, token.NewFileSet(), expr); err != nil {
			t.Fatal(err)
		}
		return b.String()
	}
	if got := render(javaTypeStringToGoTypeExpr("U", inScopeTypeParameters(ctx), ctx)); got != "any" {
		t.Fatalf("scalar=%s", got)
	}
	actual, ok := javaTypeDescriptorExpr("U", ctx)
	if !ok || render(actual) != render(stdjavaQualifiedExpr("ObjectTypeID", ctx)) {
		t.Fatal("scalar descriptor is not canonical Object")
	}
	if got := erasedAnonymousMethodJavaType("U", ctx); got != "java.lang.Object" {
		t.Fatalf("semantic scalar=%s", got)
	}
	old := ctx.Clone()
	old.genericInterfaceMethodBody = nil
	for _, typ := range []string{"U[]", "? extends U", "Box<U>", "Object", "C"} {
		t.Run(typ, func(t *testing.T) {
			before := render(javaTypeStringToGoTypeExpr(typ, inScopeTypeParameters(old), old))
			after := render(javaTypeStringToGoTypeExpr(typ, inScopeTypeParameters(ctx), ctx))
			if before != after {
				t.Fatalf("non-scalar AST changed: %s -> %s", before, after)
			}
			a, aok := javaTypeDescriptorExpr(typ, old)
			b, bok := javaTypeDescriptorExpr(typ, ctx)
			if aok != bok || !reflect.DeepEqual(a, b) {
				t.Fatalf("non-scalar descriptor changed: %#v -> %#v", a, b)
			}
			if erasedAnonymousMethodJavaType(typ, old) != erasedAnonymousMethodJavaType(typ, ctx) {
				t.Fatal("nested semantic substitution changed")
			}
		})
	}
	clone := ctx.Clone()
	if !plan.scalar("U", clone) {
		t.Fatal("same body Clone lost plan")
	}
	nested := *implementation
	clone.localScope = &nested
	if plan.scalar("U", clone) {
		t.Fatal("nested/foreign method with identical visible declarations acquired plan")
	}
	clone = ctx.Clone()
	clone.erasedGenericMethodBody = &nested
	if plan.scalar("U", clone) {
		t.Fatal("foreign erased body acquired plan")
	}
	clone = ctx.Clone()
	clone.currentClass = owner
	if plan.scalar("U", clone) {
		t.Fatal("foreign emission owner acquired plan")
	}
	copy := *implementation
	if planGenericInterfaceMethodBodyABI(entry, &copy, method.DeclarationNode, ctx) != nil {
		t.Fatal("foreign source node admitted")
	}
	foreignEntry := *entry
	foreignMethod := *method
	foreignEntry.method = &foreignMethod
	if planGenericInterfaceMethodBodyABI(&foreignEntry, implementation, implementationNode, ctx) != nil {
		t.Fatal("copied interface definition admitted")
	}
	explicit := *implementation
	explicit.TypeParameters = append([]symbol.TypeParam(nil), implementation.TypeParameters...)
	explicit.TypeParameters[0].Bounds = []symbol.JavaType{{Original: "Object"}}
	if planGenericInterfaceMethodBodyABI(entry, &explicit, implementationNode, ctx) != nil {
		t.Fatal("written bound acquired implicit Object plan")
	}
	mismatched := *implementation
	mismatched.Parameters = append([]*symbol.Definition(nil), implementation.Parameters...)
	param := *mismatched.Parameters[0]
	param.OriginalType = "Object"
	param.TypeParameterBindings = nil
	param.DirectTypeParameter = nil
	mismatched.Parameters[0] = &param
	if planGenericInterfaceMethodBodyABI(entry, &mismatched, implementationNode, ctx) != nil {
		t.Fatal("nominal parameter admitted as method ordinal")
	}
}
