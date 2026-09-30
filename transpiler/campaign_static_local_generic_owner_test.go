package transpiler

import "testing"

func TestCampaignStaticLocalGenericOwner(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>static-local-owner</artifactId><version>1</version></project>`,
		"src/main/java/review/staticlocal/Main.java": `package review.staticlocal;
class Owner<A>{
 static <A>A copy(A initial){
  class Local{A stored;Local(A value){stored=value;}A read(){return stored;}}
  Local local=new Local(initial);return local.read();
 }
}
public class Main{public static void main(String[] args){String text=Owner.copy("method");Integer number=Owner.copy(7);System.out.println(text+":"+number);}}
`}, "review.staticlocal.Main", "method:7\n")
}
