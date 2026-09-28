package transpiler

import "testing"

func TestCampaignRelativeMemberTypesAcrossFilesJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>relative-member</artifactId><version>1</version></project>`,
		"src/main/java/review/member/Generator.java": `package review.member;
public class Generator {
 public Result generate(){return new Result();}
 public static class Result { public String getValue(){return "same-package";} }
}
`,
		"src/main/java/review/member/Noise.java": `package review.member;
class Noise { static class Result { public String getValue(){return "wrong";} } }
`,
		"src/main/java/other/Holder.java": `package other;
public class Holder {
 public Result generate(){return new Result();}
 public static class Result { public String getValue(){return "imported-owner";} }
}
`,
		"src/main/java/review/member/Main.java": `package review.member;
import other.Holder;
public class Main {
 public static void main(String[] args){
  Generator.Result first=new Generator().generate();
  Holder.Result second=new Holder().generate();
  other.Holder.Result qualified=second;
  System.out.println(first.getValue()+":"+second.getValue()+":"+qualified.getValue());
 }
}
`}, "review.member.Main", "same-package:imported-owner:imported-owner\n")
}

func TestCampaignRelativeMemberTypesLexicalRootJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>relative-member-shadow</artifactId><version>1</version></project>`,
		"src/main/java/other/Holder.java": `package other;
public class Holder {
 public Result generate(){return new Result();}
 public static class Result {public String getValue(){return "external";}}
}
`,
		"src/main/java/review/member/Holder.java": `package review.member;
public class Holder {
 public Result generate(){return new Result();}
 public static class Result {public String getValue(){return "same-package";}}
}
`,
		"src/main/java/review/member/Main.java": `package review.member;
import other.Holder;
public class Main {
 static class Scope {
  static class Holder {
   Result generate(){return new Result();}
   static class Result {String getValue(){return "lexical";}}
  }
  static String read(){
   Holder.Result local=new Holder().generate();
   other.Holder.Result external=new other.Holder().generate();
   return local.getValue()+":"+external.getValue();
  }
 }
 public static void main(String[] args){Holder.Result imported=new Holder().generate();System.out.println(Scope.read()+":"+imported.getValue());}
}
`}, "review.member.Main", "lexical:external:external\n")
}
