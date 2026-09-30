package transpiler

import (
	"fmt"
	"os"
	"testing"
)

func TestCampaignStaticFieldShadowJVMParity(t *testing.T) {
	const source = `
class ShadowParent {
 static int counter = CampaignStaticShadow.mark("I");
}
class ShadowChild extends ShadowParent {
 static int dormant = CampaignStaticShadow.mark("S");
}
public class CampaignStaticShadow {
 static int value;
 static int value_1;
 static String trace = "";
 static int mark(String text) { trace += text; return 3; }
 static ShadowChild qualifier() { mark("Q"); return null; }
 static int rhs() { mark("R"); return 7; }
 static String parameter(int value) {
  CampaignStaticShadow.value = value;
  int before = CampaignStaticShadow.value++;
  int after = ++CampaignStaticShadow.value;
  CampaignStaticShadow.value += value;
  CampaignStaticShadow.value -= 1;
  return value+":"+before+":"+after+":"+CampaignStaticShadow.value;
 }
 static String inherited(int counter) {
  qualifier().counter = rhs();
  String written = trace+":"+ShadowChild.counter+":"+counter;
  qualifier().counter += rhs();
  int old = qualifier().counter++;
  int next = ++qualifier().counter;
  return written+"|"+trace+":"+old+":"+next+":"+ShadowParent.counter+":"+counter;
 }
 public static String run() {
  String parameters = parameter(10);
  int Java2goStaticStorage5_value = 100;
  int Java2goStaticStorage5_value_1 = 200;
  { int value = 5; CampaignStaticShadow.value = value; CampaignStaticShadow.value++; }
  int value = 99;
  CampaignStaticShadow.value = 4;
  int read = CampaignStaticShadow.value;
  CampaignStaticShadow.value *= 3;
  int post = CampaignStaticShadow.value--;
  int pre = --CampaignStaticShadow.value;
  return Java2goStaticStorage5_value+":"+Java2goStaticStorage5_value_1+"|"+parameters+"|"+value+":"+read+":"+post+":"+pre+":"+CampaignStaticShadow.value+"|"+inherited(88);
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignStaticShadow", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestShadow(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}

// Keep the complete callback/mutation counterexample that exposed field capture,
// rather than validating only a renamed reduction of it.
func TestCampaignStaticFieldShadowOriginalMutationJVMParity(t *testing.T) {
	source, err := os.ReadFile("../campaign/reproducers/original-map-mutation/CampaignMapMutation.java")
	if err != nil {
		t.Fatal(err)
	}
	want := campaignRuntimeJavaOracle(t, "CampaignMapMutation", string(source))
	generated := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestMutation(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}

func TestCampaignStaticFieldShadowPackagesJVMParity(t *testing.T) {
	for _, pkg := range []string{"p", "q"} {
		t.Run(pkg, func(t *testing.T) {
			files := map[string]string{
				"pom.xml": `<project><groupId>example</groupId><artifactId>shadow</artifactId><version>1</version></project>`,
				"src/main/java/p/Owner.java": `package p; public class Owner {public static int value;
 static int value_1;}`,
				"src/main/java/" + pkg + "/Main.java": fmt.Sprintf(`package %s; import p.Owner;
public class Main {
 static String modify(int Value) {
  Owner.value=Value; int old=Owner.value++; int next=++Owner.value;
  Owner.value+=Value; return Value+":"+old+":"+next+":"+Owner.value;
 }
 public static void main(String[]args){System.out.println(modify(7));}
}`, pkg),
			}
			runCampaignCompilerProjectOracle(t, files, pkg+".Main", "7:7:9:16\n")
		})
	}
}

func TestCampaignStaticFieldShadowEnumJVMParity(t *testing.T) {
	const source = `enum ShadowEnum { ONE; static int score; static int apply(int score){ShadowEnum.score=score;return ++ShadowEnum.score+score;} }
public class CampaignStaticEnum {public static int run(){return ShadowEnum.apply(4)*10+ShadowEnum.score;}}`
	want := campaignRuntimeJavaOracle(t, "CampaignStaticEnum", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing";"fmt")
func TestEnum(t *testing.T){if got:=fmt.Sprint(Run());got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
