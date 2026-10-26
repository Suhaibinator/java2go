package transpiler

import "testing"

// Value declarations do not shadow a package name in Java TYPE syntax.
func TestCampaignWrapperConstantsTypeNamespaceJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>namespace</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;public class Main {
 static java.lang.Long holder;
 static java.lang.Long[] values=new java.lang.Long[1];
 static Object token(int java){return java.lang.Long.class;}
 static <Long> int bound(){return java.lang.Long.BYTES;}
 public static void main(String[] args){System.out.println(token(5)==java.lang.Long.class);System.out.println(holder==null);System.out.println(values.length);System.out.println(bound());}
}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}

func TestCampaignWrapperConstantsRootTypeShadows(t *testing.T) {
	for _, source := range []string{
		`class Probe{static class java{static class lang{static class Long{static int BYTES=37;}}}static int read(){return java.lang.Long.BYTES;}}`,
		`class Probe<java>{int read(){return java.lang.Long.BYTES;}}`,
	} {
		helper := setupParseHelper(t, source)
		ctx := helper.Ctx
		for _, method := range ctx.currentClass.Methods {
			if method.OriginalName == "read" {
				ctx.localScope = method
			}
		}
		node := findNode(helper.File.Ast, "return_statement").NamedChild(0)
		if owner, accepted := intrinsicStaticClassName(node.ChildByFieldName("object"), ctx, helper.File.Source); accepted {
			t.Fatalf("source/root type shadow admitted canonical owner %q", owner)
		}
		if _, accepted := tryStaticFieldIntrinsic(node.ChildByFieldName("object"), node.ChildByFieldName("field").Content(helper.File.Source), helper.File.Source, ctx); accepted {
			t.Fatal("a source/root type binder cannot borrow canonical wrapper constants")
		}
	}
}

func TestCampaignWrapperConstantsSourceRootJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>wrapperfacts</groupId><artifactId>source-root</artifactId><version>1</version></project>
`,
		"src/main/java/app/Main.java": `package app;public class Main{static class java{static class lang{static class Long{static int BYTES=37;}}}public static void main(String[] args){System.out.println(java.lang.Long.BYTES);}}
`,
	}
	campaignWrapperConstantsStrictProject(t, files)
}
