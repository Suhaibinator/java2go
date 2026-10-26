package transpiler

import "testing"

func TestCampaignCollectionExportPipeline(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>export-pipeline</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.*;import java.util.stream.Collectors;
class Row {String id,label;int balance;Row(String id,String label,int balance){this.id=id;this.label=label;this.balance=balance;}}
public class Main {public static void main(String[] args){
 Map<String,Row> map=new LinkedHashMap<>();map.put("b",new Row("b","z",2));map.put("a",new Row("a","a",3));map.put("c",new Row("c","z",4));
 List<Row> rows=map.values().stream().sorted(Comparator.comparing(row->row.id)).collect(Collectors.toList());
 for(Row row:rows){System.out.println(row.id);}
 Map<String,Integer> totals=rows.stream().collect(Collectors.groupingBy(row->row.label,TreeMap::new,Collectors.summingInt(row->row.balance)));
 for(Map.Entry<String,Integer> entry:totals.entrySet()){System.out.println(entry.getKey()+":"+entry.getValue());}
 rows.stream().sorted(Comparator.comparing((Row row)->row.label).thenComparing(row->row.id)).forEach(row->System.out.println(row.id));
 Map<String,String> late=new LinkedHashMap<>();late.put("x","x");Collection<String> view=late.values();late.put("y","y");System.out.println(view.stream().count());
 Map<String,Integer> descending=rows.stream().collect(Collectors.groupingBy(row->row.label,()->new TreeMap<String,Integer>(Comparator.reverseOrder()),Collectors.summingInt(row->row.balance)));for(String key:descending.keySet()){System.out.println(key);}

}}
`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "a\nb\nc\na:3\nz:6\na\nb\nc\n2\nz\na\n")
}
