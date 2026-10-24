package transpiler

import "testing"

func TestHelpfulNPECounterDetachedCallerDefinition(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target{void call(){}}class Main{static void run(Target lane){lane.call();}}`, "Main", "run", "call")
	resolved := helpfulNPEResolution(t, ctx, n, s)
	if helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s) == nil {
		t.Fatal("actual registered source control missing")
	}
	detached := *ctx.localScope
	ctx.localScope = &detached
	if plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s); plan != nil {
		t.Fatalf("copied caller borrowed source ownership: %s", plan.message)
	}
}
func TestHelpfulNPECounterDetachedCallerFormal(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target{void call(){}}class Main{static void run(Target lane){lane.call();}}`, "Main", "run", "call")
	resolved := helpfulNPEResolution(t, ctx, n, s)
	registered := ctx.localScope
	detached := *registered
	detached.Parameters = append(registered.Parameters[:0:0], registered.Parameters...)
	copyFormal := *registered.Parameters[0]
	detached.Parameters[0] = &copyFormal
	ctx.localScope = &detached
	if plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s); plan != nil {
		t.Fatalf("copied formal borrowed real method-header role: %s", plan.message)
	}
}
func TestHelpfulNPECounterTargetFormalTypePoison(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target{void call(String value){}}class Main{static void run(Target lane){lane.call(null);}}`, "Main", "run", "call")
	resolved := helpfulNPEResolution(t, ctx, n, s)
	if helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s) == nil {
		t.Fatal("actual source String control missing")
	}
	formal := resolved.def.Parameters[0]
	formal.OriginalType = "java.lang.Object"
	if plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s); plan != nil {
		t.Fatalf("metadata overwrote actual source formal String: %s", plan.message)
	}
}
func TestHelpfulNPECounterClassFormalBoundPoison(t *testing.T) {
	ctx, n, s := helpfulNPEContext(t, `package proof;class Target<T extends String>{void call(T value){}}class Main{static void run(Target<String> lane){lane.call(null);}}`, "Main", "run", "call")
	resolved := helpfulNPEResolution(t, ctx, n, s)
	if helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s) == nil {
		t.Fatal("actual bounded source control missing")
	}
	if len(resolved.owner.TypeParameters) != 1 || len(resolved.owner.TypeParameters[0].Bounds) != 1 {
		t.Fatal("bound mutation premise")
	}
	resolved.owner.TypeParameters[0].Bounds[0].Original = "java.lang.Object"
	if plan := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s); plan != nil {
		t.Fatalf("mutable metadata overwrote actual class header bound String: %s", plan.message)
	}
}
