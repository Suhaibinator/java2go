package transpiler

import "testing"

func TestCampaignArraysAsListArray(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>arrays-list</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.Arrays;import java.util.List;import java.util.Collections;
class Item { int value; Item(int value){this.value=value;} }
public class Main {
 static int sum(Item... values){List<Item> list=Arrays.asList(values);int sum=0;for(Item item:list){sum+=item.value;}return sum;}
 public static void main(String[] args){
  System.out.println(sum(new Item(2),new Item(3)));
  String[] values={"b","a"};List<String> list=Arrays.asList(values);values[0]="c";System.out.println(list.get(0));
  System.out.println(list.set(1,"d"));System.out.println(values[1]);Collections.reverse(list);System.out.println(values[0]);
  Collections.sort(list);System.out.println(values[0]);System.out.println(list.size());
  int[] primitive={4,5};List<int[]> singleton=Arrays.asList(primitive);System.out.println(singleton.size());System.out.println(singleton.get(0)[1]);
  String[][] nested={{"x"},{"y"}};List<String[]> rows=Arrays.asList(nested);System.out.println(rows.size());System.out.println(rows.get(1)[0]);
  int visited=0;for(String value:list){System.out.println(value);if(visited++==0){values[1]="changed";}}
  List<String[]> boxedArray=Arrays.<String[]>asList(values);System.out.println(boxedArray.size());
  List<Object> broad=Arrays.asList(values);try{broad.set(0,new Item(9));}catch(ArrayStoreException failure){System.out.println("checked");}
  try {list.add("z");}catch(UnsupportedOperationException failure){System.out.println("fixed");}
  try {Arrays.asList((String[])null);}catch(NullPointerException failure){System.out.println("null");}
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "5\nc\na\nd\nd\nc\n2\n1\n5\n2\ny\nc\nchanged\n1\nchecked\nfixed\nnull\n")
}
