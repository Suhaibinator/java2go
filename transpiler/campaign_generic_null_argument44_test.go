package transpiler

import "testing"

func TestCampaignGenericNullArgument44JVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static <S extends java.lang.String> boolean absent(S value) { return value == null; }
 static <S extends java.lang.String> boolean witnessed(S value, S witness) { return value == null && witness != null; }
 static <T> boolean reference(T value) { return value == null; }
 static <T> boolean array(T[] values) { return values == null; }
 static <T> boolean same(T first, T second) { return first == second; }
 public static void main(String[] args) {
  Object actual = new Object();
  System.out.print(absent(null) + ":" + absent("") + ":" + Main.<String>absent(null)
   + ":" + witnessed(null, "witness") + ":" + reference(null) + ":" + reference(actual)
   + ":" + Main.<String>array(null) + ":" + array(new String[]{null}) + ":" + same(actual, actual));
 }
}`,
	}, "probe.Main", "true:false:true:true:true:false:true:false:true")
}
