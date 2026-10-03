package transpiler

import "testing"

func TestCampaignBuiltinInterfaceInheritance(t *testing.T) {
	files := map[string]string{
		"pom.xml":                          `<project><groupId>example</groupId><artifactId>supplier</artifactId><version>1</version></project>`,
		"src/main/java/example/Value.java": `package example; import java.util.function.Supplier; public interface Value<T> extends Supplier<T> { T read(); default T get() { return read(); } }`,
		"src/main/java/example/Only.java":  `package example; public interface Only<T> extends java.util.function.Supplier<T> {}`,
		"src/main/java/example/Impl.java":  `package example; public class Impl implements Value<String> { public String read() { return "ready"; } }`,
		"src/main/java/example/Other.java": `package example; public class Other implements Only<String> { public String get() { return "ready"; } }`,
		"src/main/java/example/Main.java":  `package example; public class Main { public static void main(String[] args) { Value<String> value = new Impl(); System.out.println(value.get()); Only<String> only = new Other(); System.out.println(only.get()); } }`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "ready\nready\n")
}
