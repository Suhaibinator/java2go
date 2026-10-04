package transpiler

import "testing"

func TestCampaignStatelessObjectAllocationIdentityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>stateless-object-identity</artifactId><version>1</version></project>`,
		"src/main/java/review/objectallocation/Main.java": `package review.objectallocation;
import java.util.HashMap;
public class Main {
 static class Empty{}
 interface Defaults {default int answer(){return 7;}}
 static class Carry implements Defaults{}
 static Object local(){class Local{}return new Local();}
 static Runnable anonymous(){return new Runnable(){public void run(){}};}
 public static void main(String[] args){
  Object first=new Empty(),second=new Empty(),alias=first;
  Object localFirst=local(),localSecond=local();
  Object anonFirst=anonymous(),anonSecond=anonymous();
  Object carryFirst=new Carry(),carrySecond=new Carry();
  System.out.println("distinct="+(first==second)+":"+(localFirst==localSecond)+":"+(anonFirst==anonSecond)+":"+(carryFirst==carrySecond));
  System.out.println("aliases="+(first==alias)+":"+first.equals(alias)+":"+first.equals(second));
  HashMap<Object,Integer> keys=new HashMap<>();
  keys.put(first,1);keys.put(second,2);keys.put(localFirst,3);keys.put(localSecond,4);
  keys.put(anonFirst,5);keys.put(anonSecond,6);keys.put(carryFirst,7);keys.put(carrySecond,8);
  System.out.println("map="+keys.size()+":"+keys.get(first)+":"+keys.get(second));
 }
}
`}, "review.objectallocation.Main", "distinct=false:false:false:false\naliases=true:true:false\nmap=8:1:2\n")
}
