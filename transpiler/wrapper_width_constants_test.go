package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"testing"
)

const wrapperWidthConstantsMain = `package probe;
public class Main {
 static String pick(int value) { return "int"; }
 static String pick(long value) { return "long"; }
 public static String run() {
  int offset = 2147483644;
  offset += Long.BYTES;
  byte narrow = 127; narrow += Long.BYTES;
  char letter = 65535; letter += Character.BYTES;
  byte direct = Long.BYTES;
  int[] values = new int[Double.BYTES];
  return Byte.BYTES+":"+Byte.SIZE+":"+Short.BYTES+":"+Short.SIZE+":"+Integer.BYTES+":"+Integer.SIZE+":"+Long.BYTES+":"+Long.SIZE+":"+Character.BYTES+":"+Character.SIZE+":"+Float.BYTES+":"+Float.SIZE+":"+Double.BYTES+":"+Double.SIZE
    +"|"+pick(Long.BYTES)+":"+pick(Double.SIZE)+":"+offset+":"+narrow+":"+(int)letter+":"+direct+":"+values.length+":"+(1L << Long.SIZE)+":"+(1 << Integer.SIZE);
 }
 public static void main(String[] args) { System.out.println(run()); }
}`
const wrapperWidthConstantsExpected = "1:8:2:16:4:32:8:64:2:16:4:32:8:64|int:int:-2147483644:-121:1:8:8:1:1"

func TestWrapperWidthConstantsDeclaredIntAndEmission(t *testing.T) {
	for _, wrapper := range []string{"Byte", "Short", "Integer", "Long", "Character", "Float", "Double"} {
		for _, field := range []string{"BYTES", "SIZE"} {
			for _, prefix := range []string{"", "java.lang."} {
				expression := prefix + wrapper + "." + field
				t.Run(expression, func(t *testing.T) {
					helper := setupParseHelper(t, "class WidthConsumer { int read() { return "+expression+"; } }")
					node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
					got, known := inferExprJavaType(node, helper.Ctx, helper.File.Source)
					if !known || got != "int" {
						t.Errorf("WIDTH_CONSTANT_TYPE %s = %q/%t; want int/true", expression, got, known)
					}
					result, emitted := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), field, helper.File.Source, helper.Ctx)
					call, typed := result.(*ast.CallExpr)
					if !emitted || !typed {
						t.Fatalf("WIDTH_CONSTANT_EMISSION %s = %T/%t; want typed int32", expression, result, emitted)
					}
					name, ok := call.Fun.(*ast.Ident)
					if !ok || name.Name != "int32" || len(call.Args) != 1 {
						t.Fatalf("WIDTH_CONSTANT_TYPED_INT %s = %#v", expression, call)
					}
					literal, ok := call.Args[0].(*ast.BasicLit)
					if !ok || literal.Kind != token.INT {
						t.Fatalf("WIDTH_CONSTANT_INTEGER %s = %#v", expression, call.Args)
					}
				})
			}
		}
	}
}

func TestWrapperWidthConstantsCompoundAssignmentProof(t *testing.T) {
	helper := setupParseHelper(t, `class WidthOffset { int read() { int offset=17; offset += Long.BYTES; return offset; } }`)
	method := helper.Ctx.currentClass.FindMethod().ByOriginalName("read")[0]
	ctx := helper.Ctx
	ctx.localScope = method
	node := findNode(helper.File.Ast, "assignment_expression")
	resetDiagnostics()
	defer resetDiagnostics()
	result := ParseExpr(node, helper.File.Source, ctx)
	if _, bad := result.(*ast.BadExpr); bad || len(Diagnostics()) != 0 {
		t.Fatalf("WIDTH_COMPOUND_ASSIGNMENT %T diagnostics=%v", result, Diagnostics())
	}
}

func TestWrapperWidthConstantsSourceAndBinderShields(t *testing.T) {
	for _, control := range []struct{ name, source string }{
		{"source", `class WidthConsumer { static class Long {static long BYTES=73L;} long read(){return Long.BYTES;} }`},
		{"class-binder", `class WidthConsumer<Long> { int read(){return Long.BYTES;} }`},
		{"method-binder", `class WidthConsumer { <Long> int read(){return Long.BYTES;} }`},
		{"value-prefix", `class WidthConsumer { static class Value {long BYTES;} long read(Value Long){return Long.BYTES;} }`},
	} {
		t.Run(control.name, func(t *testing.T) {
			helper := setupParseHelper(t, control.source)
			ctx := helper.Ctx
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("read")[0]
			node := findNode(ctx.localScope.DeclarationNode, "return_statement").NamedChild(0)
			if _, ok := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), "BYTES", helper.File.Source, ctx); ok {
				t.Fatal("source/value/binder acquired native width constant")
			}
			if got, known := inferIntrinsicFieldResultType(node, ctx, helper.File.Source); known {
				t.Fatalf("source/value/binder acquired native result %s", got)
			}
		})
	}
}

