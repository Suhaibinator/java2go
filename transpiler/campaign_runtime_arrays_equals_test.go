package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeArraysEquals(t *testing.T) {
	const source = `import java.util.Arrays;
public class CampaignArraysEquals {
 static class Token {
  static int calls;
  int value;
  Token(int value) { this.value=value; }
  public boolean equals(Object other) {
   calls++;
   return other instanceof Token && ((Token)other).value == value;
  }
 }
 public static String run() {
  StringBuilder out=new StringBuilder();
  byte[] a={1,-2,3}; byte[] b={1,-2,3}; byte[] none=null;
  out.append(Arrays.equals(a,b)).append(":"); b[1]=5;
  out.append(Arrays.equals(a,b)).append(":").append(Arrays.equals(a,a)).append(":")
   .append(Arrays.equals(none,none)).append(":").append(Arrays.equals(none,a)).append(":")
   .append(Arrays.equals(new byte[0],new byte[0])).append(":");
  out.append(Arrays.equals(new boolean[]{true,false},new boolean[]{true,false})).append(":")
   .append(Arrays.equals(new short[]{2,3},new short[]{2,4})).append(":")
   .append(Arrays.equals(new char[]{'a','漢'},new char[]{'a','漢'})).append(":")
   .append(Arrays.equals(new int[]{1,2},new int[]{1})).append(":")
   .append(Arrays.equals(new long[]{1L,2L},new long[]{1L,2L})).append(":");
  double negative=-1.0; negative=negative/Double.POSITIVE_INFINITY;
  float negativeFloat=-1.0f; negativeFloat=negativeFloat/Float.POSITIVE_INFINITY;
  out.append(Arrays.equals(new double[]{Double.NaN,negative},new double[]{Double.NaN,negative})).append(":")
   .append(Arrays.equals(new double[]{negative},new double[]{0.0})).append(":")
   .append(Arrays.equals(new float[]{Float.NaN},new float[]{Float.NaN})).append(":")
   .append(Arrays.equals(new float[]{negativeFloat},new float[]{0.0f})).append(":");
  Token first=new Token(7);
  Object[] left={first,null,"alpha"}; Object[] right={new Token(7),null,new String("alpha")};
  out.append(Arrays.equals(left,right)).append(":").append(Token.calls).append(":")
   .append(Arrays.equals(left,left)).append(":").append(Token.calls).append(":");
  Object[] nestedA={a}; Object[] nestedB={new byte[]{1,-2,3}};
  out.append(Arrays.equals(nestedA,nestedB)).append(":"); nestedB[0]=a;
  out.append(Arrays.equals(nestedA,nestedB));
  return out.toString();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignArraysEquals", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    "unicode/utf16"
)
func TestArraysEquals(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null")
    }
    const want = %q
    if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
        t.Fatalf("JVM %%q != Go UTF16 %%v", want, units)
    }
}`, want))
}
