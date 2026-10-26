package transpiler

import "testing"

func TestCampaignPackagePrivateCycleDispatch(t *testing.T) {
	files := map[string]string{
		"pom.xml":                                    `<project><groupId>example</groupId><artifactId>privacy</artifactId><version>1</version></project>`,
		"src/main/java/example/base/Base.java":       `package example.base; import example.foreign.Foreign; public class Base { private int count=12; int token(){return count++;} public int throughBase(){return token();} public int throughCallback(Foreign value){return ((Base)value).token();} }`,
		"src/main/java/example/base/Local.java":      `package example.base; public class Local extends Base { int token(){return 33;} }`,
		"src/main/java/example/foreign/Foreign.java": `package example.foreign; import example.base.Base; public class Foreign extends Base { private int count=23; int token(){return count++;} public int throughForeign(){return token();} }`,
		"src/main/java/example/app/Main.java":        `package example.app; import example.base.Base; import example.base.Local; import example.foreign.Foreign; public class Main {public static void main(String[] args){Foreign value=new Foreign();Base base=value;System.out.println(base.throughBase());System.out.println(value.throughForeign());System.out.println(base.throughCallback(value));System.out.println(value.throughForeign());Base local=new Local();System.out.println(local.throughBase());}}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.app.Main", "12\n23\n13\n24\n33\n")
}
