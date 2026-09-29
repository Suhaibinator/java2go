package transpiler

import "testing"

func TestCampaignObjectsBoundMessage43JVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.Objects;public class Main {static <S extends java.lang.String> String check(S message){try{Objects.requireNonNull((Object)null,message);return "miss";}catch(NullPointerException expected){return expected.getMessage();}}public static void main(String[] args){System.out.print(check("detail")+":"+(check(null)==null));}}`,
	}, "probe.Main", "detail:true")
}

func TestCampaignObjectsBoundDetail43JVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import java.util.Objects;public class Main {static <S extends java.lang.String> String check(S message){try{Objects.requireNonNull((Object)null,message);return "miss";}catch(NullPointerException expected){return expected.getMessage();}}public static void main(String[] args){System.out.print(check("detail"));}}`,
	}, "probe.Main", "detail")
}
