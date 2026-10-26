package transpiler

import "testing"

func TestCampaignInheritedObjectTextErasedHierarchyJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>erased-text</artifactId><version>1</version></project>`,
		"src/main/java/probe/Base.java": `package probe;
public class Base { public int hashes; @Override public int hashCode(){hashes++;return 42;} }
`,
		"src/main/java/probe/Child.java": `package probe;
public final class Child extends Base {}
`,
		"src/main/java/probe/ErasedProbe.java": `package probe;
public final class ErasedProbe {
 static Object relay(Base value){return value;}
 static String render(Object value){return String.valueOf(value);}
 public static void main(String[] args){
  Child child=new Child();Base base=child;Object erased=relay(base);
  System.out.println(render(erased));
  System.out.println("concat="+erased);
  System.out.println(base.toString());
  System.out.println(child.hashes);
 }
}
`,
	}, "probe.ErasedProbe", "probe.Child@2a\nconcat=probe.Child@2a\nprobe.Child@2a\n3\n")
}
