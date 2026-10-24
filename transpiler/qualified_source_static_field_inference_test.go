package transpiler

import "testing"

func TestQualifiedSourceStaticFieldDeclarationTypes(t *testing.T) {
	for _, tc := range []struct{ declaration, javaType string }{
		{"int value=37", "int"}, {"char value='Q'", "char"},
		{"long value=4294967297L", "long"}, {"float value=1.5f", "float"},
		{"double value=2.5", "double"}, {"boolean value=true", "boolean"},
		{"String value=\"text\"", "String"}, {"int[] value=new int[2]", "int[]"},
	} {
		t.Run(tc.javaType, func(t *testing.T) {
			h := setupParseHelper(t, `class Probe {static class Namespace {static class Owner {static `+tc.declaration+`;}} static Object read(){return Namespace.Owner.value;}}`)
			ctx := h.Ctx
			for _, method := range ctx.currentClass.Methods {
				if method.OriginalName == "read" {
					ctx.localScope = method
				}
			}
			node := findNode(h.File.Ast, "return_statement").NamedChild(0)
			access, resolved := resolveStaticFieldAccess(node, h.File.Source, ctx)
			if !resolved || access.resolution.def.OriginalType != tc.javaType {
				t.Fatal("control did not resolve exact static declaration")
			}
			var origin inferredJavaTypeOrigin
			got, known := inferExprJavaType(node, ctx, h.File.Source, &origin)
			if !known || got != tc.javaType {
				t.Fatalf("inferred %q known=%v, want %s", got, known, tc.javaType)
			}
			if origin.javaType != got {
				t.Fatalf("declaration origin lost: %q != %q", origin.javaType, got)
			}
		})
	}
}

func TestQualifiedSourceStaticFieldDeclarationContextJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml":                         `<project><modelVersion>4.0.0</modelVersion><groupId>fieldproof</groupId><artifactId>declaration</artifactId><version>1</version></project>`,
		"src/main/java/model/Base.java":   `package model;import token.Token;public class Base{public static int value=37;public static char glyph='Q';public static long amount=4294967297L;public static String text="declared";public static Token token=new Token(7);public static int[] array=new int[3];}`,
		"src/main/java/model/Child.java":  `package model;public class Child extends Base{}`,
		"src/main/java/token/Token.java":  `package token;public class Token{public int id;public Token(int n){id=n;}public String toString(){return "token:"+id;}}`,
		"src/main/java/foreign/Long.java": `package foreign;public class Long{public static int BYTES=53;}`,
		"src/main/java/app/Imported.java": `package app;import static model.Base.value;import foreign.Long;public class Imported{static int read(){return value+Long.BYTES;}}`,
		"src/main/java/app/Main.java": `package app;import model.Child;public class Main{
 static class Space{static class Part{static class Holder{static int value=61;}}}
 static int pick(int n){return 1;}static int pick(long n){return 2;}static int pick(char n){return 3;}static int pick(Object n){return 4;}
 static <Long> int canonical(){return java.lang.Long.BYTES;}
 public static void main(String[] args){System.out.println(model.Child.value);System.out.println(Child.glyph);System.out.println(model.Child.amount);System.out.println(model.Child.text);System.out.println(model.Child.token);System.out.println(model.Child.token.id);System.out.println(model.Child.array.length);System.out.println(pick(model.Child.value));System.out.println(pick(model.Child.amount));System.out.println(pick(model.Child.glyph));System.out.println(pick(model.Child.token));System.out.println("sum:"+model.Child.value);System.out.println(Imported.read());System.out.println(Space.Part.Holder.value);System.out.println(canonical());}
}`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func TestQualifiedSourceStaticFieldValueEffectsJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>fieldproof</groupId><artifactId>effects</artifactId><version>1</version></project>`,
		"src/main/java/app/Main.java": `package app;public class Main{
 static int effects;static class Space{static class Holder{static int value=37;static char glyph='Q';int instance=19;}}
 static Space.Holder receiver(){effects++;return null;}static Space.Holder throwing(){effects++;throw new IllegalStateException("qualifier");}
 static int pick(int n){return 1;}static int pick(char n){return 2;}static int pick(Object n){return 3;}
 public static void main(String[] args){System.out.println(receiver().value);System.out.println(effects);System.out.println(pick(receiver().glyph));System.out.println(effects);try{System.out.println(receiver().instance);}catch(NullPointerException expected){System.out.println("null-instance");}System.out.println(effects);try{System.out.println(throwing().value);}catch(IllegalStateException expected){System.out.println(expected.getMessage());}System.out.println(effects);Space.Holder value=null;System.out.println(value.value);System.out.println(pick(value.glyph));System.out.println(Space.Holder.value);}
}`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}
