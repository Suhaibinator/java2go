package transpiler

import "testing"

func TestCampaignMathToIntExact44BoundariesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.Math.toIntExact;public class Main {static String convert(long value){try{return "ok="+toIntExact(value);}catch(ArithmeticException e){return e.getClass().getName()+":"+e.getMessage();}}public static void main(String[] args){System.out.print(convert(0)+"|"+convert(Integer.MAX_VALUE)+"|"+convert(Integer.MIN_VALUE)+"|"+convert(2147483648L)+"|"+convert(-2147483649L)+"|"+convert(Long.MAX_VALUE)+"|"+convert(Long.MIN_VALUE));}}`,
	}, "probe.Main", "ok=0|ok=2147483647|ok=-2147483648|java.lang.ArithmeticException:integer overflow|java.lang.ArithmeticException:integer overflow|java.lang.ArithmeticException:integer overflow|java.lang.ArithmeticException:integer overflow")
}

func TestCampaignMathToIntExact44ConversionsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;public class Main {static int calls;static long value(){calls++;return 2147483648L;}public static void main(String[] args){byte b=-7;char c='\uffff';Integer i=9;Long l=11L;Long absent=null;boolean nullFailed=false,overflow=false;int consumed=0;try{consumed=java.lang.Math.toIntExact(value());}catch(ArithmeticException e){overflow=e.getMessage().equals("integer overflow");}try{java.lang.Math.toIntExact(absent);}catch(NullPointerException e){nullFailed=true;}System.out.print(Math.toIntExact(b)+":"+Math.toIntExact(c)+":"+Math.toIntExact(i)+":"+Math.toIntExact(l)+":"+overflow+":"+nullFailed+":"+calls+":"+consumed);}}`,
	}, "probe.Main", "-7:65535:9:11:true:true:1:0")
}

func TestCampaignMathToIntExact44SourceShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Math.java": `package probe;public class Math {public static int toIntExact(long value){return 17;}}`,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.Math.*;public class Main {public static void main(String[] args){System.out.print(Math.toIntExact(3)+":"+java.lang.Math.toIntExact(3)+":"+toIntExact(3));}}`,
	}, "probe.Main", "17:3:3")
}
