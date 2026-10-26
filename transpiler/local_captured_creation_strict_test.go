package transpiler

import "testing"

func TestLocalCapturedCreationImplicitStrictJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>localcapture</groupId><artifactId>localcapture</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `public class Main {
 public static void main(String[] args) {
  final int[] captured={11};
  class Local { int value() { return captured[0]; } }
  Local local=new Local();
  System.out.println(local.value());
 }
}
`,
	}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "Main", "11\n")
}

func TestLocalCapturedCreationExplicitStrictJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>localcapture</groupId><artifactId>localcapture</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `public class Main {
 public static void main(String[] args) {
  final int[] captured={11};
  class Local { Local() {} int value() { return captured[0]; } }
  Local local=new Local();
  System.out.println(local.value());
 }
}
`,
	}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "Main", "11\n")
}

func TestLocalCapturedImplicitConstructorABIStrictJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>localcapture</groupId><artifactId>reflection</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `import java.lang.reflect.*;
public class Main {
 public static void main(String[] args) throws Exception {
  final int[] captured={11};
  class Local { int value() { return captured[0]; } }
  Local original=new Local();
  Class<?> type=original.getClass();
  Constructor<?> ctor=type.getDeclaredConstructor(int[].class);
  ctor.setAccessible(true);
  System.out.println(ctor.getModifiers()+":"+ctor.canAccess(null));
  Local copy=(Local)ctor.newInstance((Object)captured);
  captured[0]=12;
  System.out.println(original.value()+":"+copy.value()+":"+(copy!=original));
  try { type.getDeclaredConstructor(); } catch(NoSuchMethodException e) { System.out.println("no-noarg"); }
  try { ctor.newInstance("bad"); } catch(IllegalArgumentException e) { System.out.println("bad-argument"); }
  Local empty=(Local)ctor.newInstance((Object)null);
  try { empty.value(); } catch(NullPointerException e) { System.out.println("null-capture"); }
 }
}
`,
	}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "Main", "0:true\n12:12:true\nno-noarg\nbad-argument\nnull-capture\n")
}

func TestLocalCapturedPublicConstructorABIStrictJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>localcapture</groupId><artifactId>reflection</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `import java.lang.reflect.*;
public class Main {
 public static void main(String[] args) throws Exception {
  final int[] captured={11}; final int tag=args.length+7;
  class Local {
   int input;
   public Local() { this(5); }
   public Local(int value) { if(value<0)throw new IllegalArgumentException("negative"); input=value; }
   int value() { return captured[0]+tag+input; }
  }
  Local original=new Local(); Class<?> type=original.getClass();
  Constructor<?> ctor=type.getDeclaredConstructor(int.class,int[].class,int.class);
  ctor.setAccessible(true);
  System.out.println(ctor.getModifiers()+":"+ctor.canAccess(null));
  Local copy=(Local)ctor.newInstance(5,captured,tag);
  captured[0]=12;
  System.out.println(original.value()+":"+copy.value()+":"+(copy!=original));
  Constructor<?> zero=type.getDeclaredConstructor(int[].class,int.class);
  zero.setAccessible(true);
  System.out.println(zero.getModifiers()+":"+zero.canAccess(null));
  System.out.println(((Local)zero.newInstance(captured,tag)).value());
  try { type.getDeclaredConstructor(); } catch(NoSuchMethodException e) { System.out.println("no-noarg"); }
  try { ctor.newInstance(-1,captured,tag); } catch(InvocationTargetException e) { System.out.println(e.getCause().getClass().getName()+":"+e.getCause().getMessage()); }
  try { ctor.newInstance(5,"bad",tag); } catch(IllegalArgumentException e) { System.out.println("bad-argument"); }
  Local empty=(Local)ctor.newInstance(5,null,tag);
  try { empty.value(); } catch(NullPointerException e) { System.out.println("null-capture"); }
 }
}
`,
	}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "Main", "1:true\n24:24:true\n1:true\n24\nno-noarg\njava.lang.IllegalArgumentException:negative\nbad-argument\nnull-capture\n")
}
