package transpiler

import "testing"

func TestCampaignSystemQualifiedValue26JVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
class Printer {int calls;String text="";void println(String value){calls++;text+=value;}}
class Holder {Printer out=new Printer();int calls;long nanoTime(){calls++;return -17L;}}
class Lang {Holder System=new Holder();}
class Root {Lang lang=new Lang();}
public class Main {
 static long via(Root java){java.lang.System.out.println("value");return java.lang.System.nanoTime();}
 public static void main(String[] args){Root root=new Root();long value=via(root);java.lang.System.out.println((value==-17L)+":"+root.lang.System.calls+":"+root.lang.System.out.calls+":"+root.lang.System.out.text);}
}`,
	}, "probe.Main", "true:1:1:value\n")
}
