package transpiler

import "testing"

func TestAbstractListSourceOverridesAndShadowsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-controls</artifactId><version>1</version></project>`,
		"src/main/java/probe/Controls.java": `package probe;import java.util.*;
  public class Controls {
   static class Business extends AbstractList<Object> implements RandomAccess {
    int calls;public Object get(int index){if(index==0)return "a";throw new IndexOutOfBoundsException("one");}public int size(){return 1;}
    public Object set(int index,long value){calls++;return "business";}public boolean add(long value){calls++;return true;}
    int AbstractListStateJava2go(){return 19;}
   }
   static class Shadow extends AbstractList<Object>{
    int modCount=17;public Object get(int index){if(index==0)return "x";throw new IndexOutOfBoundsException("one");}public int size(){return 1;}
    void change(){modCount++;}int baseGeneration(){return super.modCount;}
   }
   static class Chain extends AbstractList<String>{String value;Chain(){this("c");}Chain(String value){this.value=value;}public String get(int index){if(index==0)return value;throw new IndexOutOfBoundsException("one");}public int size(){return 1;}}
   public static void main(String[]args){
    Business business=new Business();List<Object> alias=business;RandomAccess marker=business;Object object=marker;
    System.out.println("slots:"+business.set(0,7L)+":"+business.add(8L)+":"+business.calls+":"+business.AbstractListStateJava2go());
    try{alias.set(0,"q");System.out.println("bad-set");}catch(UnsupportedOperationException e){System.out.println("set-default:"+(e.getMessage()==null)+":"+business.calls);}
    try{alias.add("q");System.out.println("bad-add");}catch(UnsupportedOperationException e){System.out.println("add-default:"+(e.getMessage()==null)+":"+business.calls);}
    System.out.println("marker:"+(marker==business)+":"+(object instanceof RandomAccess)+":"+(object instanceof List));
    Shadow shadow=new Shadow();Iterator<Object> it=shadow.iterator();shadow.change();System.out.println("shadow:"+it.next()+":"+shadow.modCount+":"+shadow.baseGeneration());
    Chain chain=new Chain();System.out.println("chain:"+chain.get(0)+":"+chain.iterator().next());
   }
  }`}, "probe.Controls", "slots:business:true:2:19\nset-default:true:2\nadd-default:true:2\nmarker:true:true:true\nshadow:x:18:0\nchain:c:c\n")
}
