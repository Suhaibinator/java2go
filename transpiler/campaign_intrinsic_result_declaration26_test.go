package transpiler

import "testing"

func TestCampaignIntrinsicResult26StringImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/shadow/String.java": `package shadow;public class String {}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.String;
public class Main {
 static java.lang.String pick(java.lang.String value){return "jdk:"+value;}
 static java.lang.String pick(String value){return "source";}
 static java.lang.String array(java.lang.String[] value){return "jdk-array:"+value.length;}
 static java.lang.String array(String[] value){return "source-array";}
 public static void main(java.lang.String[] args){java.util.TimeZone zone=java.util.TimeZone.getTimeZone("UTC");System.out.println(pick(java.lang.Integer.toString(7))+":"+pick(zone.getID())+":"+array("a:b".split(":")));}
}`}, "probe.Main", "jdk:7:jdk:UTC:jdk-array:2\n")
}

func TestCampaignIntrinsicResult26ObjectImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/shadow/Object.java": `package shadow;public class Object {}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.Object;
public class Main {
 static String pick(java.lang.Object value){return "jdk:"+(value!=null);}
 static String pick(Object value){return "source";}
 public static void main(String[] args){java.util.TimeZone zone=java.util.TimeZone.getTimeZone("UTC");System.out.println(pick(zone.clone()));}
}`}, "probe.Main", "jdk:true\n")
}

func TestCampaignIntrinsicResult26TimeZoneImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                            campaignStaticImportPOM26,
		"src/main/java/shadow/TimeZone.java": `package shadow;public class TimeZone {}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.TimeZone;
public class Main {
 static String pick(java.util.TimeZone value){return "jdk:"+value.getID();}
 static String pick(TimeZone value){return "source";}
 public static void main(String[] args){java.util.Calendar calendar=new java.util.GregorianCalendar(java.util.TimeZone.getTimeZone("UTC"));System.out.println(pick(java.util.TimeZone.getTimeZone("GMT+01:30"))+":"+pick(calendar.getTimeZone()));}
}`}, "probe.Main", "jdk:GMT+01:30:jdk:UTC\n")
}

func TestCampaignIntrinsicResult26ThreadImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/shadow/Thread.java": `package shadow;public class Thread {}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.Thread;
public class Main {
 static String pick(java.lang.Thread value){return "jdk:"+(value==java.lang.Thread.currentThread());}
 static String pick(Thread value){return "source";}
 public static void main(String[] args){System.out.println(pick(java.lang.Thread.currentThread()));}
}`}, "probe.Main", "jdk:true\n")
}

func TestCampaignIntrinsicResult26ContainerImportJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                            campaignStaticImportPOM26,
		"src/main/java/shadow/List.java":     `package shadow;public class List<T> {}`,
		"src/main/java/shadow/Runnable.java": `package shadow;public class Runnable {}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.List;import shadow.Runnable;
public class Main {
 static String pick(java.util.List<java.lang.Runnable> value){return "jdk:"+value.size();}
 static String pick(List<Runnable> value){return "source";}
 public static void main(String[] args){java.util.concurrent.ExecutorService executor=java.util.concurrent.Executors.newSingleThreadExecutor();System.out.println(pick(executor.shutdownNow()));}
}`}, "probe.Main", "jdk:0\n")
}
