package transpiler

import (
	"fmt"
	"os"
	"testing"
)

func TestCampaignConcatCovariantEvaluationOrder(t *testing.T) {
	source, err := os.ReadFile("../campaign/reproducers/concat-covariant-order/ConcatCovariantOrder.java")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>concat-order</artifactId><version>1</version></project>`,
		"src/main/java/probe/ConcatCovariantOrder.java": "package probe;\n" + string(source),
	}
	runCampaignCompilerProjectOracle(t, files, "probe.ConcatCovariantOrder", "0:true\n")
}

func TestCampaignConcatEffectsAndAbruptCompletion(t *testing.T) {
	const source = `class ConcatCell { int value; }
class ConcatRendered {
 int delta;
 ConcatRendered(int delta){this.delta=delta;}
 public String toString(){CampaignConcatEffects.value+=delta;return "rendered"+CampaignConcatEffects.value;}
}
public class CampaignConcatEffects {
 static int value;
 static String mutate(){value+=10;return "m";}
 static String fail(){throw new IllegalStateException("fail");}
 public static String run(){
  value=1;String first=value+":"+mutate();
  value=2;String second=value+":"+new ConcatRendered(5);
  value=3;String third=value+":"+(true?mutate():"skip");
  value=4;String fourth=value+":"+new ConcatRendered(7).toString()+":"+value;
  value=5;String existing="unchanged";try{existing=value+":"+fail();}catch(IllegalStateException expected){}
  ConcatCell absent=null;boolean rejected=false;value=0;
  try{String ignored=absent.value+":"+mutate();}catch(NullPointerException expected){rejected=true;}
  return first+"|"+second+"|"+third+"|"+fourth+"|"+existing+"|"+rejected+":"+value;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignConcatEffects", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestEffects(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
