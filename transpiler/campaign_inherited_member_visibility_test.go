package transpiler

import "testing"

func TestCampaignInheritedMemberVisibilityJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>inherited-visibility</artifactId><version>1</version></project>`,
		"src/main/java/base/Base.java": `package base;
public class Base {
 private static class PrivateMark { String text(){return "private-parent";} }
 static class PackageMark { String text(){return "package-parent";} }
 protected static class ProtectedMark { public ProtectedMark(){} public String text(){return "protected-parent";} }
}
`,
		"src/main/java/bridge/Bridge.java": `package bridge; public class Bridge extends base.Base {}`,
		"src/main/java/base/Returned.java": `package base;
public class Returned {
 static class PackageMark { String text(){return "returned-enclosing";} }
 static class Child extends bridge.Bridge { String read(){return new PackageMark().text();} }
 public static String read(){return new Child().read();}
}
`,
		"src/main/java/review/visibility/Main.java": `package review.visibility;
import base.Base;
import base.Returned;
public class Main {
 static class PrivateMark { String text(){return "private-enclosing";} }
 static class PackageMark { String text(){return "package-enclosing";} }
 static class ProtectedMark { String text(){return "protected-enclosing";} }
 static class Child extends Base {
  String read(){return new PrivateMark().text()+":"+new PackageMark().text()+":"+new ProtectedMark().text();}
 }
 public static void main(String[] args){System.out.println(new Child().read()+":"+Returned.read());}
}
`}, "review.visibility.Main", "private-enclosing:package-enclosing:protected-parent:returned-enclosing\n")
}

func TestCampaignNestedSwitchTargetJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>nested-switch</artifactId><version>1</version></project>`,
		"src/main/java/review/switchtarget/Main.java": `package review.switchtarget;
public class Main {
 static long read(int choice){return switch(0){case 0 -> switch(choice){case 0 -> 1;default -> 2L;};default -> 3L;};}
 public static void main(String[] args){System.out.println(read(0)+":"+read(1));}
}
`}, "review.switchtarget.Main", "1:2\n")
}

func TestCampaignSwitchVarResultJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>switch-var</artifactId><version>1</version></project>`,
		"src/main/java/review/switchvar/Main.java": `package review.switchvar;
public class Main {
 public static void main(String[] args){
  var text=switch(1){case 0 -> "left";default -> "right";};
  int total=0;
  for(var index=switch(1){case 0 -> 1;default -> 2;};index>0;index--){total+=index;}
  System.out.println(text+":"+total);
 }
}
`}, "review.switchvar.Main", "right:3\n")
}

func TestCampaignInheritedMemberLocalPackageJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>local-member-package</artifactId><version>1</version></project>`,
		"src/main/java/review/localmember/Main.java": `package review.localmember;
class Parent {
 class Mark {String text(){return "inherited-package";}}
}
public class Main {
 static class Mark {String text(){return "enclosing";}}
 static String read(){
  class Local extends Parent {String text(){return new Mark().text();}}
  return new Local().text();
 }
 public static void main(String[] args){System.out.println(read());}
}
`}, "review.localmember.Main", "inherited-package\n")
}
