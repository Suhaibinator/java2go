package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

const innerCarrierReturnTargetSource = `class Carrier<T> {
 <T> T make(){return null;}
 <T> T[] array(){return null;}
 <T> java.util.List<T> list(){return null;}
 <U> T classOnly(){return null;}
}
class Use {
 static <T> void run(Carrier<Integer> owner) {
  String scalar=owner.make();String[] array=owner.array();java.util.List<String> nested=owner.list();Integer carrier=owner.classOnly();
 }
}`

func TestInnerCarrierInferenceShadowedReturnTargets(t *testing.T) {
	helper := setupParseHelper(t, innerCarrierReturnTargetSource)
	owner := helper.File.Symbols.FindClassScope("Carrier")
	caller := helper.File.Symbols.FindClassScope("Use")
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = caller, caller.FindMethodByName("run", nil)
	tests := map[string]string{"make": "String", "array": "String[]", "list": "java.util.List<String>", "classOnly": "Integer"}
	seen := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name").Content(helper.File.Source)
			if expected, ok := tests[name]; ok {
				seen++
				def := owner.FindMethodByName(name, nil)
				if def.TypeParameters[0].Declaration == owner.TypeParameters[0].Declaration {
					t.Fatal("method/class declaration collapsed")
				}
				if def.TypeParameters[0].Name == "T" && def.TypeParameters[0].EmittedName() == owner.TypeParameters[0].EmittedName() {
					t.Fatal("same source spelling must have distinct aliases")
				}
				target := ctx.Clone()
				target.expectedType = expected
				target.expectedTypeRoot = n
				got := genericArrayInvocationTypeBindings(def, n, target, helper.File.Source)
				if name == "classOnly" {
					if len(got) != 0 {
						t.Errorf("class-only target became method variable: %v", got)
					}
				} else if got["T"] != "String" {
					t.Errorf("%s target=%s binding=%v,want String", n.Content(helper.File.Source), expected, got)
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if seen != len(tests) {
		t.Fatalf("actual target calls%d,want%d", seen, len(tests))
	}
}

func TestInnerCarrierInferenceOwnerlessReturnTarget(t *testing.T) {
	helper := setupParseHelper(t, `class Use{static void run(){String[] value=fake();}}`)
	caller := helper.File.Symbols.FindClassScope("Use")
	ctx := helper.Ctx.Clone()
	ctx.currentClass, ctx.localScope = caller, caller.FindMethodByName("run", nil)
	outer := symbol.NewTypeParam("T", nil)
	method := symbol.NewTypeParam("T", nil)
	symbol.DisambiguateTypeParamGoNames([]symbol.TypeParam{outer, method})
	def := &symbol.Definition{OriginalType: "T[]", TypeParameters: []symbol.TypeParam{method}}
	if _, owner := invocationMethodDeclarationContext(def, ctx); owner != nil {
		t.Fatal("synthetic definition unexpectedly source-owned")
	}
	seen := false
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "method_invocation" {
			seen = true
			target := ctx.Clone()
			target.expectedType = "String[]"
			target.expectedTypeRoot = n
			got := genericArrayInvocationTypeBindings(def, n, target, helper.File.Source)
			if got["T"] != "String" {
				t.Errorf("ownerless array return target binding=%v,want String", got)
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if !seen {
		t.Fatal("actual ownerless target call missing")
	}
}

func TestInnerCarrierInferenceReturnTargetsJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {public static void main(String[]args){Carrier<Integer> owner=new Carrier<Integer>();String scalar=owner.make();String[] array=owner.array();java.util.List<String> nested=owner.list();Integer carrier=owner.classOnly();System.out.print((scalar==null)+":"+(array==null)+":"+(nested==null)+":"+(carrier==null));}}`, "true:true:true:true", map[string]string{"Carrier.java": innerCarrierReturnTargetSource})
}
