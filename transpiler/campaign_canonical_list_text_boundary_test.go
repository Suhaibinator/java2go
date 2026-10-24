package transpiler

import "testing"

// The strict helper captures actual JDK21 output before comparing the fixed
// observations, transpiling, or comparing the generated race-built application.
func TestCampaignCanonicalListTextBoundaryJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
import java.util.ArrayList;
import java.util.List;
import java.util.function.Function;
import java.util.function.Supplier;
public class Main {
 public static void main(String[] args) {
  List<Object> values = new ArrayList<>();
  values.add("a"); values.add(null); values.add(values);
  String direct = values.toString();
  String converted = String.valueOf(values);
  Object erased = values;
  String throughObject = erased.toString();
  System.out.println(direct+":"+direct.equals(converted)+":"+converted.equals(throughObject)+":"+(direct==converted)+":"+(direct==throughObject));
  List<Object> empty = new ArrayList<>(); Object emptyAlias = empty;
  System.out.println("empty="+(empty.toString()=="[]")+":"+(String.valueOf(empty)=="[]")+":"+(emptyAlias.toString()=="[]"));
  Supplier<String> bound = values::toString;
  Function<List<Object>,String> unbound = List::toString;
  List<Object> original = values;
  values = new ArrayList<>(); values.add("new");
  System.out.println("refs="+bound.get()+":"+unbound.apply(original)+":"+values.toString());
 }
}
`,
	}, "probe.Main", "[a, null, (this Collection)]:true:true:false:false\nempty=true:true:true\nrefs=[a, null, (this Collection)]:[a, null, (this Collection)]:[new]\n")
}
