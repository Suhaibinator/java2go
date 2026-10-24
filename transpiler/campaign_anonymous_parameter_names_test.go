package transpiler

import "testing"

func TestCampaignAnonymousParameterKeywords(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>anonymous-keywords</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
interface Operation<T>{String run(T value,String other,String extra,int number);}
interface Counter {int value(int first,int second);}
public class Main {public static void main(String[] args){
 Operation<String> function=new Operation<String>(){public String run(String type,String type_,String map,int range){return type+":"+type_+":"+map+":"+range;}};
 System.out.println(function.run("a","b","c",4));
 Counter counter=new Counter(){int offset=2;public int value(int type,int type_){return offset+type*10+type_;}};
 System.out.println(counter.value(3,4));
 Operation<String> locals=new Operation<String>(){public String run(String chan,String other,String extra,int number){String chan_="local";return chan+":"+chan_;}};
 System.out.println(locals.run("captured","","",0));
}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "a:b:c:4\n36\ncaptured:local\n")
}
