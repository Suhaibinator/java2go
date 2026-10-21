package transpiler

import "testing"

func TestGenericInterfaceABIImplicitObjectJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `class Object{int code=99;}
class Child extends Object{}
interface Poly{<T>T id(T value);}
interface Bound{<T extends Object>T id(T value);}
class Named implements Poly{public<T>T id(T value){return value;}}
class Bounded implements Bound{public<T extends Object>T id(T value){return value;}}
public class Main{
 public static void main(String[]args){Poly p=new Named();String text=p.id("text");Bound bounded=new Bounded();Child child=new Child();Child seen=bounded.id(child);System.out.println(text+":"+(seen==child)+":"+seen.code);}
}
`}, "Main", "text:true:99\n")
}
