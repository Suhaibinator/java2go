package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbstractListActualGsonBulkAndSourceInheritanceJDK21(t *testing.T) {
	wrapper, err := os.ReadFile(filepath.Join("testdata", "gson_generic_parent_origin", "dependencies", "src", "com", "google", "gson", "internal", "NonNullElementWrapperList.java"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-bulk-inherited</artifactId><version>1</version></project>`,
		"src/main/java/com/google/gson/internal/NonNullElementWrapperList.java": string(wrapper),
		"src/main/java/probe/BulkMain.java": `package probe;import java.util.*;import com.google.gson.internal.NonNullElementWrapperList;
 public class BulkMain{
  static class Base<E> extends AbstractList<E>{E value;int reads;Base(E value){this.value=value;}public int size(){return 1;}public E get(int index){reads++;if(index!=0)throw new IndexOutOfBoundsException("one");return value;}public E set(int index,E next){E old=get(index);value=next;return old;}}
  static class Leaf extends Base<String>{Leaf(){super("base");}public String get(int index){return "leaf:"+super.get(index);}}
  public static void main(String[]args){ArrayList<String>d=new ArrayList<>();d.add("a");d.add("b");d.add("c");List<String>w=new NonNullElementWrapperList<>(d);
   System.out.println("bulk:"+w.addAll(Arrays.asList("d","e"))+":"+w);
   String[]target=new String[7];target[6]="tail";System.out.println("array:"+(w.toArray(target)==target)+":"+target[0]+":"+(target[5]==null)+":"+target[6]);
   System.out.println("filter:"+w.removeAll(Arrays.asList("b","d"))+":"+w.retainAll(Arrays.asList("a","e"))+":"+w);
   try{w.addAll(Arrays.asList("x",null,"y"));System.out.println("bad-null");}catch(NullPointerException e){System.out.println("partial:"+w+":"+e.getMessage());}
   int count=0;for(String value:w)count+=value.length();System.out.println("scan:"+count+":"+w.containsAll(Arrays.asList("a","x"))+":"+w.lastIndexOf("a"));
   Iterator<String>it=w.iterator();w.clear();try{it.next();System.out.println("bad-clear");}catch(NoSuchElementException e){System.out.println("clear:"+w.size()+":"+(e.getCause() instanceof IndexOutOfBoundsException));}
   Leaf leaf=new Leaf();List<String>alias=leaf;String first=alias.get(0);String second=alias.iterator().next();int reads=leaf.reads;String old=alias.set(0,"next");System.out.println("inherit:"+first+":"+second+":"+reads+":"+old+":"+alias.get(0));
  }
 }`}, "probe.BulkMain", "bulk:true:[a, b, c, d, e]\narray:true:a:true:tail\nfilter:true:true:[a, e]\npartial:[a, e, x]:Element must be non-null\nscan:3:true:0\nclear:0:true\ninherit:leaf:base:leaf:base:2:leaf:base:leaf:next\n")
}
