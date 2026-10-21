package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

// All dimensions are int, even when the owner represents a narrower primitive.
func TestCampaignWrapperConstantJavaTypes(t *testing.T) {
	for _, expression := range []string{
		"Byte.SIZE",
		"java.lang.Byte.SIZE",
		"Byte.BYTES",
		"java.lang.Byte.BYTES",
		"Short.SIZE",
		"java.lang.Short.SIZE",
		"Short.BYTES",
		"java.lang.Short.BYTES",
		"Integer.SIZE",
		"java.lang.Integer.SIZE",
		"Integer.BYTES",
		"java.lang.Integer.BYTES",
		"Long.SIZE",
		"java.lang.Long.SIZE",
		"Long.BYTES",
		"java.lang.Long.BYTES",
		"Float.SIZE",
		"java.lang.Float.SIZE",
		"Float.BYTES",
		"java.lang.Float.BYTES",
		"Float.PRECISION",
		"java.lang.Float.PRECISION",
		"Float.MAX_EXPONENT",
		"java.lang.Float.MAX_EXPONENT",
		"Float.MIN_EXPONENT",
		"java.lang.Float.MIN_EXPONENT",
		"Double.SIZE",
		"java.lang.Double.SIZE",
		"Double.BYTES",
		"java.lang.Double.BYTES",
		"Double.PRECISION",
		"java.lang.Double.PRECISION",
		"Double.MAX_EXPONENT",
		"java.lang.Double.MAX_EXPONENT",
		"Double.MIN_EXPONENT",
		"java.lang.Double.MIN_EXPONENT",
	} {
		t.Run(expression, func(t *testing.T) {
			helper := setupParseHelper(t, "class Probe{static int read(){return "+expression+";}}")
			node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
			if got, known := inferExprJavaType(node, helper.Ctx, helper.File.Source); !known || got != "int" {
				t.Fatalf("%s Java type=(%q,%v), want int", expression, got, known)
			}
			if !compileTimeConstantExpression(node, helper.File.Source, helper.Ctx, map[*symbol.Definition]bool{}) {
				t.Fatalf("type-qualified primitive constant %s was not classified as constant", expression)
			}
		})
	}
}

func TestCampaignWrapperConstantReferenceClassification(t *testing.T) {
	for _, expression := range []string{"Boolean.TRUE", "Boolean.FALSE", "Boolean.TYPE", "Long.TYPE"} {
		t.Run(expression, func(t *testing.T) {
			helper := setupParseHelper(t, "class Probe{static Object read(){return "+expression+";}}")
			node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
			if compileTimeConstantExpression(node, helper.File.Source, helper.Ctx, map[*symbol.Definition]bool{}) {
				t.Fatalf("reference field %s is not a Java constant variable", expression)
			}
		})
	}
}

func TestCampaignWrapperConstantOwnerGuards(t *testing.T) {
	cases := []string{
		"class Probe{static class Long{static int BYTES=47;}static int read(){return Long.BYTES;}}",
		"class Probe<Long>{int read(){return Long.BYTES;}}",
		"class Probe{static <Long> int read(){return Long.BYTES;}}",
		"import foreign.Long;class Probe{static int read(){return Long.BYTES;}}",
		"class Probe{static int read(){return Boolean.SIZE;}}",
		"class Probe{static int read(int java){return java.lang.Long.BYTES;}}",
	}
	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx
			for _, method := range ctx.currentClass.Methods {
				if method.OriginalName == "read" {
					ctx.localScope = method
				}
			}
			node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
			if _, accepted := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), node.ChildByFieldName("field").Content(helper.File.Source), helper.File.Source, ctx); accepted {
				t.Fatal("source/foreign/binder/unsupported field borrowed a canonical constant")
			}
		})
	}
}
