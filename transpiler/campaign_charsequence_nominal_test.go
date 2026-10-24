package transpiler

import "testing"

func TestCampaignCharSequenceSourceNominalJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                        `<project><modelVersion>4.0.0</modelVersion><groupId>nominalprobe</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/api/Textual.java": `package api;public interface Textual extends java.lang.CharSequence {}`,
		"src/main/java/impl/Base.java": `package impl;public class Base implements api.Textual {
 final String text;public Base(String text){this.text=text;}
 public int length(){return text.length();}public char charAt(int index){return text.charAt(index);}
 public CharSequence subSequence(int from,int to){return text.substring(from,to);}
 public String toString(){return text;}
}`,
		"src/main/java/impl/Child.java":          `package impl;public class Child extends Base {public Child(String text){super(text);}}`,
		"src/main/java/shadow/CharSequence.java": `package shadow;public interface CharSequence {}`,
		"src/main/java/shadow/Decoy.java":        `package shadow;public class Decoy implements CharSequence {public String toString(){return "decoy";}}`,
		"src/main/java/app/Main.java": `package app;import java.util.*;
public class Main {
 @SuppressWarnings({"rawtypes","unchecked"}) public static void main(String[] args){
  impl.Child child=new impl.Child("child");List<CharSequence> list=new ArrayList<>();list.add(child);list.add(new impl.Base("base"));
  System.out.println(String.join("|",list));
  Object[] array=new CharSequence[2];array[0]=child;array[1]=new impl.Base("array");
  System.out.println(String.join("/",(CharSequence[])array));
  System.out.println((array[0]==child)+":"+((CharSequence)array[0]).length());
  List raw=list;raw.clear();raw.add(new shadow.Decoy());
  try{String.join("|",list);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("decoy-rejected");}
 }
}`,
	}, "app.Main", "child|base\nchild/array\ntrue:5\ndecoy-rejected\n")
}
