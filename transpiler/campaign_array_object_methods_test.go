package transpiler

import "testing"

func TestCampaignArrayObjectMethods(t *testing.T) {
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>array-object-methods</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;public class Main {static int calls;static int[] make(){calls++;return new int[]{3};}public static void main(String[] args){String[] words={"a"};Object[] covariant=words;int[] numbers={1,2};int[][] nested={{1}};System.out.println(words.getClass()==String[].class);System.out.println(covariant.getClass()==String[].class);System.out.println(numbers.getClass()==int[].class);System.out.println(nested.getClass()==int[][].class);System.out.println(words.equals(covariant));System.out.println(words.equals(new String[]{"a"}));System.out.println(words.hashCode()==covariant.hashCode());System.out.println(make().getClass()==int[].class);System.out.println(calls);String[] missing=null;try{missing.getClass();System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null");}}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "true\ntrue\ntrue\ntrue\ntrue\nfalse\ntrue\ntrue\n1\nnull\n")
}
