package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
	"testing"
)

func TestObjectsStaticImport516WildcardArgument(t *testing.T) {
	const source = `import static java.util.Objects.requireNonNull;
public class Use {
 static void guard(Class<?> rawType) { requireNonNull(rawType); }
}`
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	ctx.localScope = ctx.currentClass.FindMethodByName("guard", nil)
	if ctx.localScope == nil {
		t.Fatal("actual source guard declaration missing")
	}
	var invocation *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" {
			invocation = n
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(n) {
			visit(child)
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if invocation == nil {
		t.Fatal("actual source invocation missing")
	}
	arg := invocationArgumentNode(invocation, 0)
	var origin inferredJavaTypeOrigin
	typ, known := inferExprJavaType(arg, ctx, []byte(source), &origin)
	origin.javaType = typ
	t.Logf("actual type=%q known=%v origin=%+v imports=%+v", typ, known, origin, staticMethodImports(ctx))
	for _, def := range staticIntrinsicImportSignatures[intrinsicKey{"Objects", "requireNonNull"}] {
		formals := methodFormalOrigins(def, nil, nil, ctx)
		score, applies := scoreMethodCandidate(def, nil, methodCandidateTypeParameterNames(nil, def), []*sitter.Node{arg}, ctx, []byte(source))
		constraints, bounds := methodInvocationConstraintBindings(def, nil, []*sitter.Node{arg}, false, ctx, []byte(source))
		t.Logf("signature=%v parameter=%p formal=%+v score=%+v applies=%v bindings=%v bounds=%v", def.OriginalParameterTypes(), def.TypeParameters[0].Declaration, formals, score, applies, constraints, bounds)
	}
	got := resolveStaticImportedMethod(invocation, ctx, []byte(source))
	t.Logf("resolved=%+v", got)
	if got.problem != "" || got.intrinsic != "Objects" || got.source != nil {
		t.Fatalf("JDK-valid actual Class<?> argument must select canonical Objects.requireNonNull: %+v", got)
	}
	if result, ok := inferIntrinsicMethodResultType(invocation, ctx, []byte(source)); !ok || result != typ {
		t.Fatalf("canonical generic result must preserve actual type, got %q/%v", result, ok)
	}
}
