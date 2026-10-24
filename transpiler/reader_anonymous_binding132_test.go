package transpiler

import "testing"

func TestReaderAnonymousBinding132(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"src/main/java/members/Main.java": "package members;import java.io.Reader;\npublic class Main{static boolean check(Object __java2goAnonymous){return new Reader(__java2goAnonymous){public Object ReaderBase;boolean first=lock==__java2goAnonymous;public int read(char[]b,int o,int n){return -1;}public void close(){}boolean same(){return first&&lock==__java2goAnonymous;}}.same();}\npublic static void main(String[]args){boolean boxed=new Reader(7){public int read(char[]b,int o,int n){return -1;}public void close(){}boolean same(){return lock instanceof java.lang.Integer&&((java.lang.Integer)lock).intValue()==7;}}.same();System.out.println(check(new Object())+\":\"+boxed);}}", "pom.xml": "<project><groupId>members</groupId><artifactId>anonymousbinding</artifactId><version>1</version></project>"}, "members.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "true:true\n"}, {name: "second", stdout: "true:true\n"}, {name: "third", stdout: "true:true\n"}})
}
