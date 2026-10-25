package transpiler

import "testing"

func TestAbstractListInheritedClearCallsSourceRemoveRangeJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-clear</artifactId><version>1</version></project>`,
		"src/main/java/probe/ClearMain.java": `package probe;import java.util.*;
 public class ClearMain{static class Values extends AbstractList<String>{ArrayList<String>data=new ArrayList<>();int calls;Values(){data.add("a");data.add("b");}public int size(){return data.size();}public String get(int index){return data.get(index);}protected void removeRange(int from,int to){calls++;data.subList(from,to).clear();}}
 public static void main(String[]args){Values values=new Values();List<String>alias=values;alias.clear();System.out.println("clear:"+values.calls+":"+values.size());}}
 `}, "probe.ClearMain", "clear:1:0\n")
}

func TestAbstractListSignatureFreeHelperKeepsErasedBodyJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-body-only</artifactId><version>1</version></project>`,
		"src/main/java/probe/BodyMain.java": `package probe;import java.util.*;
 @SuppressWarnings({"rawtypes","unchecked"}) public class BodyMain{static class One<E> extends AbstractList<E>{E value;private void poison(Object input){value=(E)input;}public int size(){return 1;}public E get(int index){poison("polluted");return value;}}
 public static void main(String[]args){One<Integer>typed=new One<>();List raw=typed;Object value=raw.get(0);System.out.println("raw:"+(value instanceof String));try{Integer integer=typed.get(0);System.out.println("bad:"+integer);}catch(ClassCastException e){System.out.println("typed:true");}}}
 `}, "probe.BodyMain", "raw:true\ntyped:true\n")
}
