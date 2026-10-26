package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestSourceArrayListParameterRankUnknownAndForeignDeclaration(t *testing.T) {
	source := `int marker;public boolean add(Object value){return super.add(value);}`
	_, owner, method, ctx := sourceArrayListParameterRankFixture(t, source, "add")
	positive := func() {
		t.Helper()
		if !sourceArrayListMethodOwned(method, owner, ctx) || !sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx) || sourceArrayListRoot(owner, ctx) != owner {
			t.Fatal("actual ordinary source premise lost before or after unknown-rank guard")
		}
	}
	positive()
	originalGraph := symbol.GlobalScope
	foreign, foreignOwner, foreignMethod, foreignCtx := sourceArrayListParameterRankFixture(t, source, "add")
	if !sourceArrayListMethodOwned(foreignMethod, foreignOwner, foreignCtx) || !sourceArrayListSlotMatches(foreignMethod, []string{"java.lang.Object"}, foreignCtx) || sourceArrayListRoot(foreignOwner, foreignCtx) != foreignOwner {
		t.Fatal("foreign tree did not establish its own ordinary source positive")
	}
	if !sameSourceNode(method.DeclarationNode, foreignMethod.DeclarationNode) || method.DeclarationNode.Equal(foreignMethod.DeclarationNode) {
		t.Fatal("foreign declaration control must be equal-span independent tree")
	}
	symbol.GlobalScope = originalGraph
	positive()
	wrongFile := ctx.Clone()
	wrongFile.currentFile = foreign.File.Symbols
	if sourceArrayListMethodOwned(method, owner, wrongFile) || sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, wrongFile) {
		t.Fatal("independent file context donated an actual formal rank")
	}
	originalNode, originalParameter, originalParameters := method.DeclarationNode, method.Parameters[0], method.Parameters
	if len(owner.Fields) != 1 || owner.Fields[0].DeclarationNode == nil {
		t.Fatal("actual field declaration premise missing")
	}
	for _, row := range []struct {
		name   string
		mutate func() func()
	}{
		{"missing declaration", func() func() { method.DeclarationNode = nil; return func() { method.DeclarationNode = originalNode } }},
		{"foreign rebound declaration", func() func() {
			method.DeclarationNode = foreignMethod.DeclarationNode
			return func() { method.DeclarationNode = originalNode }
		}},
		{"field masquerading as method", func() func() {
			method.DeclarationNode = owner.Fields[0].DeclarationNode
			return func() { method.DeclarationNode = originalNode }
		}},
		{"nil formal metadata", func() func() { method.Parameters[0] = nil; return func() { method.Parameters[0] = originalParameter } }},
		{"parameter count mismatch", func() func() { method.Parameters = nil; return func() { method.Parameters = originalParameters } }},
		{"forged prefix array metadata", func() func() {
			old := method.Parameters[0].OriginalType
			method.Parameters[0].OriginalType = "Object[]"
			return func() { method.Parameters[0].OriginalType = old }
		}},
		{"forged parameter name", func() func() {
			old := method.Parameters[0].OriginalName
			method.Parameters[0].OriginalName = "donor"
			return func() { method.Parameters[0].OriginalName = old }
		}},
		{"variadic metadata on scalar declaration", func() func() {
			old := method.IsVariadic
			method.IsVariadic = true
			return func() { method.IsVariadic = old }
		}},
	} {
		positive()
		restore := row.mutate()
		owned := sourceArrayListMethodOwned(method, owner, ctx)
		scalar := sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx)
		array := sourceArrayListSlotMatches(method, []string{"java.lang.Object[]"}, ctx)
		admitted := sourceArrayListRoot(owner, ctx) != nil
		restore()
		positive()
		if owned || scalar || array || admitted {
			t.Fatalf("unknown or unattached parameter rank donated authority (%s): owned=%v scalar=%v array=%v admitted=%v", row.name, owned, scalar, array, admitted)
		}
	}
}

func TestSourceArrayListParameterRankStrictRefusalAndBusinessCollision(t *testing.T) {
	for _, row := range []struct{ name, method string }{
		{"add", `public boolean add(Object value[]){return super.add(value);}`},
		{"add", `public boolean add(Object value[][]){return super.add(value);}`},
		{"add", `public boolean add(Object... values){return super.add(values);}`},
		{"toArray", `public Object[] toArray(Object value[]){return value;}`},
	} {
		h, owner, _, ctx := sourceArrayListParameterRankFixture(t, row.method, row.name)
		ctx.className = owner.Class.Name
		strictRoutingState(t)
		ctx.importAliases = map[string]string{"kept": "kept"}
		ctx.usedImports = map[string]bool{"kept": true}
		var hoisted []ast.Decl
		ctx.hoistedDecls = &hoisted
		anonymous, labels := 7, 11
		ctx.anonClassCounter = &anonymous
		ctx.controlLabelCounter = &labels
		var output []ast.Decl
		var observed any
		func() {
			defer func() { observed = recover() }()
			output = parseDecls(owner.Class.DeclarationNode, h.File.Source, ctx)
		}()
		failure, ok := observed.(strictModeError)
		if !ok || failure.diagnostic.Kind != "ArrayList source capability" || output != nil || len(hoisted) != 0 || anonymous != 7 || labels != 11 || !reflect.DeepEqual(ctx.importAliases, map[string]string{"kept": "kept"}) || !reflect.DeepEqual(ctx.usedImports, map[string]bool{"kept": true}) {
			t.Fatalf("unsupported formal rank emitted partial AST or lacked strict capability refusal: panic=%T %v output=%v", observed, observed, output)
		}
	}
	h := setupParseHelper(t, `class Object{}class C extends java.util.ArrayList<java.lang.Object>{public boolean add(Object value){return false;}}`)
	owner := h.Ctx.currentFile.FindClassScope("C")
	method := owner.FindMethodByName("add", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	selected, _ := sourceArrayListSelectedAdd(owner, ctx)
	if sourceArrayListRoot(owner, ctx) != owner || !sourceArrayListMethodOwned(method, owner, ctx) || sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, ctx) || selected == method {
		t.Fatal("same-spelled source Object business overload acquired canonical native Object slot or was newly admitted as an array business method")
	}
}
