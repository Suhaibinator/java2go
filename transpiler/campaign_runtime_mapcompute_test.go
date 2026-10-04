package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeMapComputeIfAbsent(t *testing.T) {
	const source = `import java.util.*;
public class CampaignRuntimeMapComputeIfAbsent {
 static String exercise(Map<String,String> map) {
  int[] calls=new int[]{0};
  map.put("present","kept");map.put("nil",null);
  String kept=map.computeIfAbsent("present",k->{calls[0]++;return "wrong";});
  String made=map.computeIfAbsent("nil",k->{calls[0]++;return k+"-value";});
  String missing=map.computeIfAbsent("missing",k->{calls[0]++;return null;});
  boolean absent=!map.containsKey("missing") && missing==null;
  boolean rejected=false;
  try{map.computeIfAbsent("present",null);}catch(NullPointerException expected){rejected=true;}
  boolean mutated=false;
  try{map.computeIfAbsent("new",k->{map.put("side","effect");return "unused";});}catch(ConcurrentModificationException expected){mutated=true;}
  boolean failed=false;
  try{map.computeIfAbsent("throws",k->{map.put("throw-side","effect");throw new IllegalStateException("original");});}catch(IllegalStateException expected){failed=true;}
  map.put("replacement",null);
  String computedNull=map.computeIfAbsent("replacement",k->{map.put(k,"callback");return null;});
  return kept+":"+made+":"+calls[0]+":"+absent+":"+rejected+":"+mutated+":"+map.get("side")+":"+map.containsKey("new")+":"+failed+":"+map.get("throw-side")+":"+map.containsKey("throws")+":"+(computedNull==null)+":"+map.get("replacement");
 }
 public static String run(){
  String hash=exercise(new HashMap<String,String>());
  String tree=exercise(new TreeMap<String,String>());
  TreeMap<String,String> empty=new TreeMap<>();
  String ignored=empty.computeIfAbsent(null,k->null);
  return hash+"|"+tree+"|"+(ignored==null)+":"+empty.size();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeMapComputeIfAbsent", source)
	t.Logf("JVM compute oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    "unicode/utf16"
)
func TestComputeOracle(t *testing.T) {
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
