package transpiler

import "testing"

func TestCampaignObjectTextDescriptorMethodCollisionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>method-name</artifactId><version>1</version></project>`,
		"src/main/java/probe/Leaf.java": `package probe;
public final class Leaf { public int JavaDynamicTypeID(){return 7;} }
`,
		"src/main/java/probe/MethodProbe.java": `package probe;
public final class MethodProbe {public static void main(String[] args){System.out.println(new Leaf().JavaDynamicTypeID());}}
`,
	}, "probe.MethodProbe", "7\n")
}

func TestCampaignObjectTextDescriptorFieldCollisionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>field-name</artifactId><version>1</version></project>`,
		"src/main/java/probe/Leaf.java": `package probe;
public final class Leaf { public int JavaDynamicTypeID=9; }
`,
		"src/main/java/probe/FieldProbe.java": `package probe;
public final class FieldProbe {public static void main(String[] args){System.out.println(new Leaf().JavaDynamicTypeID);}}
`,
	}, "probe.FieldProbe", "9\n")
}

func TestCampaignObjectTextImplicitVarargsPrecedenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>implicit-text</artifactId><version>1</version></project>`,
		"src/main/java/probe/Implicit.java": `package probe;
public final class Implicit {
 @Override public int hashCode(){return 42;}
 public String toString(String... args){return "varargs:"+args.length;}
 public String implicit(){return toString();}
}
`,
		"src/main/java/probe/ImplicitProbe.java": `package probe;
public final class ImplicitProbe {public static void main(String[] args){Implicit value=new Implicit();System.out.println(value.implicit());System.out.println(value.toString("x"));}}
`,
	}, "probe.ImplicitProbe", "probe.Implicit@2a\nvarargs:1\n")
}

func TestCampaignObjectTextBoundedExecutionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>bound-text</artifactId><version>1</version></project>`,
		"src/main/java/probe/Locked.java": `package probe;
public final class Locked implements Runnable {
 public Thread owner; public int hashes; public boolean correct=true;
 public void run(){}
 @Override public synchronized int hashCode(){hashes++;correct=correct && Thread.currentThread()==owner;return 42;}
}
`,
		"src/main/java/probe/Generic.java": `package probe;
public final class Generic<Locked> {
 public <Locked extends probe.Locked> String shadow(Locked value){return String.valueOf(value);}
}
`,
		"src/main/java/probe/BoundProbe.java": `package probe;
public final class BoundProbe {
 static <T extends Locked> String text(T value){return String.valueOf(value);}
 static <T extends Locked & Runnable> String intersection(T value){return String.valueOf(value);}
 public static void main(String[] args){
  Locked value=new Locked();value.owner=Thread.currentThread();Generic<String> generic=new Generic<>();
  synchronized(value){System.out.println(text(value));System.out.println(intersection(value));System.out.println(generic.shadow(value));}
  System.out.println(value.correct+":"+value.hashes);
 }
}
`,
	}, "probe.BoundProbe", "probe.Locked@2a\nprobe.Locked@2a\nprobe.Locked@2a\ntrue:3\n")
}
