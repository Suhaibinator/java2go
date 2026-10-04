package transpiler

import "testing"

func TestCampaignEscapedSurrogatePairLiteral(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>surrogate-literal</artifactId><version>1</version></project>`,
		"src/main/java/review/literal/Main.java": `package review.literal;public class Main{public static void main(String[] args){
 String digit="\uD835\uDFD9";System.out.println(digit.length()+":"+(int)digit.charAt(0)+":"+(int)digit.charAt(1)+":"+digit.equals("𝟙"));
 String spelled="\\uD835\\uDFD9";System.out.println(spelled.length()+":"+(int)spelled.charAt(0));
 char high='\uD835';char low='\uDFD9';System.out.println((int)high+":"+(int)low);
}}`}, "review.literal.Main", "2:55349:57305:true\n12:92\n55349:57305\n")
}

func TestCampaignEscapedSurrogatePairBackslashEligibility(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>surrogate-eligibility</artifactId><version>1</version></project>`,
		"src/main/java/review/literal/Main.java": `package review.literal;public class Main{public static void main(String[] args){
 String odd="\\\uD835\uDFD9";System.out.println(odd.length()+":"+(int)odd.charAt(0)+":"+(int)odd.charAt(1));
 String repeated="\uuD835\uuuDFD9";System.out.println(repeated.equals("𝟙"));
 String adjacent="A\uD835\uDFD9Z";System.out.println(adjacent.length()+":"+(int)adjacent.charAt(0)+":"+(int)adjacent.charAt(3));
}}`}, "review.literal.Main", "3:92:55349\ntrue\n4:65:90\n")
}
