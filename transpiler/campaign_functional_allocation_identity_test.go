package transpiler

import "testing"

func TestCampaignFunctionalStatelessAllocationIdentityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>functional-allocation-identity</artifactId><version>1</version></project>`,
		"src/main/java/review/functionallocation/Main.java": `package review.functionallocation;
import java.util.HashMap;
import java.util.function.Function;
public class Main {
 static class Named implements Function<String,String>{public String apply(String text){return text;}}
 static Function<String,String> anonymous(){return new Function<>(){public String apply(String text){return text;}};}
 public static void main(String[] args){
  Function<String,String> first=anonymous(),second=anonymous(),alias=first;
  Object firstObject=first,secondObject=second;
  System.out.println("anon="+(first==second)+":"+(first==alias)+":"+(firstObject==secondObject)+":"+firstObject.equals(secondObject));
  Function<String,String> namedFirst=new Named(),namedSecond=new Named(),namedAlias=namedFirst;
  Object namedObject=namedFirst,namedOther=namedSecond;
  System.out.println("named="+(namedFirst==namedSecond)+":"+(namedFirst==namedAlias)+":"+(namedObject==namedOther)+":"+namedObject.equals(namedOther));
  HashMap<Function<String,String>,Integer> keys=new HashMap<>();
  keys.put(first,1);keys.put(second,2);keys.put(namedFirst,3);keys.put(namedSecond,4);
  System.out.println("map="+keys.size()+":"+keys.get(first)+":"+keys.get(second)+":"+keys.get(namedFirst)+":"+keys.get(namedSecond));
 }
}
`}, "review.functionallocation.Main", "anon=false:true:false:false\nnamed=false:true:false:false\nmap=4:1:2:3:4\n")
}
