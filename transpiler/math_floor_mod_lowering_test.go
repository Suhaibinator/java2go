package transpiler

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestMathFloorModRegisteredDeclarations(t *testing.T) {
	defs := staticIntrinsicImportSignatures[intrinsicKey{"Math", "floorMod"}]
	if len(defs) != 3 {
		t.Fatalf("want3 actual JDK declarations, got%d", len(defs))
	}
	for index, want := range mathFloorModDeclarations {
		def := defs[index]
		if def == nil || !def.IsStatic || def.IsVariadic || len(def.TypeParameters) != 0 || def.OriginalName != "floorMod" || !reflect.DeepEqual(def.OriginalParameterTypes(), want.parameters[:]) || def.OriginalType != want.result {
			t.Fatalf("declaration%d malformed: %+v", index, def)
		}
	}
}

func TestMathFloorModInvocationWidths(t *testing.T) {
	for _, row := range []struct{ left, right, result, helper string }{
		{"int", "int", "int", "MathFloorModInt"}, {"long", "int", "int", "MathFloorModLongInt"}, {"long", "long", "long", "MathFloorModLong"}, {"int", "long", "long", "MathFloorModLong"},
		{"byte", "short", "int", "MathFloorModInt"}, {"char", "byte", "int", "MathFloorModInt"}, {"Integer", "Integer", "int", "MathFloorModInt"}, {"Long", "Integer", "int", "MathFloorModLongInt"}, {"Integer", "Long", "long", "MathFloorModLong"},
	} {
		for _, prefix := range []string{"Math.", "java.lang.Math.", ""} {
			t.Run(row.left+"/"+row.right+"/"+prefix, func(t *testing.T) {
				source := fmt.Sprintf("import static java.lang.Math.floorMod; class Use { %s call(%s a,%s b){return %sfloorMod(a,b);} }", row.result, row.left, row.right, prefix)
				h := setupParseHelper(t, source)
				ctx := h.Ctx.Clone()
				ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
				node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
				got, known := inferExprJavaType(node, ctx, h.File.Source)
				if !known || got != row.result {
					t.Fatalf("selectedJavaresult=%q/%v want%s", got, known, row.result)
				}
				expression := ParseExpr(node, h.File.Source, ctx)
				var out bytes.Buffer
				if err := format.Node(&out, token.NewFileSet(), expression); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(out.String(), "stdjava."+row.helper+"(") {
					t.Fatalf("wrongphysicaloverload:\n%s", out.String())
				}
			})
		}
	}
}

func TestMathFloorModStaticImportApplicability(t *testing.T) {
	for _, expression := range []string{"floorMod(1.0,2)", "floorMod(1,2.0f)", "floorMod(true,false)", "floorMod(null,1)", "floorMod(1)", "floorMod(1,2,3)"} {
		t.Run(expression, func(t *testing.T) {
			source := "import static java.lang.Math.*;class Use{void call(){" + expression + ";}}"
			h := setupParseHelper(t, source)
			ctx := h.Ctx.Clone()
			ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			resolution := resolveStaticImportedMethod(node, ctx, h.File.Source)
			if resolution.intrinsic != "" || resolution.source != nil || resolution.problem != "no applicable static method import floorMod" {
				t.Fatalf("invalid primitive invocation admitted: %+v", resolution)
			}
		})
	}
}

func TestMathFloorModNamespaceShields(t *testing.T) {
	for _, source := range []string{
		"class Math{static int floorMod(int a,int b){return 17;}}class Use{int call(){return Math.floorMod(-7,3);}}",
		"class Fake{int floorMod(int a,int b){return 23;}}class Use{int call(Fake Math){return Math.floorMod(-7,3);}}",
		"class Use{<Math>int call(){return Math.floorMod(-7,3);}}",
		"class Use<java>{int call(){return java.lang.Math.floorMod(-7,3);}}",
	} {
		h := setupParseHelper(t, source)
		scope := h.File.Symbols.FindClassScope("Use")
		if scope == nil {
			t.Fatal("actual Use declaration missing")
		}
		ctx := classScopeCtx(scope, h.Ctx)
		ctx.localScope = scope.FindMethodByName("call", nil)
		node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		object := node.ChildByFieldName("object")
		if owner, known := intrinsicStaticClassName(object, ctx, h.File.Source); known && owner == "Math" {
			t.Fatalf("source/value/binder donatedcanonicalMath: %s", source)
		}
	}
}

func TestMathFloorModReferenceConversions(t *testing.T) {
	h := setupParseHelper(t, "class Use{}")
	ctx := h.Ctx
	for _, row := range []struct {
		actual     []string
		parameters [2]string
		result     string
	}{
		{[]string{"int", "int"}, [2]string{"int", "int"}, "int"}, {[]string{"long", "int"}, [2]string{"long", "int"}, "int"},
		{[]string{"int", "long"}, [2]string{"long", "long"}, "long"}, {[]string{"long", "long"}, [2]string{"long", "long"}, "long"},
		{[]string{"byte", "short"}, [2]string{"int", "int"}, "int"}, {[]string{"java.lang.Integer", "java.lang.Integer"}, [2]string{"int", "int"}, "int"},
		{[]string{"java.lang.Long", "java.lang.Integer"}, [2]string{"long", "int"}, "int"}, {[]string{"java.lang.Integer", "java.lang.Long"}, [2]string{"long", "long"}, "long"},
	} {
		got, handled := mathFloorModReferenceDeclaration("Math", "floorMod", row.actual, ctx)
		if !handled || got.parameters != row.parameters || got.result != row.result {
			t.Fatalf("ref%v -> %+v/%v", row.actual, got, handled)
		}
	}
	for _, actual := range [][]string{{"float", "int"}, {"int", "double"}, {"boolean", "boolean"}, {"java.lang.Number", "int"}, {"java.lang.Object", "int"}, {"int"}} {
		got, handled := mathFloorModReferenceDeclaration("Math", "floorMod", actual, ctx)
		if !handled || got.result != "" {
			t.Fatalf("inapplicablecallbackadmitted: %v %+v/%v", actual, got, handled)
		}
	}
	if _, handled := mathFloorModReferenceDeclaration("Other", "floorMod", []string{"int", "int"}, ctx); handled {
		t.Fatal("unrelatedownerhandled")
	}
	if _, handled := mathFloorModReferenceDeclaration("Math", "other", []string{"int", "int"}, ctx); handled {
		t.Fatal("unrelatedmethodhandled")
	}
}
