package transpiler

import "testing"

func TestCampaignQualifiedInterfaceNullReturns(t *testing.T) {
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>interface-null</artifactId><version>1</version></project>`,
		"src/main/java/api/Plain.java":    `package api;public interface Plain {}`,
		"src/main/java/api/Box.java":      `package api;public interface Box<T> {}`,
		"src/main/java/example/Main.java": `package example;import api.Plain;import api.Box;import java.lang.reflect.Type;public class Main {static Plain plain(){return null;}static Box<String> box(){return null;}static Type type(){return null;}public static void main(String[] args){System.out.println(plain()==null);System.out.println(box()==null);System.out.println(type()==null);}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "true\ntrue\ntrue\n")
}
