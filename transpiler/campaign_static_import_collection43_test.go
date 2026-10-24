package transpiler

import "testing"

// This reduction retains package-private generic source static import and the
// List<Factory> -> Collection<E> formal conversion from the full Gson failure.
// It is a prerequisite only; the unchanged complete Gson closure remains a gate.
func TestCampaignStaticImportCollection43PackageGenericJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/probe/Factory.java": `package probe;public class Factory {public int id;public Factory(int id){this.id=id;}}`,
		"src/main/java/probe/Builder.java": `package probe;import java.util.Collection;import java.util.List;import java.util.ArrayList;public class Builder {final List<Factory> factories=new ArrayList<>();static <E> List<E> snapshot(Collection<E> values){return new ArrayList<E>(values);}public Builder(){factories.add(new Factory(7));}}`,
		"src/main/java/probe/Main.java":    `package probe;import static probe.Builder.snapshot;import java.util.List;public class Main {public static void main(String[] args){Builder builder=new Builder();List<Factory> first=snapshot(builder.factories);builder.factories.add(new Factory(9));List<Factory> second=Builder.snapshot(builder.factories);System.out.println(first.size()+":"+first.get(0).id+":"+second.size()+":"+(first.get(0)==second.get(0)));}}`,
	}, "probe.Main", "1:7:2:true\n")
}
func TestCampaignStaticImportCollection43QualifiedWildcardJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                           campaignStaticImportPOM26,
		"src/main/java/library/Copies.java": `package library;public class Copies {public static <E> java.util.List<E> snapshot(java.util.Collection<E> values){return new java.util.ArrayList<E>(values);}}`,
		"src/main/java/probe/Main.java":     `package probe;import static library.Copies.*;import java.util.List;import java.util.ArrayList;import java.util.Collection;public class Main {public static void main(String[] args){List<String> list=new ArrayList<>();list.add("x");Collection<String> collection=list;List<String> a=snapshot(list),b=snapshot(collection);System.out.println(a.size()+":"+b.get(0)+":"+(a!=b));}}`,
	}, "probe.Main", "1:x:true\n")
}
