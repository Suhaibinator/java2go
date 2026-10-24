package transpiler

import "testing"

func TestCampaignInheritedInterfaceOverloads(t *testing.T) {
	files := map[string]string{
		"pom.xml":                           `<project><groupId>example</groupId><artifactId>overloads</artifactId><version>1</version></project>`,
		"src/main/java/example/Root.java":   `package example; public interface Root { Object decode(Object source); }`,
		"src/main/java/example/Binary.java": `package example; public interface Binary extends Root { byte[] decode(byte[] source); }`,
		"src/main/java/example/Text.java":   `package example; public interface Text extends Root { String decode(String source); }`,
		"src/main/java/example/Codec.java":  `package example; public class Codec implements Binary, Text { public Object decode(Object source) { return "object"; } public byte[] decode(byte[] source) { return source; } public String decode(String source) { return "text"; } }`,
		"src/main/java/example/Main.java":   `package example; public class Main { public static void main(String[] args) { Root root=new Codec(); Binary binary=new Codec(); Text text=new Codec(); System.out.println(root.decode("x")); System.out.println(binary.decode(new byte[]{4})[0]); System.out.println(text.decode("x")); } }`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "object\n4\ntext\n")
}
