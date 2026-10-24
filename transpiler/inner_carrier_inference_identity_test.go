package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestHiddenLexicalReceiverSlotInterfaceBoundJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner=new Carrier<String>(); Carrier<Integer> otherOwner=new Carrier<Integer>();
  Carrier<String>.Cell<Integer> same=owner.new Cell<Integer>(); Carrier<Integer>.Cell<Integer> other=otherOwner.new Cell<Integer>();
  Carrier<String>.Cell<Child> sameBound=owner.new Cell<Child>(); Carrier<Integer>.Cell<Child> otherBound=otherOwner.new Cell<Child>();
  System.out.print(owner.implicit(same)+":"+owner.implicit(other)+":"+owner.explicit(same)+":"+owner.explicit(other)+":"+owner.shadow(sameBound)+":"+owner.shadow(otherBound));
 }
}`, "1:2:3:4:5:6", map[string]string{"Carrier.java": hiddenLexicalReceiverSource})
}

func TestHiddenLexicalReceiverSlotDependentBoundJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner=new Carrier<String>();
  System.out.print(owner.wildcard((Carrier<String>.Cell<?>)null)+":"+owner.raw((Carrier<String>.Cell<?>)null));
 }
}`, "11:12", map[string]string{"Carrier.java": hiddenLexicalDependentSource})
}

func TestInnerCarrierInferenceMethodDeclarationIdentity(t *testing.T) {
	source := hiddenLexicalReceiverSource + `
class Use {
 static <T> void run(Carrier<String> owner,Carrier<String>.Cell<Child> child,Carrier<Child>.Cell<Child> written) {
  owner.shadow(child); owner.<Child>shadow(child); owner.written(written);
 }
}`
	helper := setupParseHelper(t, source)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	caller := helper.File.Symbols.FindClassScope("Use")
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = caller, caller.FindMethodByName("run", nil)
	classParam := owner.TypeParameters[0]
	method := owner.FindMethodByName("shadow", nil)
	if classParam.Declaration == method.TypeParameters[0].Declaration || classParam.EmittedName() == method.TypeParameters[0].EmittedName() {
		t.Fatal("class/method declarations must differ")
	}
	want := map[string]string{"owner.shadow(child)": "Child", "owner.<Child>shadow(child)": "Child", "owner.written(written)": "Child"}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			if expected, ok := want[n.Content(helper.File.Source)]; ok {
				seen++
				def := owner.FindMethodByName(n.ChildByFieldName("name").Content(helper.File.Source), nil)
				_, declaring := invocationMethodDeclarationContext(def, ctx)
				if declaring != owner {
					t.Fatal("actual declaration owner missing")
				}
				got := genericArrayInvocationTypeBindings(def, n, ctx, helper.File.Source)
				if got[def.TypeParameters[0].Name] != expected {
					t.Errorf("%s binding=%v,want %s", n.Content(helper.File.Source), got, expected)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != len(want) {
		t.Fatalf("observed%d,want%d", seen, len(want))
	}
}

func TestInnerCarrierInferenceClassOnlySlotIsNotMethodVariable(t *testing.T) {
	helper := setupParseHelper(t, hiddenLexicalDependentSource+`
class Use { static <T> void run(Carrier<String> owner,Carrier<String>.Cell<?> value){owner.wildcard(value);owner.raw(value);owner.<Marker>wildcard(value);owner.<Marker>raw(value);} }`)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	caller := helper.File.Symbols.FindClassScope("Use")
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = caller, caller.FindMethodByName("run", nil)
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name").Content(helper.File.Source)
			if name == "wildcard" || name == "raw" {
				seen++
				def := owner.FindMethodByName(name, nil)
				got := genericArrayInvocationTypeBindings(def, n, ctx, helper.File.Source)
				expected := ""
				if n.ChildByFieldName("type_arguments") != nil {
					expected = "Marker"
				}
				if got[def.TypeParameters[0].Name] != expected {
					t.Errorf("%s binding=%v,want method binding %q (absent uses declared bound)", n.Content(helper.File.Source), got, expected)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != 4 {
		t.Fatalf("observed%d,want4", seen)
	}
}

func TestInnerCarrierInferenceOwnerlessSourceAlias(t *testing.T) {
	helper := setupParseHelper(t, `class Child {} class Use {static void run(Child value){fake(value);}}`)
	caller := helper.File.Symbols.FindClassScope("Use")
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = caller, caller.FindMethodByName("run", nil)
	outer := symbol.NewTypeParam("T", nil)
	method := symbol.NewTypeParam("T", nil)
	symbol.DisambiguateTypeParamGoNames([]symbol.TypeParam{outer, method})
	def := &symbol.Definition{TypeParameters: []symbol.TypeParam{method}, Parameters: []*symbol.Definition{{OriginalType: "T"}}, OriginalType: "void"}
	if _, owner := invocationMethodDeclarationContext(def, ctx); owner != nil {
		t.Fatal("synthetic method unexpectedly has source owner")
	}
	seen := false
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			seen = true
			got := genericArrayInvocationTypeBindings(def, n, ctx, helper.File.Source)
			if got["T"] != "Child" {
				t.Errorf("ownerless source alias binding=%v,want Child", got)
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if !seen {
		t.Fatal("ownerless actual caller missing")
	}
}
