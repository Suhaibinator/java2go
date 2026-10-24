package transpiler

import "testing"

func TestCampaignClassNameMethodReferenceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>class-name-ref</artifactId><version>1</version></project>`,
		"src/main/java/owned/Class.java": `package owned; public class Class {
 private final String name="source";
 public String getName(){return name;}
}`,
		"src/main/java/probe/Main.java": `package probe; import owned.Class;
public class Main {
 public static void main(String[] args){
  java.lang.Class<?> type=String.class;String first=type.getName();
  java.util.function.Supplier<String> bound=type::getName;
  java.util.function.Function<java.lang.Class<?>,String> unbound=java.lang.Class::getName;
  System.out.println("bound="+bound.get()+":"+(bound.get()==first)+":"+(bound.get()==bound.get()));
  System.out.println("unbound="+unbound.apply(type)+":"+(unbound.apply(type)==first)+":"+(unbound.apply(type)==unbound.apply(type)));
  Class source=new Class();java.util.function.Supplier<String> owned=source::getName;
  System.out.println("source="+owned.get()+":"+(owned.get()==source.getName()));
  java.lang.Class<?> absent=null;boolean boundNull=false,unboundNull=false;
  try{java.util.function.Supplier<String> bad=absent::getName;}catch(NullPointerException expected){boundNull=true;}
  try{unbound.apply(absent);}catch(NullPointerException expected){unboundNull=true;}
  System.out.println("null="+boundNull+":"+unboundNull);
 }
}`,
	}, "probe.Main", "bound=java.lang.String:true:true\nunbound=java.lang.String:true:true\nsource=source:true\nnull=true:true\n")
}
