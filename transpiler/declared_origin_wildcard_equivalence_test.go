package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

const declaredOriginWildcardEquivalenceJava = `package p;
public class Use {
 static void takeBound(java.util.Map<java.lang.reflect.TypeVariable<? extends java.lang.reflect.GenericDeclaration>,java.lang.reflect.Type> value){}
 static void takeUnbound(java.util.Map<java.lang.reflect.TypeVariable<?>,java.lang.reflect.Type> value){}
 static void takeObjectList(java.util.Map<java.util.List<? extends Object>,java.lang.reflect.Type> value){}
 static void bound(java.util.Map<java.lang.reflect.TypeVariable<?>,java.lang.reflect.Type> value){takeBound(value);}
 static void unbound(java.util.Map<java.lang.reflect.TypeVariable<? extends java.lang.reflect.GenericDeclaration>,java.lang.reflect.Type> value){takeUnbound(value);}
 static void objectList(java.util.Map<java.util.List<?>,java.lang.reflect.Type> value){takeObjectList(value);}
}
`

func TestDeclaredOriginRefusalWildcardUpperEquivalence(t *testing.T) {
	for _, r := range []struct{ caller, target string }{{"bound", "takeBound"}, {"unbound", "takeUnbound"}, {"objectList", "takeObjectList"}} {
		t.Run(r.caller, func(t *testing.T) {
			ctx := systematicNegativeFullPipelineContext(t, declaredOriginWildcardEquivalenceJava)
			owner := ctx.currentClass
			var caller, target *symbol.Definition
			for _, d := range owner.Methods {
				if d.OriginalName == r.caller {
					caller = d
				}
				if d.OriginalName == r.target {
					target = d
				}
			}
			if caller == nil || target == nil {
				t.Fatal("actual original declaration missing")
			}
			ctx.localScope = caller
			var call *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "method_invocation" {
					call = n
				}
				for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
					walk(c)
				}
			}
			walk(caller.DeclarationNode.ChildByFieldName("body"))
			if call == nil {
				t.Fatal("exact original invocation absent")
			}
			args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
			var actual inferredJavaTypeOrigin
			typ, known := inferExprJavaType(args[0], ctx, ctx.currentFile.Source, &actual)
			actual.javaType = typ
			formal := methodFormalOrigins(target, owner, nil, ctx)[0]
			cost, exact, conversion, handled := declaredOriginConversion(actual, formal, ctx)
			score, applies := scoreMethodCandidate(target, owner, methodCandidateTypeParameterNames(owner, target), args, ctx, ctx.currentFile.Source)
			t.Logf("actual=%s formal=%s conversion=(%d,%t,%t,%t) score=%+v applies=%t", gsonParentDiagnosticOrigin(actual, nil), gsonParentDiagnosticOrigin(formal, nil), cost, exact, conversion, handled, score, applies)
			if !known || !applies {
				t.Error("JDK-valid wildcard upper-bound equivalence must remain applicable")
			}
		})
	}
}
