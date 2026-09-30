package transpiler

import "testing"

func TestCampaignRuntimeSystemOut18SourceAndValueShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>systemout</groupId><artifactId>shadow</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class Printer {int calls;String text="";public void println(String value){calls++;text+=value;}}
class System {static final Printer out=new Printer();}
class Holder {final Printer out=new Printer();}
public class Main {
 static void throughValue(Holder System){System.out.println("value");}
 public static void main(String[] args){System.out.println("source");Holder holder=new Holder();throughValue(holder);java.lang.System.out.println(System.out.calls+":"+System.out.text+":"+holder.out.calls+":"+holder.out.text);}
}`,
	}, "probe.Main", "1:source:1:value\n")
}
