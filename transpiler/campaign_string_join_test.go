package transpiler

import "testing"

func TestCampaignStringJoinOverloadsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>joinprobe</groupId><artifactId>probe</artifactId><version>1</version></project>`, "src/main/java/joinprobe/Main.java": `package joinprobe;
import java.util.*;
public class Main {
 static String trace="";
 static final class Text implements CharSequence {
  final String id;final String value;final boolean abrupt;
  Text(String id,String value){this.id=id;this.value=value;this.abrupt=false;}
  Text(String id,String value,boolean abrupt){this.id=id;this.value=value;this.abrupt=abrupt;}
  public int length(){return value.length();}
  public char charAt(int index){return value.charAt(index);}
  public CharSequence subSequence(int start,int end){return value.substring(start,end);}
  public String toString(){trace+=id;if(abrupt)throw new IllegalStateException(id);return value;}
 }
 static CharSequence delimiter(){trace+="d";return new Text("D","|");}
 static List<CharSequence> values(){trace+="v";List<CharSequence> result=new ArrayList<>();result.add(new Text("A","a"));result.add(null);result.add(new Text("B","b"));return result;}
 public static void main(String[] args){
  List<String> words=new ArrayList<>();words.add("alpha");words.add(null);words.add("omega");
  System.out.println(String.join("/",words));
  System.out.println("empty="+String.join("/",new ArrayList<String>()));
  System.out.println(String.join("/",new String[]{"x",null,"y"}));
  System.out.println(String.join("/","x",null,"y"));
  System.out.println("zero="+String.join("/"));
  System.out.println("single="+String.join("/","x"));
  trace="";System.out.println(String.join(delimiter(),values())+":"+trace);
  Iterable<CharSequence> absent=null;CharSequence[] absentArray=null;
  trace="";try{String.join(new Text("D","|"),absent);}catch(NullPointerException e){System.out.println("iter-null:"+trace);}
  trace="";try{String.join(new Text("D","|"),absentArray);}catch(NullPointerException e){System.out.println("array-null:"+trace);}
  trace="";List<CharSequence> bad=new ArrayList<>();bad.add(new Text("A",null));bad.add(new Text("B","b"));
  try{String.join(new Text("D","|"),bad);}catch(NullPointerException e){System.out.println("null-text:"+trace);}
  trace="";bad.clear();bad.add(new Text("A","a",true));bad.add(new Text("B","b"));
  try{String.join(new Text("D","|"),bad);}catch(IllegalStateException e){System.out.println("abrupt:"+e.getMessage()+":"+trace);}
 }
}
`}, "joinprobe.Main", "alpha/null/omega\nempty=\nx/null/y\nx/null/y\nzero=\nsingle=x\na|null|b:dvDAB\niter-null:\narray-null:D\nnull-text:DAB\nabrupt:A:DA\n")
}

func TestCampaignStringJoinIterationMutationJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>joinprobe</groupId><artifactId>probe</artifactId><version>1</version></project>`, "src/main/java/joinprobe/Main.java": `package joinprobe;
import java.util.*;
public class Main {
 static List<CharSequence> values;
 static String trace="";
 static final class Text implements CharSequence {
  final boolean structural;
  Text(boolean structural){this.structural=structural;}
  public int length(){return 1;}
  public char charAt(int index){return 'a';}
  public CharSequence subSequence(int start,int end){return "a".substring(start,end);}
  public String toString(){
   trace+="A";
   if(structural){values.add("extra");values.remove(values.size()-1);}
   else{values.set(1,"changed");}
   return "a";
  }
 }
 static final class Delimiter implements CharSequence {
  public int length(){return 1;}
  public char charAt(int index){return '|';}
  public CharSequence subSequence(int start,int end){return "|".substring(start,end);}
  public String toString(){trace+="D";values.add("third");return "|";}
 }
 public static void main(String[] args){
  values=new ArrayList<>();values.add(new Text(false));values.add("old");
  System.out.println("replace="+String.join("|",values)+":"+trace);
  trace="";values.clear();values.add(new Text(true));values.add("old");
  try{String.join("|",values);System.out.println("wrong");}catch(ConcurrentModificationException e){System.out.println("structural:"+trace+":"+values.size());}
  trace="";values.clear();values.add("first");values.add("second");
  System.out.println("delimiter="+String.join(new Delimiter(),values)+":"+trace);
 }
}
`}, "joinprobe.Main", "replace=a|changed:A\nstructural:A:2\ndelimiter=first|second|third:D\n")
}
