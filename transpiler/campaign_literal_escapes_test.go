package transpiler

import "testing"

func TestCampaignJavaQuoteAndSpaceEscapes(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>quote-literals</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;public class Main {public static void main(String[] args){
 char plain='"';char escaped='\"';char single='\'';char backslash='\\';char space='\s';
 System.out.println((int)plain+","+(int)escaped+","+(int)single+","+(int)backslash+","+(int)space);
 String text="\'\"\s\\s\\\"\\\'";
 StringBuilder values=new StringBuilder();for(int i=0;i<text.length();i++){values.append((int)text.charAt(i)).append(',');}
 System.out.println(values.toString());
 System.out.println("\'".equals("'"));System.out.println("\s".equals(" "));
}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "34,34,39,92,32\n39,34,32,92,115,92,34,92,39,\ntrue\ntrue\n")
}
