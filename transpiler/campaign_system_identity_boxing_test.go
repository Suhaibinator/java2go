package transpiler

import "testing"

func TestCampaignSystemIdentityBoxingJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>identityboxing</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/identityboxing/Main.java": `package identityboxing;
public class Main {
 public static void main(String[] args){
  Integer boxed=7;
  System.out.println(System.identityHashCode(7)==System.identityHashCode(boxed));
  Boolean logical=true;
  System.out.println(System.identityHashCode(true)==System.identityHashCode(logical));
 }
}`}, "identityboxing.Main", "true\ntrue\n")
}
