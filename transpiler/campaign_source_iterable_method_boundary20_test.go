package transpiler

import "testing"

func TestCampaignSourceIterableMethodBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>iterable-boundary</artifactId><version>1</version></project>
`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
public class Main {
 static Iterable<String> widen(Collection<String> source){return source;}
 static String read(Iterable<String> source){String result="";for(String item:source)result=result+item;return result;}
 public static void main(String[] args){List<String> values=new ArrayList<>();values.add("a");Iterable<String> view=widen(values);values.add("b");System.out.println(read(view)+":"+(view==values)+":"+(widen(null)==null));}
}
`,
	}, "probe.Main", "ab:true:true\n")
}
