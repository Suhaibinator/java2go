package transpiler

import "testing"

// A bound contributes nominal conversion and overload specificity even when
// the actual argument's spelling is a type parameter rather than a class name.
func TestCampaignCharSequenceSourceBoundsReviewJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                        `<project><modelVersion>4.0.0</modelVersion><groupId>boundreview</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/api/Textual.java": `package api; public interface Textual extends java.lang.CharSequence {}`,
		"src/main/java/impl/Base.java": `package impl; public class Base implements api.Textual {
 public int length(){return 4;} public char charAt(int index){return "text".charAt(index);}
 public java.lang.CharSequence subSequence(int start,int end){return "text".substring(start,end);}
 public String toString(){return "text";}
}`,
		"src/main/java/app/Main.java": `package app;
public class Main {
 static String choose(Object value){return "object";}
 static String choose(java.lang.CharSequence value){return value==null?"text-null":"text";}
 static <T extends impl.Base> String classBound(T value){return choose(value);}
 static <T extends api.Textual> String interfaceBound(T value){return choose(value);}
 static <T extends api.Textual,U extends T> String chainedBound(U value){return choose(value);}
 static <CharSequence extends api.Textual> String shadowBound(CharSequence value){return choose(value);}
 static <T> String unbounded(T value){return choose(value);}
 public static void main(String[] args){
  impl.Base text=new impl.Base();
  System.out.println(classBound(text)+":"+interfaceBound(text)+":"+chainedBound(text)+":"+shadowBound(text));
  System.out.println(classBound(null)+":"+interfaceBound(null)+":"+chainedBound(null)+":"+shadowBound(null));
  System.out.println(unbounded(text));
 }
}`,
	}, "app.Main", "text:text:text:text\ntext-null:text-null:text-null:text-null\nobject\n")
}

func TestCharSequenceBoundAssignableReview(t *testing.T) {
	helper := setupParseHelper(t, `interface Textual extends java.lang.CharSequence {}
 class Scope<T extends Textual> {
  <U extends T> void call(U value) {}
 }`)
	ctx := helper.Ctx
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Scope")
	if ctx.currentClass == nil {
		t.Fatal("missing test class Scope")
	}
	if !javaInferenceTypeAssignable("T", "java.lang.CharSequence", ctx) {
		t.Fatal("class binder's source-subinterface bound did not reach canonical CharSequence")
	}
	ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
	if !javaInferenceTypeAssignable("U", "java.lang.CharSequence", ctx) {
		t.Fatal("method binder's transitive bound did not reach canonical CharSequence")
	}
}
