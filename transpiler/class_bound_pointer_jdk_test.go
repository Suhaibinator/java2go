package transpiler

import "testing"

func TestClassBoundPointerAnimalDeclarationJDK(t *testing.T) {
	files := map[string]string{
		"pom.xml":                           "<project><groupId>observe</groupId><artifactId>guard128</artifactId><version>1</version></project>",
		"src/main/java/foreign/Animal.java": "package foreign;\npublic class Animal {public String voice(){return \"base\";} }\n",
		"src/main/java/app/Main.java":       "package app;\nimport foreign.Animal;\nclass Child extends Animal {@Override public String voice(){return \"child\";} }\nclass Bound<Animal extends foreign.Animal> {\n int calls;\n Animal once(Animal value){calls++;return value;}\n String read(Animal value){return once(value).voice();}\n <Animal extends foreign.Animal> String shadow(Animal value){return value.voice();}\n}\npublic class Main {\n static <T extends foreign.Animal> String witness(T value){return value.voice();}\n public static void main(String[] args){Bound<Animal> bound=new Bound<>();Child child=new Child();\n System.out.println(bound.read(child)+\":\"+bound.calls+\":\"+bound.shadow(child)+\":\"+witness(child));\n try{bound.read(null);}catch(NullPointerException expected){System.out.println(\"nil:\"+bound.calls);}\n System.out.println(bound.read(child)+\":\"+bound.calls+\":\"+(child.getClass().getSuperclass()==foreign.Animal.class));\n }\n}\n",
	}
	runCampaignCompilerStrictProjectOracle(t, files, "app.Main", "child:1:child:child\nnil:2\nchild:3:true\n")
}
