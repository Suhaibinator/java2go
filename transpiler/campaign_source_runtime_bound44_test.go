package transpiler

import "testing"

func TestCampaignSourceRuntimeBound44InheritedJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/Job.java":   `package library; public interface Job extends java.lang.Runnable {}`,
		"src/main/java/library/Base.java":  `package library; public class Base implements Job { public void run(){} }`,
		"src/main/java/library/Child.java": `package library; public final class Child extends Base {}`,
		"src/main/java/probe/Main.java": `package probe; import library.Child; import library.Job;
public class Main {
 static <T extends java.lang.Runnable> int choose(T value){return 1;}
 static int choose(Object value){return 2;}
 public static void main(String[] args){Child child=new Child();Job job=child;System.out.println(choose(child)+":"+choose(job));}
}`,
	}, "probe.Main", "1:1\n")
}

func TestCampaignSourceRuntimeBound44ShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                             campaignStaticImportPOM26,
		"src/main/java/library/Runnable.java": `package library; public interface Runnable {void run();}`,
		"src/main/java/library/Foreign.java":  `package library; public class Foreign implements Runnable {public void run(){}}`,
		"src/main/java/probe/Main.java": `package probe; import library.Foreign;
public class Main {
 static <T extends java.lang.Runnable> int choose(T value){return 1;}
 static int choose(Object value){return 2;}
 public static void main(String[] args){System.out.println(choose(new Foreign()));}
}`,
	}, "probe.Main", "2\n")
}

func TestCampaignSourceRuntimeBound44StaticObjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Task.java": `package probe; public class Task implements java.lang.Runnable {public void run(){}}`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static <T extends java.lang.Runnable> int choose(T value){return 1;}
 static int choose(Object value){return 2;}
 public static void main(String[] args){Object value=new Task();System.out.println(choose(value));}
}`,
	}, "probe.Main", "2\n")
}
