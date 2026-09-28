package transpiler

import "testing"

func TestCampaignQualifiedStaticCallAndShadow(t *testing.T) {
	files := map[string]string{
		"pom.xml":                           `<project><groupId>example</groupId><artifactId>qualified</artifactId><version>1</version></project>`,
		"src/main/java/tools/Utility.java":  `package tools; public class Utility { public static int value(int n){return n*2;} public static String value(String s){return "static:"+s;} }`,
		"src/main/java/example/Holder.java": `package example; public class Holder { public Worker Utility=new Worker(); }`,
		"src/main/java/example/Worker.java": `package example; public class Worker {public int value(int n){return n+7;}}`,
		"src/main/java/example/Main.java":   `package example; public class Main {public static void main(String[] args){System.out.println(tools.Utility.value(2));System.out.println(tools.Utility.value("ok"));{Holder tools=new Holder();System.out.println(tools.Utility.value(2));}}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "4\nstatic:ok\n9\n")
}
