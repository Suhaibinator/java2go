package transpiler

import "testing"

func TestCampaignImportKind44StaticFieldTypeNamespaceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/E.java":    `package probe;public class E {public int marker(){return 17;}}`,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.Math.E;public class Main {public static void main(String[] args){E value=new E();System.out.println(value.marker()+":"+(E>2));}}`,
	}, "probe.Main", "17:true\n")
}
