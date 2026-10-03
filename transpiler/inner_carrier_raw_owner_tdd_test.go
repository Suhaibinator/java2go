package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

const innerCarrierRawOwnerSource = `class Box<T> {
 class Member {}
 int raw(Box.Member value){return 1;} long raw(Object value){return 2L;}
 int implicit(Member value){return 3;} long implicit(Object value){return 4L;}
 int parameterized(Box<T>.Member value){return 5;} long parameterized(Object value){return 6L;}
}
class Probe {
 static String run(Box<String> owner,Box<String>.Member same,Box<Integer>.Member other){
 return owner.raw(same)+":"+owner.raw(other)+":"+owner.implicit(same)+":"+owner.implicit(other)+":"+owner.parameterized(same)+":"+owner.parameterized(other);
 }
}`

func TestInnerCarrierRawOwnerKeepsSourceQualification(t *testing.T) {
	helper := setupParseHelper(t, innerCarrierRawOwnerSource)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Probe")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	want := map[string]string{"owner.raw(same)": "int", "owner.raw(other)": "int", "owner.implicit(same)": "int", "owner.implicit(other)": "long", "owner.parameterized(same)": "int", "owner.parameterized(other)": "long"}
	observed := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			text := node.Content(helper.File.Source)
			if expected, found := want[text]; found {
				observed[text] = true
				actual, known := inferExprJavaType(node, ctx, helper.File.Source)
				if !known || actual != expected {
					t.Errorf("%s infers %q/%t, want %s/true", text, actual, known, expected)
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(observed) != len(want) {
		t.Fatalf("observed %d raw qualification controls, want%d", len(observed), len(want))
	}
}

func TestInnerCarrierPartiallyQualifiedAndImportedOwners(t *testing.T) {
	const source = `package p; import p.Envelope.Middle;
class Envelope<T> {
 class Middle { class Leaf {} }
 int implicit(Middle.Leaf value){return 1;} long implicit(Object value){return 2L;}
 int raw(Envelope.Middle.Leaf value){return 3;} long raw(Object value){return 4L;}
 int parameterized(Envelope<T>.Middle.Leaf value){return 5;} long parameterized(Object value){return 6L;}
}
class Receiver { int imported(Middle.Leaf value){return 7;} long imported(Object value){return 8L;} }
class Probe {
 static String run(Envelope<String> owner,Receiver receiver,Envelope<String>.Middle.Leaf same,Envelope<Integer>.Middle.Leaf other){
 return owner.implicit(same)+":"+owner.implicit(other)+":"+owner.raw(same)+":"+owner.raw(other)+":"+owner.parameterized(same)+":"+owner.parameterized(other)+":"+receiver.imported(same)+":"+receiver.imported(other);
 }
}`
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Probe")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	want := map[string]string{"owner.implicit(same)": "int", "owner.implicit(other)": "long", "owner.raw(same)": "int", "owner.raw(other)": "int", "owner.parameterized(same)": "int", "owner.parameterized(other)": "long", "receiver.imported(same)": "int", "receiver.imported(other)": "int"}
	observed := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "method_invocation" {
			text := node.Content(helper.File.Source)
			if expected, found := want[text]; found {
				observed[text] = true
				actual, known := inferExprJavaType(node, ctx, helper.File.Source)
				if !known || actual != expected {
					t.Errorf("%s infers %q/%t, want %s/true", text, actual, known, expected)
				}
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			walk(node.NamedChild(index))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if len(observed) != len(want) {
		t.Fatalf("observed %d partial/imported controls, want%d", len(observed), len(want))
	}
}

func TestInnerCarrierNonRawOwnerJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {public static void main(String[]args){Box<String> owner=new Box<String>();Box<Integer> otherOwner=new Box<Integer>();Box<String>.Member same=owner.new Member();Box<Integer>.Member other=otherOwner.new Member();System.out.print(owner.raw(same)+":"+owner.implicit(same)+":"+owner.implicit(other)+":"+owner.parameterized(same)+":"+owner.parameterized(other));}}`, "1:3:4:5:6", map[string]string{"Box.java": `class Box<T> {
 class Member {}
 int raw(Box.Member value){return 1;} long raw(Object value){return 2L;}
 int implicit(Member value){return 3;} long implicit(Object value){return 4L;}
 int parameterized(Box<T>.Member value){return 5;} long parameterized(Object value){return 6L;}
}`})
}
