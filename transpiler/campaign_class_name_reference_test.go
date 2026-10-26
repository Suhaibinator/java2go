package transpiler

import "testing"

func TestCampaignClassNameReferenceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>class-name</artifactId><version>1</version></project>`,
		"src/main/java/owned/Class.java": `package owned; public class Class {
 private final String name="source";
 public String getName(){return name;}
}`,
		"src/main/java/probe/Main.java": `package probe; import owned.Class;
public class Main {
 public static void main(String[] args){
  java.lang.Class<?> type=String.class,alias=type;
  String first=type.getName(),second=alias.getName();Object object=first;
  System.out.println(first+":"+(first==second)+":"+(first==object)+":"+first.equals("java.lang.String"));
  java.lang.Class<?> array=String[].class,primitive=int.class;
  System.out.println(array.getName()+":"+(array.getName()==array.getName())+":"+primitive.getName()+":"+(primitive.getName()==primitive.getName()));
  Class source=new Class();String sourceName=source.getName();
  System.out.println(sourceName+":"+(sourceName==source.getName()));
  java.lang.Class<?> absent=null;boolean caught=false;try{absent.getName();}catch(NullPointerException expected){caught=true;}
  System.out.println("null="+caught);
 }
}`,
	}, "probe.Main", "java.lang.String:true:true:true\n[Ljava.lang.String;:true:int:true\nsource:true\nnull=true\n")
}
