package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeStringBuilderUTF16(t *testing.T) {
	const source = `public class CampaignRuntimeStringBuilderUTF16 {
 static String units(StringBuilder builder) {
  String result = "" + builder.length();
  for (int i=0;i<builder.length();i++) result += "," + (int)builder.charAt(i);
  return result;
 }
 public static String run() {
  StringBuilder value = new StringBuilder("A😀B");
  String result = units(value);
  value.append((char)55357).append((char)56832);
  result += ";" + units(value) + ";" + value.toString();
  value.reverse();
  result += ";" + units(value) + ";" + value.toString();
  StringBuilder pair = new StringBuilder();
  pair.append((char)56832).append((char)55357);
  result += ";" + units(pair);
  pair.reverse();
  result += ";" + units(pair) + ";" + pair.toString();
  pair.reverse();
  result += ";" + units(pair);
  pair.deleteCharAt(0);
  result += ";" + units(pair);
  pair.insert(0,(char)55357);
  result += ";" + pair.toString();
  pair.insert(1,'X');
  result += ";" + units(pair);
  pair.deleteCharAt(1);
  result += ";" + pair.toString();
  StringBuilder arrays = new StringBuilder();
  arrays.append(new char[]{(char)55357}).append(new char[]{(char)56832});
  result += ";" + arrays.toString();
  try { pair.charAt(-1); } catch (IndexOutOfBoundsException e) {result += ";"+e.getClass().getSimpleName()+":"+e.getMessage();}
  try { pair.deleteCharAt(2); } catch (IndexOutOfBoundsException e) {result += ";"+e.getClass().getSimpleName()+":"+e.getMessage();}
  try { pair.insert(3,'x'); } catch (IndexOutOfBoundsException e) {result += ";"+e.getClass().getSimpleName()+":"+e.getMessage();}
  return result + ";" + pair.toString();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeStringBuilderUTF16", source)
	t.Logf("JVM UTF16 builder oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestBuilderUnits(t *testing.T) {
 got := Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a nonnull String") }
 wantUnits := %#v
 gotUnits := got.UTF16Copy()
 if !slices.Equal(gotUnits, wantUnits) { t.Fatalf("JVM %%q (UTF16 %%x) != Go UTF16 %%x", %q, wantUnits, gotUnits) }
}
`, utf16.Encode([]rune(want)), want))
}
