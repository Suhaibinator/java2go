package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
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
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestText(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != Go UTF16 %%#v", %q, units) }
}`, utf16.Encode([]rune(want)), want))
}
