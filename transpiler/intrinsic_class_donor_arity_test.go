package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestClassIntrinsicDonorGenericArity(t *testing.T) {
	h := setupParseHelper(t, `class T2{}class foreign{static class Class<T>{}}class Provider<U extends java.lang.Class<T2>>{static java.lang.Class<T2> token;static java.lang.Class raw;static java.lang.Class<T2,T2> wrong;static java.lang.Class<Missing> unknown;static foreign.Class<T2> source;static absent.Class<T2> foreign;static java.lang.Class<?> wildcard;U capture;}class Main{static class java{static class lang{static class Class{}}}static<T2>void probe(Provider<java.lang.Class<T2>> p){Provider.token.getName();Provider.raw.getName();Provider.wrong.getName();Provider.unknown.getName();Provider.source.getName();Provider.foreign.getName();Provider.wildcard.getName();p.capture.getName();}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	rows := map[string]bool{"Provider.token": true, "Provider.raw": true, "Provider.wrong": false, "Provider.unknown": false, "Provider.source": false, "Provider.foreign": false, "Provider.wildcard": true, "p.capture": false}
	found := map[string]bool{}
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" {
			receiver := n.ChildByFieldName("object")
			if receiver != nil {
				if want, yes := rows[receiver.Content(h.File.Source)]; yes {
					name := receiver.Content(h.File.Source)
					selected, ok := intrinsicReceiverTypeName(receiver, ctx, h.File.Source)
					got := ok && selected == "Class"
					if got != want {
						t.Errorf("%s: owner=%q admitted=%v wantcanonical=%v", name, selected, ok, want)
					}
					found[name] = true
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	for name := range rows {
		if !found[name] {
			t.Errorf("missing original receiver %s", name)
		}
	}
}
