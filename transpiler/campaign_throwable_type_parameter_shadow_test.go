package transpiler

import "testing"

func TestCampaignThrowableTypeParameterShadowJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>throwable-binder</artifactId><version>1</version></project>`,
		"src/main/java/p/Main.java": `package p; public class Main {
 static String choose(Object value){return "object";}
 static String choose(Throwable value){return "throwable";}
 static <ClassNotFoundException> String call(ClassNotFoundException value){return choose(value);}
 public static void main(String[] args){System.out.println(call("text"));}
 }`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "p.Main", "object\n")
}
