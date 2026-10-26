package transpiler

import "testing"

func TestCampaignCollectionViewsAndCopy(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>views</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.Map;import java.util.LinkedHashMap;import java.util.List;import java.util.ArrayList;
public class Main {
 static Iterable<Integer> values(Map<String,Integer> map){return map.values();}
 static int sum(Iterable<Integer> values){int result=0;for(Integer value:values){result+=value;}return result;}
 public static void main(String[] args){
  Map<String,Integer> map=new LinkedHashMap<>();map.put("a",2);map.put("b",3);
  for(Map.Entry<String,Integer> entry:map.entrySet()){System.out.println(entry.getKey()+":"+entry.getValue());}
  System.out.println(sum(values(map)));
  List<Integer> source=new ArrayList<>(4);source.add(7);List<Integer> copy=new ArrayList<>(source);Iterable<Integer> view=source;source.add(8);
  System.out.println(sum(view));System.out.println(copy.size());System.out.println(copy.get(0));
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "a:2\nb:3\n5\n15\n1\n7\n")
}
