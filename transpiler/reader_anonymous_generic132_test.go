package transpiler

import "testing"

func TestReaderAnonymousGenericBinding132(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"src/main/java/members/Main.java": "package members;import java.io.Reader;\npublic class Main{static <__java2goAnonymous> boolean check(__java2goAnonymous value){return new Reader(value){__java2goAnonymous kept=value;boolean first=lock==value;public int read(char[]b,int o,int n){return -1;}public void close(){}boolean same(){return first&&lock==kept;}}.same();}\npublic static void main(String[]args){System.out.println(check(new Object()));}}", "pom.xml": "<project><groupId>members</groupId><artifactId>anonymousgenericbinding</artifactId><version>1</version></project>"}, "members.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "true\n"}, {name: "second", stdout: "true\n"}, {name: "third", stdout: "true\n"}})
}
