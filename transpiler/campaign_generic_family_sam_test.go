package transpiler

import "testing"

func TestCampaignGenericFamilySAMBridgeTiming(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>family-sam</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
interface Action<T>{T apply(T value);}
class Target{static int created;int calls;Target(){created++;}String process(String value){calls++;return value;}}
public class Main{
 static <X> Action<X> same(Action<X> action){return action;}
 @SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
 int[] entered=new int[]{0};Action<String> typed=same((String value)->{entered[0]++;return value;});Action raw=typed;
 try{raw.apply(17);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("before="+entered[0]);}
 System.out.println(typed.apply("lambda")+":"+entered[0]+":"+((Object)typed==(Object)raw));
 Target target=new Target();Action<String> reference=same(target::process);Action erased=reference;
 try{erased.apply(23);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("method="+target.calls+":"+Target.created);}
 System.out.println(reference.apply("reference")+":"+target.calls);
 }}
`}, "example.Main", "before=0\nlambda:1:true\nmethod=0:1\nreference:1\n")
}
