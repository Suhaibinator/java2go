package transpiler

import "testing"

func TestCampaignJavaOctalLiterals(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>octal-literals</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example; public class Main {public static void main(String[] args){
 char zero='\0';char one='\1';char line='\12';char letter='\101';char high='\377';char question='\77';
 System.out.println((int)zero+","+(int)one+","+(int)line+","+(int)letter+","+(int)high+","+(int)question);
 String text="\0\12\101\377\400\777\1234\08\\101";
 StringBuilder result=new StringBuilder();
 for(int i=0;i<text.length();i++){result.append((int)text.charAt(i)).append(',');}
 System.out.println(text.length()+":"+result.toString());
 System.out.println("\377".equals("\u00ff"));
 char[] surrogate={'\uD83D','\uDE00'};System.out.println((int)surrogate[0]+","+(int)surrogate[1]);
}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "0,1,10,65,255,63\n16:0,10,65,255,32,48,63,55,83,52,0,56,92,49,48,49,\ntrue\n55357,56832\n")
}
