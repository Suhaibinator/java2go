package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestClassBoundPointerProjectionPhysicalProof(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		project      bool
	}{
		{"source class", `class Animal {String voice(){return "base";}} class Bound<T extends Animal>{String run(T value){return value.voice();}}`, true},
		{"source interface", `interface Animal {String voice();} class Bound<T extends Animal>{String run(T value){return value.voice();}}`, false},
		{"abstract interface representation", `abstract class Animal {abstract String voice();} class Bound<T extends Animal>{String run(T value){return value.voice();}}`, false},
		{"method-owned shadow", `class Animal {String voice(){return "base";}} class Bound<T extends Animal>{<T extends Animal> String run(T value){return value.voice();}}`, false},
		{"method-owned only", `class Animal {String voice(){return "base";}} class Bound{<T extends Animal> String run(T value){return value.voice();}}`, false},
		{"dependent", `class Animal {String voice(){return "base";}} class Bound<B extends Animal,T extends B>{String run(T value){return value.voice();}}`, false},
		{"intersection", `interface Mark{} class Animal {String voice(){return "base";}} class Bound<T extends Animal & Mark>{String run(T value){return value.voice();}}`, false},
		{"unbounded", `class Animal {String voice(){return "base";}} class Bound<T>{String run(T value){return value.toString();}}`, false},
		{"canonical", `class Animal {} class Bound<T extends java.security.GeneralSecurityException>{Throwable run(T value){return value.getCause();}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope("Bound")
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			object := invocation.ChildByFieldName("object")
			owner := helper.File.Symbols.FindClassScope("Animal")
			value := ast.NewIdent("value")
			javaType, known := inferExprJavaType(object, ctx, helper.File.Source)
			if !known {
				t.Fatal("receiver type not resolved")
			}
			got := ast.Expr(value)
			if projected, ok := classBoundConcretePointerReceiverExpr(value, javaType, owner, ctx); ok {
				got = projected
			}
			if tc.project {
				call, ok := got.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 || call.Args[0] != value {
					t.Fatalf("missing exact once-only bound conversion: %s", symbol.NodeToStr(got))
				}
				if _, ok := call.Fun.(*ast.StarExpr); !ok {
					t.Fatalf("conversion is not proven pointer: %s", symbol.NodeToStr(got))
				}
			} else if got != value {
				t.Fatalf("unproven binder projected: %s", symbol.NodeToStr(got))
			}
		})
	}
}
