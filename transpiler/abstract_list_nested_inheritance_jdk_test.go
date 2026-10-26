package transpiler

import "testing"

func TestAbstractListMultipleGenericAncestorsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-list-ancestors</artifactId><version>1</version></project>`,
		"src/main/java/probe/AncestorMain.java": `package probe;import java.util.*;
 public class AncestorMain{
  static class Base<E> extends AbstractList<E>{E value;int reads;Base(E value){this.value=value;}public int size(){return 1;}public E get(int index){reads++;return value;}}
  static class Middle<F> extends Base<F>{int extras;Middle(F value){super(value);}public F get(int index){extras++;return super.get(index);}}
  static class Leaf extends Middle<String>{Leaf(){super("base");}public String get(int index){return "deep:"+super.get(index);}}
  public static void main(String[]args){Leaf leaf=new Leaf();List<String>alias=leaf;System.out.println(alias.get(0)+":"+leaf.reads+":"+leaf.extras);}
 }`}, "probe.AncestorMain", "deep:base:1:1\n")
}
