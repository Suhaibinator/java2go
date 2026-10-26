package transpiler

import "testing"

func TestCampaignMapPutIfAbsent(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>maps</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.Map; import java.util.LinkedHashMap;
public class Main {
 public static void main(String[] args) {
  Map<String,String> values = new LinkedHashMap<>();
  System.out.println(values.putIfAbsent("first", "one"));
  System.out.println(values.putIfAbsent("first", "ignored"));
  values.put("second", null);
  System.out.println(values.putIfAbsent("second", "two"));
  System.out.println(values.putIfAbsent("third", null));
  System.out.println(values.containsKey("third"));
  System.out.println(values.putIfAbsent("third", "three"));
  System.out.println(values.get("first") + ":" + values.get("second") + ":" + values.get("third"));
  System.out.println(values.size());
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "null\none\nnull\nnull\ntrue\nnull\none:two:three\n3\n")
}
