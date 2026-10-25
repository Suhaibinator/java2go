package transpiler

import "testing"

func TestAbstractListScalarSourceStorageJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-one</artifactId><version>1</version></project>`,
		"src/main/java/probe/OneMain.java": `package probe;import java.util.*;
  @SuppressWarnings({"rawtypes","unchecked"}) public class OneMain{
   static class One<E> extends AbstractList<E>{E value;int reads,writes;One(E value){this.value=value;}public int size(){return 1;}public E get(int index){reads++;if(index!=0)throw new IndexOutOfBoundsException("one");return value;}public E set(int index,E value){writes++;if(index!=0)throw new IndexOutOfBoundsException("one");E old=this.value;this.value=value;return old;}}
   public static void main(String[]args){One<String> one=new One<>("a");List raw=one;raw.set(0,Integer.valueOf(7));System.out.println("one:"+raw.get(0)+":"+one.writes);try{String text=one.get(0);System.out.println("bad:"+text);}catch(ClassCastException e){System.out.println("cast:"+one.reads);}one.set(0,"a");System.out.println("restored:"+one.get(0)+":"+one.writes+":"+one.reads);}
  }`}, "probe.OneMain", "one:7:1\ncast:2\nrestored:a:2:3\n")
}
