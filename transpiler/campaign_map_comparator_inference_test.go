package transpiler

import "testing"

func TestCampaignMapComputeDiamondTarget(t *testing.T) {
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>map-diamond</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;import java.util.*;public class Main {public static void main(String[] args){Map<String,List<String>> groups=new TreeMap<>();groups.computeIfAbsent("a",key->new ArrayList<>()).add("one");groups.computeIfAbsent("a",key->new ArrayList<>()).add("two");System.out.println(groups.get("a").size());for(String value:groups.get("a"))System.out.println(value);}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "2\none\ntwo\n")
}

func TestCampaignComparatorMethodReferenceChain(t *testing.T) {
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>comparator-references</artifactId><version>1</version></project>`,
		"src/main/java/model/Row.java":    `package model;public record Row(String id,int priority) {}`,
		"src/main/java/example/Main.java": `package example;import java.util.*;import model.Row;public class Main {public static void main(String[] args){List<Row> rows=new ArrayList<>();rows.add(new Row("c",1));rows.add(new Row("b",2));rows.add(new Row("a",2));Comparator<Row> comparator=Comparator.comparingInt(Row::priority).reversed().thenComparing(Row::id);rows.sort(comparator);for(Row row:rows)System.out.println(row.id());}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "a\nb\nc\n")
}
