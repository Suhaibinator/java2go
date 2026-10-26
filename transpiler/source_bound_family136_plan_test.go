package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestSourceBound136FamilyRequiresCompleteDeclarationProof(t *testing.T) {
	h := setupParseHelper(t, `class Item{int key;}class Other extends Item{}
 class Box<T extends Item> implements java.lang.Comparable<T>{T value;Box(T value){this.value=value;}T get(){return value;}void put(T value){this.value=value;}public int compareTo(T value){return value.key;}<T>T shadow(T value){return value;}}`)
	owner := h.File.Symbols.FindClassScope("Box")
	plan, err := planGenericFamily(owner, h.Ctx)
	if err != nil || plan == nil {
		t.Fatalf("complete source-bound family declined: %v", err)
	}
	if len(plan.representations) != 1 {
		t.Fatalf("method shadow joined class slot: %#v", plan.representations)
	}
	for _, method := range owner.Methods {
		if method.OriginalName == "shadow" {
			if _, ok := plan.representations[method.TypeParameters[0].Declaration]; ok {
				t.Fatal("shadow method slot erased as class storage")
			}
		}
	}
}
func TestSourceBound136FamilyRefusesUnsupportedProofs(t *testing.T) {
	for _, src := range []string{
		`class Item{}class Box<T extends Item>{T value;}`,
		`interface Item{}class Box<T extends Item> implements java.lang.Comparable<T>{public int compareTo(T value){return 0;}}`,
		`class Item<U>{}class Box<T extends Item<String>> implements java.lang.Comparable<T>{public int compareTo(T value){return 0;}}`,
		`class Item{}class Box<T extends Item> implements java.lang.Comparable<T>{T[] values;public int compareTo(T value){return 0;}}`,
		`class Item{}class Box<T extends Item> implements java.lang.Comparable<T>{void put(T... values){}public int compareTo(T value){return 0;}}`,
		`class Item{}class Comparable<T>{}class Box<T extends Item> extends Comparable<T>{T value;}`,
	} {
		h := setupParseHelper(t, src)
		if plan, err := planGenericFamily(h.File.Symbols.FindClassScope("Box"), h.Ctx); plan != nil || err == nil {
			t.Fatalf("unsupported family admitted: %s", src)
		}
	}
}

func TestSourceBound136UnprovenInitializerRefused(t *testing.T) {
	h := setupParseHelper(t, `class Item{} class Box<T extends Item> implements java.lang.Comparable<T>{T value=(T)new Item();public int compareTo(T value){return 0;}}`)
	if plan, err := planGenericFamily(h.File.Symbols.FindClassScope("Box"), h.Ctx); plan != nil || err == nil {
		t.Fatal("initializer without a callable physical-body proof admitted")
	}
}

func TestSourceBound136IdentityRequiresActualCertifiedClassSlot(t *testing.T) {
	source := `class Item{}class Other extends Item{}
 class Box<T extends Item> implements java.lang.Comparable<T>{T value;static Other unrelated;Box(T value){this.value=value;}T get(){return value;}<U extends Item>U own(U value){return value;}public int compareTo(T value){return 0;}}
 class Use{boolean fields(Box<Other> box,Other value){return box.value==value;}boolean methods(Box<Other> box,Other value){return box.get()==value;}boolean methodOwned(Box<Other> box,Other value){return box.own(value)==value;}boolean staticField(Box<Other> box,Other value){return Box.unrelated==value;}boolean explicitCast(Box<Other> box,Other value){return (Other)box.get()==value;}}
 `
	h := setupParseHelper(t, source)
	use := h.File.Symbols.FindClassScope("Use")
	ctx := h.Ctx.Clone()
	ctx.currentClass = use
	for _, method := range use.Methods {
		node := findNode(method.DeclarationNode, "binary_expression")
		if node == nil {
			continue
		}
		ctx.localScope = method
		want := method.OriginalName == "fields" || method.OriginalName == "methods"
		if got := genericFamilySourceBoundIdentityOperand(node.ChildByFieldName("left"), ctx, h.File.Source); got != want {
			t.Fatalf("%s identity admission=%v want %v", method.OriginalName, got, want)
		}
	}
	box := h.File.Symbols.FindClassScope("Box")
	ctx.currentClass = box
	ctx.localScope = box.Methods[0]
	var get *symbol.Definition
	for _, method := range box.Methods {
		if method.OriginalName == "get" {
			get = method
		}
	}
	if get == nil {
		t.Fatal("missing get declaration")
	}
	ctx.localScope = get
	var valueNode *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "identifier" && node.Content(h.File.Source) == "value" {
			valueNode = node
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(get.DeclarationNode.ChildByFieldName("body"))
	if valueNode == nil || !genericFamilySourceBoundIdentityOperand(valueNode, ctx, h.File.Source) {
		t.Fatal("implicit certified field missing")
	}
	clone := *get
	clone.Children = append(clone.Children, &symbol.Definition{OriginalName: "value", OriginalType: "Other"})
	ctx.localScope = &clone
	if genericFamilySourceBoundIdentityOperand(valueNode, ctx, h.File.Source) {
		t.Fatal("local shadow admitted as field")
	}
}
