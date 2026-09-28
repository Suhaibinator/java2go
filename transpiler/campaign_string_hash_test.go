package transpiler

import "testing"

func TestCampaignStringHashCode(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>string-hash</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
import java.util.function.ToIntFunction;
public class Main {
 static int calls;
 static String next(){calls++;return "A😀B";}
 public static void main(String[] args){
  String[] values={"", "abc", "A😀B", "\0é中", "overflow-overflow-overflow"};
  ToIntFunction<String> hash=String::hashCode;
  for(String value:values){
   Object erased=value;CharSequence sequence=value;
   System.out.println(value.hashCode()+":"+erased.hashCode()+":"+sequence.hashCode()+":"+hash.applyAsInt(value));
  }
  System.out.println(next().hashCode()+":"+calls);
  String missing=null;
  try {System.out.println(missing.hashCode());}catch(NullPointerException expected){System.out.println("null");}
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "0:0:0:0\n96354:96354:96354:96354\n56896350:56896350:56896350:56896350\n27236:27236:27236:27236\n1112218722:1112218722:1112218722:1112218722\n56896350:1\nnull\n")
}
