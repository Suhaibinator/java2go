package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

const genericNominalReturnSamePackageSource = `class Base{}class Child extends Base{}class T2 extends Base{int code(){return 22;}}class T3 extends Base{int code(){return 33;}}
class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}}<T extends Base>T2 named(Cell<T> value){return new T2();}<T extends Base>T3 third(Cell<T> value){return new T3();}}
public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(new Child());T2 second=owner.named(cell);T3 third=owner.third(cell);System.out.println("returns="+second.code()+":"+third.code());}}
`

func TestGenericNominalReturnSamePackageJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, genericNominalReturnSamePackageSource, "returns=22:33\n")
}

func TestGenericNominalReturnPublicSamePackageJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                 `<project><groupId>probe</groupId><artifactId>nominal-public</artifactId><version>1</version></project>`,
		"src/main/java/Base.java": `public class Base{}`,
		"src/main/java/T2.java":   `public class T2 extends Base{public int code(){return 22;}}`,
		"src/main/java/T3.java":   `public class T3 extends Base{public int code(){return 33;}}`,
		"src/main/java/Main.java": `class Child extends Base{}class Owner<T>{class Cell<U>{U value;Cell(U value){this.value=value;}}<T extends Base>T2 named(Cell<T> value){return new T2();}<T extends Base>T3 third(Cell<T> value){return new T3();}}public class Main{public static void main(String[]args){Owner<String> owner=new Owner<String>();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(new Child());T2 second=owner.named(cell);T3 third=owner.third(cell);System.out.println("returns="+second.code()+":"+third.code());}}`,
	}, "Main", "returns=22:33\n")
}
func TestGenericNominalReturnSamePackageDeclarationNamespaces(t *testing.T) {
	helper := setupParseHelper(t, genericNominalReturnSamePackageSource)
	owner := helper.File.Symbols.FindClassScope("Owner")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethodByName("main", nil)
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name").Content(helper.File.Source)
			if name == "named" || name == "third" {
				seen++
				def := owner.FindMethodByName(name, nil)
				want := "T2"
				if name == "third" {
					want = "T3"
				}
				if got, known := inferredGenericInvocationResult(def, n, ctx, helper.File.Source); !known || got != want {
					t.Errorf("nominal semantic result=%s/%v want %s", got, known, want)
				}
				if got := genericMethodErasedJavaType(def, def.OriginalType); got != want {
					t.Errorf("nominal erased signature=%s want%s", got, want)
				}
				for _, scopeName := range []string{"T2", "T3"} {
					scope := helper.File.Symbols.FindClassScope(scopeName)
					if def.TypeParameters[0].EmittedName() == scope.Class.OriginalName || def.TypeParameters[0].EmittedName() == scope.Class.Name {
						t.Errorf("method alias %s collides with nominal Java/Go namespace %s/%s", def.TypeParameters[0].EmittedName(), scope.Class.OriginalName, scope.Class.Name)
					}
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 2 {
		t.Fatalf("calls=%d want2", seen)
	}
}

func TestGenericNominalReturnPrivateProjectPendingJDK21(t *testing.T) {
	for _, pkg := range []string{"", "aliasprivate"} {
		t.Run("package="+pkg, func(t *testing.T) {
			source := genericNominalReturnSamePackageSource
			name := "Main"
			if pkg != "" {
				source = "package " + pkg + ";" + source
				name = pkg + ".Main"
			}
			runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><groupId>probe</groupId><artifactId>nominal-private</artifactId><version>1</version></project>`, "src/main/java/Main.java": source}, name, "returns=22:33\n")
		})
	}
}
