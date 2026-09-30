package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignSuperObjectToStringJVMParity(t *testing.T) {
	const source = `public class ObjectText {
  public static class Tag {
   public int hashCode(){return 42;}
   public String toString(){return super.toString()+"[tag]";}
  }
  public static class Child extends Tag {public int hashCode(){return 67;}}
  public static String run() {return new Tag().toString()+":"+new Child().toString();}
 }`
	want := campaignRuntimeJavaOracle(t, "ObjectText", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestText(t *testing.T) {if got:=Run();got!=%q {t.Fatalf("got %%q want %%q",got,%q)}}`, want, want))
}
