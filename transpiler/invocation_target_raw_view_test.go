package transpiler

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestResolveInvocationTargetPreservesRawReceiverProvenance(t *testing.T) {
	helper := setupParseHelper(t, `
public class InvocationRawViewProbe {
    interface Numbered { }

    static final class First implements Numbered { }

    static class Outer<T extends Numbered> {
        void touch() { }
    }

    static void probe(
            Outer rawParam,
            Outer<First> typedParam,
            Outer<?> wildcardParam) {
        Outer rawLocal = rawParam;
        Outer<First> typedLocal = typedParam;
        Outer<?> wildcardLocal = wildcardParam;

        rawParam.touch();
        typedParam.touch();
        wildcardParam.touch();
        rawLocal.touch();
        typedLocal.touch();
        wildcardLocal.touch();
    }
}
`)

	scope := helper.File.Symbols.FindClassScope("InvocationRawViewProbe")
	if scope == nil {
		t.Fatal("InvocationRawViewProbe scope was not parsed")
	}
	methods := scope.FindMethod().ByOriginalName("probe")
	if len(methods) != 1 {
		t.Fatalf("probe definitions = %d, want 1", len(methods))
	}

	ctx := helper.Ctx
	ctx.currentClass = scope
	ctx.localScope = methods[0]
	ctx.syntheticTypeParameters, ctx.rawGenericParameterTypes = synthesizeRawGenericFunctionParameters(methods[0], ctx)
	for _, rewrittenParameter := range []string{"rawParam", "wildcardParam"} {
		if rewritten := ctx.rawGenericParameterTypes[rewrittenParameter]; rewritten == "" {
			t.Fatalf("%s did not receive a synthesized representation", rewrittenParameter)
		}
	}

	targets := make(map[string]*invocationTargetInfo)
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			object := node.ChildByFieldName("object")
			if object != nil && object.Type() == "identifier" {
				targets[object.Content(helper.File.Source)] = resolveInvocationTarget(object, ctx, helper.File.Source)
			}
		}
		for index := 0; index < int(node.NamedChildCount()); index++ {
			visit(node.NamedChild(index))
		}
	}
	visit(helper.File.Ast)

	expected := map[string]bool{
		"rawParam":      true,
		"typedParam":    false,
		"wildcardParam": false,
		"rawLocal":      true,
		"typedLocal":    false,
		"wildcardLocal": false,
	}
	outer := helper.File.Symbols.FindClassScope("Outer")
	if outer == nil {
		t.Fatal("Outer scope was not parsed")
	}
	for receiver, wantRaw := range expected {
		target := targets[receiver]
		if target == nil {
			t.Fatalf("%s invocation target was not resolved", receiver)
		}
		if target.classScope != outer {
			t.Fatalf("%s target scope = %v, want Outer", receiver, target.classScope)
		}
		if target.rawGenericView != wantRaw {
			t.Errorf("%s rawGenericView = %t, want %t", receiver, target.rawGenericView, wantRaw)
		}
		if got := len(target.classTypeArgs); got != 1 {
			t.Errorf("%s normalized type argument count = %d, want 1", receiver, got)
		}
	}
}
