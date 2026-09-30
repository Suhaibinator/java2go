package transpiler

import "testing"

func TestCampaignFamilyNestedOwnerInterfaces(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>nested-owner</artifactId><version>1</version></project>`,
		"src/main/java/review/nested/Main.java": `package review.nested;
public class Main {
 interface Left<T>{T read();}interface Right<T>{T get();}
 static class Owner {
  interface Left<T>{T read();}interface Right<T>{T get();}
  @SuppressWarnings("unchecked")static <X>Left<X> left(Object value){return (Left<X>)value;}
  @SuppressWarnings("unchecked")static <X>Right<X> right(Object value){return (Right<X>)value;}
  static String run(){
   class Both implements Left<String>,Right<String>{public String read(){return "left";}public String get(){return "right";}}
   Both both=new Both();Left<String> left=left(both);Right<String> right=right(both);
   return left.read()+":"+right.get()+":"+((Object)left==(Object)right)+":"+((Object)both instanceof Main.Left)+":"+((Object)both instanceof Main.Right);
  }
 }
 public static void main(String[] args){System.out.println(Owner.run());}
}
`}, "review.nested.Main", "left:right:true:false:false\n")
}
