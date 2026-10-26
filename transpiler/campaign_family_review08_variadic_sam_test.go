package transpiler

import "testing"

func TestCampaignFamilyReview08VariadicSAM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>variadic-sam</artifactId><version>1</version></project>`,
		"src/main/java/review/varargs/Main.java": `package review.varargs;
interface Action<T>{T first(T... values);}
public class Main{
 static <X> Action<X> same(Action<X> action){return action;}
 @SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
 int[] entered=new int[]{0};Action<String> typed=same((String[] values)->{entered[0]++;if(values==null){return "null";}return values.length==0?"empty":values[0];});
 System.out.println("scalar="+typed.first("one","later")+":"+entered[0]);
 String[] given=new String[]{"chosen"};System.out.println("array="+typed.first(given)+":"+entered[0]);
 Action raw=typed;try{raw.first(new Object[]{17});System.out.println("wrong");}catch(ClassCastException expected){System.out.println("bridge="+entered[0]);}
 System.out.println("empty="+typed.first()+":"+entered[0]);
 System.out.println("null="+typed.first((String[])null)+":"+entered[0]);
 }}
`}, "review.varargs.Main", "scalar=one:1\narray=chosen:2\nbridge=2\nempty=empty:3\nnull=null:4\n")
}
