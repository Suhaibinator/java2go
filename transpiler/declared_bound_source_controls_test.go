package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

var declaredBoundSourceControls = []struct{ name, setup, actual, formal string }{
	{"source-interface", "interface Face{} class Cell<D extends Face>{}", "Cell<?>", "Cell<? extends Face>"},
	{"interface-owner", "interface Face{} interface Cell<D extends Face>{}", "Cell<? extends Face>", "Cell<?>"},
	{"intersection", "class Base{} interface Face{} class Cell<D extends Base & Face>{}", "Cell<?>", "Cell<? extends Base>"},
	{"dependent-slot", "class Pair<A,B extends A>{}", "Pair<Number,?>", "Pair<Number,? extends Number>"},
	{"reordered-slots", "class Pair<B extends A,A>{}", "Pair<?,Number>", "Pair<? extends Number,Number>"},
	{"source-homonym", "interface GenericDeclaration{} class TypeVariable<D extends GenericDeclaration>{}", "TypeVariable<?>", "TypeVariable<? extends GenericDeclaration>"},
	{"top-variance", "interface Face{} class Cell<D extends Face>{}", "Cell<?>", "Cell<? extends Face>"},
	{"lower-positive", "", "java.util.List<? super Number>", "java.util.List<? super Number>"},
}

func declaredBoundControlSource(setup, actual, formal string) string {
	return "package p; " + setup + " public class Use{void take(java.util.Map<" + formal + ",Integer> value){} void test(java.util.Map<" + actual + ",Integer> value){take(value);}}"
}

func TestDeclaredBoundSourcePositiveControls(t *testing.T) {
	for _, r := range declaredBoundSourceControls {
		t.Run(r.name, func(t *testing.T) {
			actual, formal, ctx, args, target := declaredBoundControlOrigins(t, declaredBoundControlSource(r.setup, r.actual, r.formal))
			_, _, _, handled := declaredOriginConversion(actual, formal, ctx)
			_, applies := scoreMethodCandidate(target, ctx.currentClass, methodCandidateTypeParameterNames(ctx.currentClass, target), args, ctx, ctx.currentFile.Source)
			if !applies {
				t.Fatalf("JDK-valid declaration-bound call rejected, handled=%t actual=%s formal=%s", handled, gsonParentDiagnosticOrigin(actual, nil), gsonParentDiagnosticOrigin(formal, nil))
			}
		})
	}
}

func declaredBoundControlOrigins(t *testing.T, source string) (inferredJavaTypeOrigin, inferredJavaTypeOrigin, Ctx, []*sitter.Node, *symbol.Definition) {
	t.Helper()
	ctx := systematicNegativeFullPipelineContext(t, source)
	var caller, target *symbol.Definition
	for _, d := range ctx.currentClass.Methods {
		if d.OriginalName == "test" {
			caller = d
		}
		if d.OriginalName == "take" {
			target = d
		}
	}
	if caller == nil || target == nil {
		t.Fatal("actual registered declarations absent")
	}
	ctx.localScope = caller
	var call *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" {
			call = n
		}
		for _, c := range nodeutil.SemanticNamedChildrenOf(n) {
			visit(c)
		}
	}
	visit(caller.DeclarationNode.ChildByFieldName("body"))
	if call == nil {
		t.Fatal("actual invocation absent")
	}
	args := nodeutil.SemanticNamedChildrenOf(call.ChildByFieldName("arguments"))
	var actual inferredJavaTypeOrigin
	typ, known := inferExprJavaType(args[0], ctx, ctx.currentFile.Source, &actual)
	actual.javaType = typ
	if !known {
		t.Fatal("actual origin absent")
	}
	return actual, methodFormalOrigins(target, ctx.currentClass, nil, ctx)[0], ctx, args, target
}
