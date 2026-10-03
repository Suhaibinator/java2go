package transpiler

import (
	"fmt"
	"go/ast"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestCallableSubclassInventory_WholeASTQueriesArePerRender(t *testing.T) {
	const methods = 64
	var declarations strings.Builder
	declarations.WriteString("class Root {} class Cell<T> extends Root {")
	for index := 0; index < methods; index++ {
		fmt.Fprintf(&declarations, "public T get%d(){return null;}", index)
	}
	declarations.WriteString("}")
	helper := setupParseHelper(t, declarations.String())
	var unrelated strings.Builder
	for index := 0; index < 64; index++ {
		fmt.Fprintf(&unrelated, "class Unrelated%d {}", index)
	}
	unrelated.WriteString("class Padding {void noise(){int value=0;")
	for index := 0; index < 4096; index++ {
		unrelated.WriteString("value++;")
	}
	unrelated.WriteString("}}")
	padding := parsing.SourceFile{Name: "Padding.java", Source: []byte(unrelated.String())}
	if err := padding.ParseAST(); err != nil {
		t.Fatal(err)
	}
	padding.ParseSymbols()
	symbol.AddSymbolsToPackage(padding.Symbols)
	ResolveFiles([]parsing.SourceFile{helper.File, padding})
	target := helper.File.Symbols.FindClassScope("Cell")
	if target == nil || len(target.TypeParameters) != 1 || len(allSourceClassScopes()) != 67 {
		t.Fatal("complete legal67class graph was not retained")
	}
	declaration := target.TypeParameters[0].Declaration
	selectors := map[string]bool{}
	for _, method := range target.Methods {
		if !strings.HasPrefix(method.OriginalName, "get") {
			continue
		}
		if method.OriginalType != "T" || method.TypeParameterBindings["T"] != declaration {
			t.Fatal("getter result lost the class-owned declaration identity")
		}
		selectors[method.Name] = true
	}
	if len(selectors) != methods {
		t.Fatalf("getters=%d want64", len(selectors))
	}
	render := func() ast.Node {
		ctx := helper.Ctx.Clone()
		ctx.genericFamilies = &genericFamilyAnalysis{}
		node, err := convertFileNode(helper.File, ctx)
		if err != nil {
			t.Fatal(err)
		}
		return node
	}
	t.Run("rendered-selectors-and-declaration-identity", func(t *testing.T) {
		if classHasUnmodeledCallableSubclass(target, helper.Ctx) {
			t.Fatal("named-only graph has an unmodeled subclass")
		}
		emitted := map[string]bool{}
		ast.Inspect(render(), func(node ast.Node) bool {
			if function, ok := node.(*ast.FuncDecl); ok && function.Recv != nil && selectors[function.Name.Name] {
				emitted[function.Name.Name] = true
			}
			return true
		})
		if len(emitted) != methods || target.TypeParameters[0].Declaration != declaration {
			t.Fatalf("rendered selectors=%d want64; source binder must remain identical", len(emitted))
		}
	})
	t.Run("inventory-budget", func(t *testing.T) {
		oneScan := testing.AllocsPerRun(1, func() {
			if classHasUnmodeledCallableSubclass(target, helper.Ctx) {
				t.Fatal("unexpected subclass")
			}
		})
		rendered := testing.AllocsPerRun(1, func() { _ = render() })
		t.Logf("same67class/64getter graph: single subclass scan allocations=%.0f render allocations=%.0f ratio=%.2f", oneScan, rendered, rendered/oneScan)
		if oneScan == 0 || rendered > 6*oneScan {
			t.Fatalf("CALLABLE_SUBCLASS_REPEATED_WHOLE_AST_SCAN: %.0f allocations >6*single scan %.0f", rendered, oneScan)
		}
	})
	t.Run("anonymous-local-and-inherited-subclasses", func(t *testing.T) {
		h := setupParseHelper(t, `class Base<T>{} class Child extends Base<String>{} class Other{} class Use{Base<String> make(){return new Child(){};} void local(){class Local extends Child{}}}`)
		base, child, other := h.File.Symbols.FindClassScope("Base"), h.File.Symbols.FindClassScope("Child"), h.File.Symbols.FindClassScope("Other")
		if !classHasUnmodeledCallableSubclass(base, h.Ctx) || !classHasUnmodeledCallableSubclass(child, h.Ctx) || classHasUnmodeledCallableSubclass(other, h.Ctx) || classHasUnmodeledCallableSubclass(nil, h.Ctx) {
			t.Fatal("anonymous/local superclass edges or ancestor queries changed")
		}
	})
	t.Run("fresh-graph-named-only", func(t *testing.T) {
		h := setupParseHelper(t, `class Base<T>{} class Child extends Base<String>{} class Use{static class Named extends Child{}}`)
		if classHasUnmodeledCallableSubclass(h.File.Symbols.FindClassScope("Base"), h.Ctx) || classHasUnmodeledCallableSubclass(h.File.Symbols.FindClassScope("Child"), h.Ctx) {
			t.Fatal("fresh named-only graph inherited an unmodeled subclass")
		}
	})
}
