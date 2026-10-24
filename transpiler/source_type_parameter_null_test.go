package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// A typed null belongs to its resolved Java binder, even when a source class or
// an enclosing binder has the same spelling. An enclosing expected type must
// not leak into a null operand of an unrelated expression.
func TestSourceTypeParameterNullDeclarationControls(t *testing.T) {
	for _, tc := range []struct {
		name, source, target string
		binder, owned        bool
	}{
		{"method", `class Owner {<T> T run(){return null;}}`, "T", true, true},
		{"dependent", `class Base {} class Owner {<B extends Base,T extends B> T run(){return null;}}`, "T", true, true},
		{"class", `class Owner<T> {T run(){return null;}}`, "T", true, true},
		{"shadowed class binder", `class Owner<T> {<T> T run(){return null;}}`, "T", true, true},
		{"shadowed source class", `class T {} class Owner {<T> T run(){return null;}}`, "T", true, true},
		{"nominal source class", `class T {} class Owner {T run(){return null;}}`, "T", false, true},
		{"array of binder", `class Owner<T> {T[] run(){return null;}}`, "T[]", false, true},
		{"qualified nominal", `class Owner<T> {static class Nested {} Nested run(){return null;}}`, "Owner.Nested", false, true},
		{"unowned null operand", `class Owner<T> {boolean run(T value){return value==null;}}`, "T", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope("Owner")
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("missing method")
			}
			ctx.localScope = methods[0]
			null := findNode(methods[0].DeclarationNode, "null_literal")
			ctx.expectedType = tc.target
			if tc.owned {
				ctx.expectedTypeRoot = null
			} else {
				ctx.expectedTypeRoot = findNode(methods[0].DeclarationNode, "binary_expression")
			}
			declaration := visibleTypeParameterDeclarationForJavaType(tc.target, ctx)
			if (declaration != nil) != tc.binder {
				t.Fatal("fixture lost declaration identity")
			}
			want := "nil"
			if tc.binder && tc.owned {
				want = "*new(" + declaration.GoName + ")"
			}
			if got := symbol.NodeToStr(ParseExpr(null, helper.File.Source, ctx)); got != want {
				t.Fatalf("null lowered as %s, want %s", got, want)
			}
			// Argument conversion explicitly owns its boundary independently of an
			// enclosing expression's expected type.
			if tc.binder {
				want = "*new(" + declaration.GoName + ")"
				if got := symbol.NodeToStr(coerceArgumentToExpectedType(ParseExpr(null, helper.File.Source, ctx), null, tc.target, ctx, helper.File.Source)); got != want {
					t.Fatalf("argument lowered as %s, want %s", got, want)
				}
			}
		})
	}
}

func TestSourceTypeParameterNullBoundariesJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
public class Main {
 public static class Base {}
 public static class Box<X> {
  public X value;
  public Box(X value){this.value=value;}
 }
 static <X> X identity(X value){return value;}
 static <X> X empty(){return (null);}
 static <B extends Base,T extends B> boolean dependent(){
  B first=null; T second=(null); B widened=second;
  second=null;
  T direct=Main.<T>identity(null);
  T parenthesized=Main.<T>identity((null));
  T returned=Main.<T>empty();
  Box<T> box=new Box<T>(null);
  T selected=true?null:second;
  return first==widened && direct==null && parenthesized==null && returned==null && box.value==null && selected==null;
 }
 static <T extends java.lang.Integer> boolean boxed(){T empty=null;return empty==null;}
 public static class T {}
 static <T> boolean shadow(){T empty=null; return empty==null;}
 static boolean nominal(){T empty=null; return empty==null;}
 public static void main(String[] args){
  System.out.println(Main.<Base,Base>dependent());
  System.out.println(Main.<java.lang.Integer>boxed());
  System.out.println(Main.<Base>shadow());
  System.out.println(nominal());
 }
}`, "true\ntrue\ntrue\ntrue\n")
}

func TestSourceTypeParameterNullClassAndArrayControlsJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
public class Main {
 public static class Base {}
 public static class Box<X> {
  X initial=null;
  X direct(){return null;}
  X parenthesized(){return (null);}
 }
 static <T> boolean array(){T[] empty=null;return empty==null;}
 static <T> int primitive(T value){int number=17;return number;}
 public static void main(String[] args){
  Box<Base> box=new Box<Base>();
  System.out.println(box.initial==null);
  System.out.println(box.direct()==null);
  System.out.println(box.parenthesized()==null);
  System.out.println(Main.<Base>array());
  System.out.println(Main.<java.lang.Integer>primitive(5));
 }
}`, "true\ntrue\ntrue\ntrue\n17\n")
}
