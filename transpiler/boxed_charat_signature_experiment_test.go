package transpiler

import (
	"bytes"
	"go/format"
	"go/token"
	"reflect"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func boxedCharAtExperimentInvocation(t *testing.T, sources map[string]string) (Ctx, *sitter.Node, []byte) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
	methods := ctx.currentClass.FindMethod().ByOriginalName("read")
	if len(methods) != 1 {
		t.Fatal("read missing")
	}
	ctx.localScope = methods[0]
	var call *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "charAt" {
			call = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(methods[0].DeclarationNode)
	if call == nil {
		t.Fatal("charAt missing")
	}
	return ctx, call, ctx.currentFile.Source
}
func boxedCharAtExperimentRendered(t *testing.T, ctx Ctx, call *sitter.Node, source []byte) string {
	t.Helper()
	expr, ok := tryInstanceIntrinsic(call.ChildByFieldName("object"), "charAt", source, ctx)
	if !ok {
		t.Fatal("actual native callback declined")
	}
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
func TestBoxedCharAtSignatureExperimentCanonicalDonor(t *testing.T) {
	for _, family := range []string{"StringBuilder", "StringBuffer"} {
		t.Run(family, func(t *testing.T) {
			sources := map[string]string{
				"model/Provider.java": "package model; public class Provider { public java.lang." + family + " target; public java.lang.Integer index(){return null;} }",
				"app/Main.java":       "package app; class Main {static class java {static class lang {static class Integer {} static class StringBuilder {} static class StringBuffer {}}} <Integer> int read(model.Provider p){return p.target.charAt(p.index());}}",
			}
			ctx, call, source := boxedCharAtExperimentInvocation(t, sources)
			argument := invocationArgumentNode(call, 0)
			var origin inferredJavaTypeOrigin
			actual, known := inferExprJavaType(argument, ctx, source, &origin)
			if !known || origin.externalName != "java.lang.Integer" || origin.nominalScope != nil || origin.parameter != nil || origin.declaringOwner != findQualifiedSourceClass("model.Provider") {
				t.Fatalf("actual canonical donor lost: %s %+v", actual, origin)
			}
			before := ctx.Clone()
			imports := map[string]string{}
			for name, pkg := range ctx.currentFile.Imports {
				imports[name] = pkg
			}
			for repeat := 0; repeat < 3; repeat++ {
				expected := intrinsicInvocationExpectedArgumentTypes(call, call.ChildByFieldName("object"), "", "charAt", ctx, source)
				if len(expected) != 1 || expected[0] != "int" {
					t.Fatalf("missing fixed signature: %v", expected)
				}
				if !reflect.DeepEqual(ctx, before) || !reflect.DeepEqual(ctx.currentFile.Imports, imports) {
					t.Fatal("signature inference mutated caller context/imports")
				}
			}
			rendered := boxedCharAtExperimentRendered(t, ctx, call, source)
			if !strings.Contains(rendered, "UnboxInteger(") || !strings.Contains(rendered, ".CharAt(") {
				t.Fatalf("missing actual unbox: %s", rendered)
			}
		})
	}
}
func TestBoxedCharAtSignatureExperimentSourceNominalGuard(t *testing.T) {
	ctx, call, source := boxedCharAtExperimentInvocation(t, map[string]string{
		"java.java":     "class java { static class lang { static class Integer {} } }",
		"app/Main.java": "class Main {java.lang.Integer fake; int read(){return new StringBuilder(\"x\").charAt(fake);}}",
	})
	argument := invocationArgumentNode(call, 0)
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(argument, ctx, source, &origin)
	if !known || actual != "java.lang.Integer" || origin.nominalScope == nil || origin.externalName != "" {
		t.Fatalf("actual source nominal identity absent: %s %+v", actual, origin)
	}
	expected := intrinsicInvocationExpectedArgumentTypes(call, call.ChildByFieldName("object"), "", "charAt", ctx, source)
	if len(expected) != 1 || expected[0] != "" {
		t.Fatalf("source nominal acquired int target: %v", expected)
	}
	rendered := boxedCharAtExperimentRendered(t, ctx, call, source)
	t.Logf("actual=%s nominal=%p external=%q rendered=%s", actual, origin.nominalScope, origin.externalName, rendered)
	if strings.Contains(rendered, "UnboxInteger(") {
		t.Fatal("actual source declaration was reclassified as native Integer")
	}
}

func TestBoxedCharAtSignatureRepairSourceMethodUnchanged(t *testing.T) {
	ctx, call, source := boxedCharAtExperimentInvocation(t, map[string]string{"app/Main.java": `class Main { static class Integer {} static class StringBuilder {int charAt(Integer value){return 9;}} int read(StringBuilder target,Integer value){return target.charAt(value);}}`})
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(call.ChildByFieldName("object"), ctx, source, &origin)
	if !known || origin.nominalScope == nil {
		t.Fatal("source receiver origin absent")
	}
	if _, ok := tryInstanceIntrinsic(call.ChildByFieldName("object"), "charAt", source, ctx); ok {
		t.Fatal("legal source charAt became native")
	}
	expected := intrinsicInvocationExpectedArgumentTypes(call, call.ChildByFieldName("object"), "", "charAt", ctx, source)
	if len(expected) != 1 || expected[0] != "" {
		t.Fatalf("source formal acquired native int target: %v", expected)
	}
}
func TestBoxedCharAtSignatureRepairCapturedCanonicalBound(t *testing.T) {
	for _, declaration := range []string{"T extends Integer", "Integer extends java.lang.Integer"} {
		t.Run(declaration, func(t *testing.T) {
			name := strings.Split(declaration, " ")[0]
			ctx, call, source := boxedCharAtExperimentInvocation(t, map[string]string{"app/Main.java": "class Main {<" + declaration + "> int read(StringBuilder target," + name + " value){return target.charAt(value);}}"})
			var origin inferredJavaTypeOrigin
			_, known := inferExprJavaType(invocationArgumentNode(call, 0), ctx, source, &origin)
			if !known || origin.parameter != ctx.localScope.TypeParameters[0].Declaration || origin.externalName != "" {
				t.Fatalf("captured declaration lost: %+v", origin)
			}
			primitive, proved := javaUnboxingPrimitive(name, ctx)
			if !proved || primitive != "int" {
				t.Fatal("canonical bound declaration proof missing")
			}
			rendered := boxedCharAtExperimentRendered(t, ctx, call, source)
			if !strings.Contains(rendered, "UnboxInteger(") {
				t.Fatalf("legal canonical bound refused conversion: %s", rendered)
			}
		})
	}
}
func TestBoxedCharAtSignatureRepairUnknownGuard(t *testing.T) {
	ctx, call, source := boxedCharAtExperimentInvocation(t, map[string]string{"app/Main.java": `class Main {int read(StringBuilder target){return target.charAt(missing());}}`})
	argument := invocationArgumentNode(call, 0)
	if actual, known := inferExprJavaType(argument, ctx, source); known {
		t.Fatalf("unknown unexpectedly known: %s", actual)
	}
	if cost, _, applicable := javaInvocationConversionCost(argument, "int", nil, ctx, source); cost != 64 || !applicable {
		t.Fatal("old unknown scorer behavior changed")
	}
	expected := intrinsicInvocationExpectedArgumentTypes(call, call.ChildByFieldName("object"), "", "charAt", ctx, source)
	if len(expected) != 1 || expected[0] != "" {
		t.Fatalf("unknown scorer was treated as typed int proof: %v", expected)
	}
}
func TestBoxedCharAtSignatureRepairPrimitiveAndNarrowing(t *testing.T) {
	for _, row := range []struct{ javaType, unbox string }{
		{"byte", ""}, {"short", ""}, {"char", ""}, {"int", ""}, {"Byte", "UnboxByte("}, {"Short", "UnboxShort("}, {"Character", "UnboxCharacter("}, {"Integer", "UnboxInteger("},
		{"long", ""}, {"Long", ""}, {"float", ""}, {"Float", ""}, {"double", ""}, {"Double", ""}, {"boolean", ""}, {"Boolean", ""}, {"Integer[]", ""},
	} {
		t.Run(row.javaType, func(t *testing.T) {
			ctx, call, source := boxedCharAtExperimentInvocation(t, map[string]string{"app/Main.java": "class Main {int read(StringBuilder target," + row.javaType + " value){return target.charAt(value);}}"})
			rendered := boxedCharAtExperimentRendered(t, ctx, call, source)
			if row.unbox != "" {
				if !strings.Contains(rendered, row.unbox) {
					t.Fatalf("missing widening unbox: %s", rendered)
				}
			} else if strings.Contains(rendered, "Unbox") {
				t.Fatalf("illegal primitive/wrapper narrowing or array unboxing: %s", rendered)
			}
			if row.javaType == "long" || row.javaType == "float" || row.javaType == "double" {
				if strings.Contains(rendered, "int32(") {
					t.Fatalf("illegal primitive narrowing: %s", rendered)
				}
			}
		})
	}
}
