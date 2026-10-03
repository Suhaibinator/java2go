package transpiler

import "testing"

// Array text conversion must preserve the Java String result boundary before
// callers proceed to direct/default Comparator operations. The source default
// interface participates in the rendered Comparable values as well.
func TestCampaignComparatorArrayStringBoundaryStrictJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>boundary</groupId><artifactId>array-comparators</artifactId><version>1</version></project>`,
		"src/main/java/boundary/Main.java": `package boundary;
import java.util.Arrays;
import java.util.Comparator;
interface Label { default String label() { return "v"; } }
public class Main {
 static class Version implements Comparable<Version>, Label {
  int key;
  Version(int key) { this.key=key; }
  public int compareTo(Version other) { return key-other.key; }
  public String toString() { return label()+key; }
 }
 public static void main(String[] args) {
  Version[] values={new Version(3),new Version(1),new Version(2)};
  Arrays.sort(values);
  String first=Arrays.toString(values),second=Arrays.toString(values);
  System.out.println(first+":"+(first!=second)+":"+first.equals(second));
  double[] floating={Double.NaN,0.0,-0.0,1.0};
  Arrays.sort(floating);
  String text=Arrays.toString(floating);
  System.out.println(text+":"+text.length());
  Comparator<Version> byKey=(a,b)->a.key-b.key;
  System.out.println(byKey.compare(values[0],values[2])+":"+byKey.reversed().compare(values[0],values[2]));
  Comparator<Version> byText=byKey.thenComparing((Version value)->value.label());
  System.out.println(byText.compare(values[0],new Version(1)));
 }
}`,
	}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "boundary.Main", "[v1, v2, v3]:true:true\n[-0.0, 0.0, 1.0, NaN]:21\n-2:2\n0\n")
}
