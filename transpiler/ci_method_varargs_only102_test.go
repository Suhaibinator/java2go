package transpiler

import "testing"

func TestCI102MethodVarArgsOnlyJDK21(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>ci</groupId><artifactId>varargs</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;public class Main {
 public static void ordinary(String[] values){}
 public static void spread(String... values){}
 static class Method {boolean isVarArgs(){return false;}}
 static class Scope<Method> {boolean check(java.lang.reflect.Method value){return value.isVarArgs();}}
 public static void main(String[]args)throws Exception {
  java.lang.reflect.Method plain=Main.class.getDeclaredMethod("ordinary",String[].class);
  java.lang.reflect.Method variable=Main.class.getDeclaredMethod("spread",String[].class);
  System.out.println(plain.isVarArgs());System.out.println(variable.isVarArgs());
  System.out.println(new Scope<Integer>().check(variable));System.out.println(new Method().isVarArgs());
  java.lang.reflect.Method absent=null;
  try {absent.isVarArgs();System.out.println("null-missed");}catch(NullPointerException e){System.out.println("null");}
 }
}`}
	runCampaignCompilerStrictProjectOracle(t, files, "probe.Main", "false\ntrue\ntrue\nfalse\nnull\n")
}
