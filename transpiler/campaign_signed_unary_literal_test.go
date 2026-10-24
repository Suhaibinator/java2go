package transpiler

import "testing"

func TestCampaignSignedUnaryLiterals(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>signed-unary</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
public class Main {
 static final long MINIMUM=-9223372036854775808L;
 static final int INT_MINIMUM=-2147483648;
 static String kind(int value){return "int:"+value;}
 static String kind(long value){return "long:"+value;}
 public static void main(String[] args){
  System.out.println(MINIMUM);System.out.println(INT_MINIMUM);
  System.out.println(kind(-9_223_372_036_854_775_808l));
  System.out.println(kind(-2_147_483_648));
  System.out.println(kind(-(-9223372036854775808L)));
  System.out.println(kind(-(-2147483648)));
  System.out.println(kind(~(-9223372036854775808L)));
  System.out.println(kind(-0xffff_ffff));
  System.out.println(kind(-0xffff_ffff_ffff_ffffL));
  Long boxed=-9223372036854775808L;System.out.println(boxed.longValue());
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "-9223372036854775808\n-2147483648\nlong:-9223372036854775808\nint:-2147483648\nlong:-9223372036854775808\nint:-2147483648\nlong:9223372036854775807\nint:1\nlong:1\n-9223372036854775808\n")
}
