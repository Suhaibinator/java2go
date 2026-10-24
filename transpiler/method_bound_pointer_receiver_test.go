package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestMethodBoundPointerCombinedActualScalarProof(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		project      bool
	}{
		{"method", `class Base{String voice(){return "base";}}class Host{<U extends Base>String run(U value){return value.voice();}}`, true},
		{"shadow-alias", `class Base{String voice(){return "base";}}class Host<U extends Base>{<U extends Base>String run(U value){return value.voice();}}`, true},
		{"transitive", `class Base{String voice(){return "base";}}class Host{<B extends Base,U extends B>String run(U value){return value.voice();}}`, true},
		{"private", `class Base{String voice(){return "base";}}class Host{private<U extends Base>String run(U value){return value.voice();}}`, false},
		{"implicit-object", `class Base{String voice(){return "base";}}class Host{<U>String run(U value){return value.toString();}}`, false},
		{"array", `class Base{String voice(){return "base";}}class Host{<U extends Base>String run(U[] value){return value.toString();}}`, false},
		{"parameterized", `class Base<X>{String voice(){return "base";}}class Host{<U extends Base<String>>String run(U value){return value.voice();}}`, false},
		{"intersection", `interface Mark{}class Base{String voice(){return "base";}}class Host{<U extends Base & Mark>String run(U value){return value.voice();}}`, false},
		{"nominal", `class Base{String voice(){return "base";}}class U{String voice(){return "nominal";}}class Provider{static U value(){return new U();}}class Host{<U extends Base>String run(){return Provider.value().voice();}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = h.File.Symbols.FindClassScope("Host")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			receiver := invocation.ChildByFieldName("object")
			var origin inferredJavaTypeOrigin
			if _, ok := inferExprJavaType(receiver, ctx, h.File.Source, &origin); !ok {
				t.Fatal("missing actual receiver origin")
			}
			scope := h.File.Symbols.FindClassScope("Base")
			value := ast.NewIdent("onceOnly")
			got, ok := methodBoundConcretePointerReceiverExpr(value, origin, scope, ctx)
			if ok != tc.project {
				t.Fatalf("method proof=%v want=%v origin=%#v", ok, tc.project, origin)
			}
			combined := projectDependentTypeParameterReceiver(value, receiver, scope, ctx, h.File.Source)
			if tc.project {
				assertMethodBoundPointerOnce(t, got, value)
				assertMethodBoundPointerOnce(t, combined, value)
				// Typed null remains nil after pointer conversion; the helper may not
				// select a subobject/self field or introduce a duplicated operand.
				null := ast.NewIdent("nil")
				projected, valid := methodBoundConcretePointerReceiverExpr(null, origin, scope, ctx)
				if !valid {
					t.Fatal("null AST lost exact method proof")
				}
				assertMethodBoundPointerOnce(t, projected, null)
			} else if combined != value {
				t.Fatal("unsupported combined receiver acquired method projection")
			}
		})
	}
}
func assertMethodBoundPointerOnce(t *testing.T, expr, operand ast.Expr) {
	t.Helper()
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Args[0] != operand {
		t.Fatalf("receiver is not a single conversion: %#v", expr)
	}
	if _, ok := call.Fun.(*ast.StarExpr); !ok {
		t.Fatalf("receiver did not use an actual pointer constraint: %#v", call.Fun)
	}
}
func TestMethodBoundPointerOriginOwnerAndSlotRefusal(t *testing.T) {
	h := setupParseHelper(t, `class Base{String voice(){return "base";}}class Other{String voice(){return "other";}}class Host<C extends Base>{<U extends Base>String run(U value){return value.voice();}<V extends Base>String other(V value){return value.voice();}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Host")
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	receiver := invocation.ChildByFieldName("object")
	var origin inferredJavaTypeOrigin
	if _, ok := inferExprJavaType(receiver, ctx, h.File.Source, &origin); !ok || origin.parameter == nil {
		t.Fatal("actual method origin missing")
	}
	base := h.File.Symbols.FindClassScope("Base")
	value := ast.NewIdent("value")
	refuse := func(name string, current inferredJavaTypeOrigin, scope *symbol.ClassScope, c Ctx) {
		t.Helper()
		if _, ok := methodBoundConcretePointerReceiverExpr(value, current, scope, c); ok {
			t.Fatalf("%s admitted", name)
		}
	}
	unresolved := origin
	unresolved.unresolved = true
	refuse("unresolved", unresolved, base, ctx)
	missing := origin
	missing.parameter = nil
	refuse("missing-origin", missing, base, ctx)
	nominal := origin
	nominal.nominalScope = base
	refuse("nominal-origin", nominal, base, ctx)
	refuse("wrong-selected-bound", origin, h.File.Symbols.FindClassScope("Other"), ctx)
	wrongSlot := origin
	wrongSlot.parameter = ctx.currentClass.FindMethodByName("other", nil).TypeParameters[0].Declaration
	refuse("other-method-slot", wrongSlot, base, ctx)
	classSlot := origin
	classSlot.parameter = ctx.currentClass.TypeParameters[0].Declaration
	refuse("class-owned-slot", classSlot, base, ctx)
	copied := *ctx.localScope
	c := ctx.Clone()
	c.localScope = &copied
	refuse("copied-method", origin, base, c)
	c = ctx.Clone()
	c.currentClass = nil
	refuse("missing-owner", origin, base, c)
	c = ctx.Clone()
	owner := *ctx.currentClass
	c.currentClass = &owner
	refuse("copied-owner", origin, base, c)
	// Array proof comes from an actual declaration, not a synthetic type string.
	a := setupParseHelper(t, `class Base{String voice(){return "base";}}class Host{<U extends Base>String run(U[] value){return value.toString();}}`)
	ac := a.Ctx.Clone()
	ac.currentClass = a.File.Symbols.FindClassScope("Host")
	ac.localScope = ac.currentClass.FindMethodByName("run", nil)
	arrayOrigin := declaredJavaTypeOrigin(definitionDeclaredJavaType(ac.localScope.Parameters[0]), ac)
	if _, ok := methodBoundConcretePointerReceiverExpr(value, arrayOrigin, a.File.Symbols.FindClassScope("Base"), ac); ok {
		t.Fatal("declared array origin admitted")
	}
}
