package transpiler

import "testing"

func TestCampaignInheritedOverloadPreservesDispatch(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "parent-first"
		if reverse {
			name = "child-first"
		}
		t.Run(name, func(t *testing.T) {
			parent := `class Parent{int broad;public Parent choose(Object value){broad++;return this;}public String number(long value){return "long";}}`
			child := `interface TextChoice{Parent choose(String value);} class Child extends Parent implements TextChoice{int narrow;public Child choose(String value){narrow++;return this;}public String number(int value){return "int";}}`
			leaf := `class Leaf extends Child{public Leaf choose(Object value){broad+=10;return this;}}`
			declarations := parent + child + leaf
			if reverse {
				declarations = leaf + child + parent
			}
			source := `package example;` + declarations + `public class Main{public static void main(String[] args){
     Child child=new Child();Parent baseView=child;TextChoice text=child;Object object="x";
     Parent broad=baseView.choose(object);Parent narrow=text.choose("x");child.choose(object);
     System.out.println(child.broad+":"+child.narrow+":"+(broad==child)+":"+(narrow==child));
     Leaf leaf=new Leaf();Parent asParent=leaf;TextChoice asText=leaf;Parent chosen=asParent.choose(object);asText.choose("x");leaf.choose(object);
     System.out.println(leaf.broad+":"+leaf.narrow+":"+(chosen==leaf));
     System.out.println(child.number(1)+":"+child.number(1L)+":"+baseView.number(1));
   }}`
			runCampaignCompilerProjectOracle(t, map[string]string{
				"pom.xml":                         `<project><groupId>example</groupId><artifactId>overload</artifactId><version>1</version></project>`,
				"src/main/java/example/Main.java": source,
			}, "example.Main", "2:1:true:true\n20:1:true\nint:long:long\n")
		})
	}
}
