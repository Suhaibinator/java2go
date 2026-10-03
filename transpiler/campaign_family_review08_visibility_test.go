package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestCampaignFamilyReview08DeclarationVisibility(t *testing.T) {
	helper := setupParseHelper(t, `interface Slot<T>{T get();} interface Factory{<T> Slot<T> make();} class Host{void first(){class Slot{}new Slot();}void second(){}}`)
	host := helper.File.Symbols.FindClassScope("Host")
	global := helper.File.Symbols.FindClassScope("Slot")
	var first, second *symbol.Definition
	for _, method := range host.Methods {
		switch method.OriginalName {
		case "first":
			first = method
		case "second":
			second = method
		}
	}
	var localNode *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node.Type() == "class_declaration" {
			localNode = node
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(first.DeclarationNode)
	if localNode == nil {
		t.Fatal("missing local declaration")
	}
	local := &symbol.ClassScope{Class: &symbol.Definition{Name: "HoistedSlot", OriginalName: "Slot", DeclarationNode: localNode}}
	ctx := helper.Ctx.Clone()
	ctx.currentClass = host
	ctx.localScope = first
	ctx.localClasses = map[string]*localClassInfo{"Slot": {scope: local}}
	if got := resolveClassScopeByQualifiedName(ctx, "Slot"); got != local {
		t.Fatal("local declaration must resolve inside its declaring method")
	}
	ctx.localScope = second
	if got := resolveClassScopeByQualifiedName(ctx, "Slot"); got != global {
		t.Fatal("local declaration leaked into a different method")
	}
	if got := resolveClassScopeByQualifiedName(classScopeCtx(global, ctx), "Slot"); got != global {
		t.Fatal("local declaration leaked into declaration-bound inventory lookup")
	}
	ctx.localScope = first
	plans := discoverCanonicalGenericFamilies(ctx)
	if len(plans) != 1 {
		t.Fatalf("polluted caller suppressed global generic family: %d plans", len(plans))
	}
	if _, present := plans[0].members[global]; !present {
		t.Fatal("family did not retain the global Slot declaration")
	}
}
