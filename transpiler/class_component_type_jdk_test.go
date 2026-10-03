package transpiler

import "testing"

func TestClassComponentTypeCanonicalDescriptorJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 static int initialized,calls;
 static class Source {static int value=initialize();}
 static class Nest {static class Leaf {}}
 enum Choice {A}
 static int initialize(){initialized++;return 1;}
 static java.lang.Class<?> absent(){calls++;return null;}
 public static void main(String[]args) {
  int[] primitive=new int[0];java.lang.Class<?> typed=primitive.getClass();
  System.out.println("primitive="+typed.getComponentType().getName()+":"+(typed.getComponentType()==int.class));
  System.out.println("reference="+new String[0].getClass().getComponentType().getName()+":"+(new String[0].getClass().getComponentType()==String.class));
  java.lang.Class<?> nested=new String[0][0].getClass();
  System.out.println("rank="+(nested.getComponentType()==String[].class)+":"+(nested.getComponentType().getComponentType()==String.class));
  java.lang.Class<?> primitiveNested=new int[0][0].getClass();
  System.out.println("primitiveRank="+(primitiveNested.getComponentType()==int[].class)+":"+(primitiveNested.getComponentType().getComponentType()==int.class));
  System.out.println("source="+(new Source[0].getClass().getComponentType()==Source.class)+":"+initialized);
  System.out.println("inner="+new Nest.Leaf[0].getClass().getComponentType().getName()+":"+(new Nest.Leaf[0].getClass().getComponentType()==Nest.Leaf.class));
  System.out.println("enum="+(new Choice[0].getClass().getComponentType()==Choice.class));
  System.out.println("ordinary="+(String.class.getComponentType()==null)+":"+(int.class.getComponentType()==null)+":"+(void.class.getComponentType()==null));
  java.util.function.Supplier<java.lang.Class<?>> bound=typed::getComponentType;
  java.util.function.Function<java.lang.Class<?>,java.lang.Class<?>> unbound=java.lang.Class::getComponentType;
  System.out.println("references="+(bound.get()==int.class)+":"+(unbound.apply(typed)==int.class));
  try {absent().getComponentType();System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+calls);}
 }
}`, "primitive=int:true\nreference=java.lang.String:true\nrank=true:true\nprimitiveRank=true:true\nsource=true:0\ninner=Main$Nest$Leaf:true\nenum=true\nordinary=true:true:true\nreferences=true:true\nnull=1\n")
}

func TestClassComponentTypeSourceAndBinderOwnerJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>class-component-type</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `import owned.Class;
public class Main {
 static <Class extends owned.Class> String through(Class value){return value.getComponentType().getName();}
 public static void main(String[]args) {
  Class source=new Class();
  System.out.println("source="+(source.getComponentType()==source)+":"+source.getComponentType().getName());
  System.out.println("binder="+through(source));
  java.lang.Class<?> canonical=new String[0].getClass();
  System.out.println("canonical="+canonical.getComponentType().getName()+":"+(canonical.getComponentType()==String.class));
 }
}`,
		"src/main/java/owned/Class.java": `package owned;
public class Class {
 public Class getComponentType(){return this;}
 public String getName(){return "source";}
}`,
	}, "Main", "source=true:source\nbinder=source\ncanonical=java.lang.String:true\n")
}
