package transpiler

import "testing"

func TestCampaignInheritedMemberLookupIdentity(t *testing.T) {
	helper := setupParseHelper(t, `
interface Root { class Mark {} }
interface Left extends Root {}
interface Right extends Root {}
class Outer {
 class Mark {}
 class Child implements Left, Right {}
 class Hidden extends Child { class Mark {} }
}
`)
	root := helper.File.Symbols.FindClassScope("Root")
	child := helper.File.Symbols.FindClassScope("Child")
	hidden := helper.File.Symbols.FindClassScope("Hidden")
	if root == nil || child == nil || hidden == nil {
		t.Fatal("missing source scopes")
	}
	for repeat := 0; repeat < 3; repeat++ {
		ctx := helper.Ctx.Clone()
		ctx.currentClass = child
		if got := resolveClassScopeByQualifiedName(ctx, "Mark"); got != root.Subclasses[0] {
			t.Fatalf("inherited diamond selected %v instead of Root.Mark", got)
		}
		if got := resolveClassScopeByQualifiedName(ctx, "Missing"); got != nil {
			t.Fatalf("unexpected member %v", got)
		}
		ctx.currentClass = hidden
		if got := resolveClassScopeByQualifiedName(ctx, "Mark"); got != hidden.Subclasses[0] {
			t.Fatalf("direct member did not hide inherited member: %v", got)
		}
		if len(ctx.memberTypeLookupPath) != 0 {
			t.Fatal("lookup guard escaped into caller context")
		}
	}
}

func TestCampaignInheritedMemberSwitchQualifierJVM(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>inherited-member</artifactId><version>1</version></project>`,
		"src/main/java/review/inherited/Main.java": `package review.inherited;
public class Main {
 static int effects;
 static class Unrelated { class Inner { String read(){return "wrong";} } }
 static class Base<T> {
  T value; Base(T value){this.value=value;}
  class Inner {String read(){return String.valueOf(value);}}
 }
 static class Child extends Base<String> {
  Child(String value){super(value);}
  String inherited(){return new Inner().read();}
 }
 static Child choose(Child child){effects++;return child;}
 public static void main(String[] args){
  Child left=new Child("left"),right=new Child("right");
  System.out.println((switch(1){case 0 -> choose(left);default -> choose(right);}).new Inner().read()+":"+effects);
  System.out.println(left.inherited()+":"+right.inherited());
 }
}
`}, "review.inherited.Main", "right:1\nleft:right\n")
}