func TestWrapperWidthConstantsForeignOwnerContext(t *testing.T) {
	files := map[string]string{
		"foreign/Long.java": `package foreign;public class Long {public static long BYTES=73L;public static char SIZE='Q';}`,
		"donor/Widths.java": `package donor;public class Widths {public static int bytes(){return java.lang.Long.BYTES;} public static int bits(){return java.lang.Double.SIZE;}}`,
		"app/Main.java":     `package app;import foreign.Long;public class Main<Integer>{long source(){return Long.BYTES;} int nativeWidth(){return donor.Widths.bytes();}}`,
	}
	ctx := sourceGenericViewDemandTestContext(t, files)
	owner := findQualifiedSourceClass("app.Main")
	ctx = classScopeCtx(owner, ctx)
	method := owner.FindMethod().ByOriginalName("source")[0]
	ctx.localScope = method
	node := findNode(method.DeclarationNode, "return_statement").NamedChild(0)
	source := []byte(files["app/Main.java"])
	if _, ok := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), "BYTES", source, ctx); ok {
		t.Fatal("imported source Long intercepted")
	}
	got, known := inferExprJavaType(node, ctx, source)
	if !known || got != "long" {
		t.Fatalf("source Long width %s/%t", got, known)
	}
	donor := findQualifiedSourceClass("donor.Widths")
	ctx = classScopeCtx(donor, ctx)
	ctx.localScope = donor.FindMethod().ByOriginalName("bytes")[0]
	node = findNode(ctx.localScope.DeclarationNode, "return_statement").NamedChild(0)
	source = []byte(files["donor/Widths.java"])
	got, known = inferExprJavaType(node, ctx, source)
	if !known || got != "int" {
		t.Fatalf("donor canonical width %s/%t", got, known)
	}
}

func TestWrapperWidthConstantsBooleanHasNoInventedFields(t *testing.T) {
	for _, field := range []string{"BYTES", "SIZE"} {
		helper := setupParseHelper(t, "class WidthConsumer { int read(){return Boolean."+field+";} }")
		node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
		if _, ok := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), field, helper.File.Source, helper.Ctx); ok {
			t.Fatal("Boolean invented " + field)
		}
		if typ, ok := inferIntrinsicFieldResultType(node, helper.Ctx, helper.File.Source); ok {
			t.Fatalf("Boolean invented %s result %s", field, typ)
		}
	}
}

func TestWrapperWidthConstantsGeneratedConsumer(t *testing.T) {
	generated := renderGoFileFromJava(t, wrapperWidthConstantsMain)
	runGeneratedWithStdjava(t, generated, fmt.Sprintf(`package probe
import ("slices"; "testing"; "unicode/utf16")
func TestWidthConsumer(t *testing.T) {
 got := Run()
 if got == nil || !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune(%q))) {t.Fatalf("unexpected wrapper widths: %%v", got)}
}`, wrapperWidthConstantsExpected))
}

func TestWrapperWidthConstantsStaticImportsAndLocalShadow(t *testing.T) {
	for _, control := range []struct {
		source, want string
		intrinsic    bool
	}{
		{`import static java.lang.Long.BYTES;class WidthImport {int read(){return BYTES;}}`, "int", true},
		{`import static java.lang.Double.*;class WidthImport {int read(){return SIZE;}}`, "int", true},
		{`import static java.lang.Long.BYTES;class WidthImport {long BYTES=73L;long read(){return BYTES;}}`, "long", false},
	} {
		t.Run(control.source, func(t *testing.T) {
			helper := setupParseHelper(t, control.source)
			ctx := helper.Ctx
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("read")[0]
			node := findNode(ctx.localScope.DeclarationNode, "return_statement").NamedChild(0)
			got, known := inferExprJavaType(node, ctx, helper.File.Source)
			if !known || got != control.want {
				t.Fatalf("static import type %s/%t want %s", got, known, control.want)
			}
			result := ParseExpr(node, helper.File.Source, ctx)
			if call, ok := result.(*ast.CallExpr); ok {
				if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "int32" {
					if control.intrinsic {
						return
					}
					t.Fatal("local field replaced by imported native width")
				}
			}
			if control.intrinsic {
				t.Fatalf("native static import not typed int32: %T", result)
			}
		})
	}
}
