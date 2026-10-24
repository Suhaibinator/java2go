package transpiler

import "testing"

// Constants are pinned to actual OpenJDK21 classfile ConstantValue entries.
// The harness independently compiles/runs these unchanged Java files before
// comparing the complete generated-Go output.
func TestDoubleNaN181CanonicalFieldAndSourceBindings(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/constants/Fields.java": `package probe.constants;
import static java.lang.Double.NaN;
public class Fields {
 public static final double LITERAL = java.lang.Double.NaN;
 public static long simple(){return Double.doubleToRawLongBits(Double.NaN);}
 public static long imported(){return Double.doubleToRawLongBits(NaN);}
 public static int floatBits(){return Float.floatToRawIntBits(Float.NaN);}
}`,
		"src/main/java/probe/source/Double.java": `package probe.source; public class Double {public static final long NaN=37L;}`,
		"src/main/java/probe/source/Float.java":  `package probe.source; public class Float {public static final int NaN=41;}`,
		"src/main/java/probe/Main.java": `package probe;
import probe.constants.Fields;
import probe.source.Double;
import probe.source.Float;
public class Main {
 public static void main(String[] args){
  System.out.println(java.lang.Double.doubleToRawLongBits(java.lang.Double.NaN));
  System.out.println(Fields.simple());
  System.out.println(Fields.imported());
  System.out.println(java.lang.Double.doubleToRawLongBits(Fields.LITERAL));
  System.out.println(Fields.floatBits());
  System.out.println(Double.NaN+":"+Float.NaN);
  System.out.println(probe.source.Double.NaN+":"+probe.source.Float.NaN);
 }
}`,
	}, "probe.Main", "9221120237041090560\n9221120237041090560\n9221120237041090560\n9221120237041090560\n2143289344\n37:41\n37:41\n")
}
