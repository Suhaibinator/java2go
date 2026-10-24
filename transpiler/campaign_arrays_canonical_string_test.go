package transpiler

import "testing"

func TestCampaignArraysCanonicalStringJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>arrayprobe</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/arrayprobe/Main.java": `package arrayprobe;
import java.util.Arrays;
public class Main {
 static class Text {
  String value;int calls,ordinary;boolean held;Object[] mutate;RuntimeException abrupt;
  Text(String value){this.value=value;}
  public String String(){ordinary++;return "impostor";}
  public synchronized String toString(){calls++;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;if(mutate!=null)mutate[1]="new";return value;}
 }
 static String units(String value){String out="";for(int i=0;i<value.length();i++)out+=(int)value.charAt(i)+",";return out;}
 public static void main(String[] args){
  int[] empty=new int[0],absent=null;Object[] objects=new Object[0];
  System.out.println((Arrays.toString(absent)=="null")+":"+(Arrays.toString(empty)=="[]")+":"+(Arrays.toString(objects)=="[]")+":"+(Arrays.deepToString(objects)!=Arrays.deepToString(objects)));
  int[] ints={65,-2};String first=Arrays.toString(ints),second=Arrays.toString(ints);System.out.println(first+":"+(first!=second)+":"+first.equals(second));
  System.out.println(Arrays.toString(new boolean[]{true,false}));System.out.println(Arrays.toString(new byte[]{-128,127}));System.out.println(Arrays.toString(new short[]{-32768}));System.out.println(Arrays.toString(new long[]{-9223372036854775808L}));System.out.println(Arrays.toString(new float[]{-0.0f,Float.NaN,Float.POSITIVE_INFINITY}));System.out.println(Arrays.toString(new double[]{1.5,Double.NEGATIVE_INFINITY}));
  char[] chars={(char)0xD800,0,(char)0xDFFF};System.out.println(units(Arrays.toString(chars)));System.out.println(units(Arrays.deepToString(new Object[]{chars})));
  Text text=new Text("k");Object[] live={text,"old"};text.mutate=live;System.out.println(Arrays.toString(live)+":"+text.calls+":"+text.ordinary+":"+text.held);
  Text nullText=new Text(null);Object[] cycle={null,nullText};cycle[0]=cycle;System.out.println(Arrays.deepToString(cycle)+":"+nullText.calls+":"+nullText.held);
  RuntimeException marker=new IllegalStateException("marker");nullText.abrupt=marker;boolean same=false;try{Arrays.deepToString(cycle);}catch(RuntimeException got){same=got==marker;}nullText.abrupt=null;System.out.println(same+":"+Arrays.deepToString(cycle)+":"+nullText.calls);
  Object[] siblings={cycle,cycle,new int[][]{{1,2},{3}}};System.out.println(Arrays.deepToString(siblings));
  String identity="[I@"+Integer.toHexString(System.identityHashCode(ints));System.out.println(String.valueOf(ints).equals(identity)+":"+Arrays.toString(new Object[]{ints}).equals("["+identity+"]"));
 }
}`}, "arrayprobe.Main", "true:true:true:true\n[65, -2]:true:true\n[true, false]\n[-128, 127]\n[-32768]\n[-9223372036854775808]\n[-0.0, NaN, Infinity]\n[1.5, -Infinity]\n91,55296,44,32,0,44,32,57343,93,\n91,91,55296,44,32,0,44,32,57343,93,93,\n[k, new]:1:0:true\n[[...], null]:1:true\ntrue:[[...], null]:3\n[[[...], null], [[...], null], [[1, 2], [3]]]\ntrue:true\n")
}
