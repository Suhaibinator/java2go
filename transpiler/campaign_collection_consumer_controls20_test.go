package transpiler

import (
	"os"
	"testing"
)

func TestCampaignCollectionConsumerControls20JVM(t *testing.T) {
	oracle, err := os.ReadFile("testdata/collection_consumer_controls20_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
public class Main {
 @SuppressWarnings({"rawtypes", "unchecked"})
 static void write(Collection target, Object value) { target.add(value); }
 @SuppressWarnings("rawtypes")
 static Object removeAt(List target, int index) { return target.remove(index); }
 @SuppressWarnings("rawtypes")
 static boolean removeValue(List target, Object value) { return target.remove(value); }
 static Object trimmed(List<String> target) { return target.get(0).trim(); }
 public static void main(String[] args) {
  List<String> values = new ArrayList<>(); write(values,Integer.valueOf(7));
  Object first = values.get(0); values.set(0,"changed");
  write(values,Integer.valueOf(8)); values.remove(1);
  System.out.println("consumers=" + first + ":" + values.get(0) + ":" + values.size());
  write(values,Integer.valueOf(9)); Object removed=removeAt(values,1);
  write(values,Integer.valueOf(10)); boolean erased=removeValue(values,Integer.valueOf(10));
  System.out.println("raw-remove="+removed+":"+erased+":"+values.size()+":"+trimmed(values));
  List<String> ordered = new ArrayList<>(); ordered.add("b"); ordered.add("a");
  write(ordered,"c"); Object alias=ordered;
  Collections.sort(ordered); System.out.println("sorted=" + ordered + ":" + (alias==ordered));
  Collections.reverse(ordered); System.out.println("reverse=" + ordered + ":" + (alias==ordered));
  Collection<String> collection=ordered; Iterable<String> iterable=collection;
  String order=""; for(String item:iterable) order=order+item;
  System.out.println("iterable=" + order + ":" + (iterable==collection));
  List<Integer> boxed=new ArrayList<>(); boxed.add(Integer.valueOf(2)); boxed.add(null);
  Collection<Integer> integers=boxed; int seen=0;
  try { for(int item:integers) seen=seen+item; } catch(NullPointerException expected) { System.out.println("unbox=" + seen); }
 }
}
`,
	}, "probe.Main", string(oracle))
}
func TestCampaignCollectionSourceObjectProjection20JVM(t *testing.T) {
	oracle, err := os.ReadFile("testdata/collection_source_object_projection20_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import probe.model.Object;
import probe.model.Holder;
import probe.flow.Readers;
public class Main {
 @SuppressWarnings({"rawtypes","unchecked"}) static void pollute(Collection target,java.lang.Object value){target.add(value);}
 public static void main(String[] args){
  Object value=new Object(7);List<Object> values=new ArrayList<>();values.add(value);
  System.out.println("source="+Readers.exact(values).value()+":"+(Readers.cast(values)==value));
  List<Holder> holders=new ArrayList<>();holders.add(new Holder(value));
  System.out.println("nested="+(Readers.nested(holders,0)==value)+":"+Holder.reads);
  pollute(holders,Integer.valueOf(9));
  try{Readers.nested(holders,1);System.out.println("unexpected");}catch(ClassCastException expected){System.out.println("nested-cast="+Holder.reads+":"+holders.size());}
 }
}
`,
		"src/main/java/probe/flow/Readers.java": `package probe.flow;
import java.util.List;
import probe.model.Object;
import probe.model.Holder;
public final class Readers {
 public static Object exact(List<Object> source){return source.get(0);}
 public static Object cast(List<Object> source){return (Object)source.get(0);}
 public static java.lang.Object nested(List<Holder> source,int index){return source.get(index).child();}
}
`,
		"src/main/java/probe/model/Holder.java": `package probe.model;
public final class Holder { public static int reads; private Object value; public Holder(Object value){this.value=value;} public Object child(){reads++;return value;} }
`,
		"src/main/java/probe/model/Object.java": `package probe.model;
public final class Object { private int value; public Object(int value){this.value=value;} public int value(){return value;} }
`,
	}, "probe.Main", string(oracle))
}
