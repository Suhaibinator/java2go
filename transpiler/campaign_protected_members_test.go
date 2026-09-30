package transpiler

import "testing"

func TestCampaignProtectedMembersAcrossPackages(t *testing.T) {
	files := map[string]string{
		"pom.xml":                                `<project><groupId>example</groupId><artifactId>protected</artifactId><version>1</version></project>`,
		"src/main/java/example/base/Base.java":   `package example.base; public class Base<T> { protected final T value; protected Base(T value){this.value=value;} protected int token(){return 4;} public int through(){return token();} public static Base<String> create(){return new example.child.Child("id");} }`,
		"src/main/java/example/child/Child.java": `package example.child; public class Child extends example.base.Base<String> { public Child(String value){super(value);} protected int token(){return 9;} public String read(){return value;} }`,
		"src/main/java/example/app/Main.java":    `package example.app; import example.base.Base; import example.child.Child; public class Main {public static void main(String[] args){Child child=new Child("id");System.out.println(child.read()+":"+child.through());System.out.println(Base.create().through());}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.app.Main", "id:9\n9\n")
}
