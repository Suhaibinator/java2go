package transpiler

import "testing"

// Joined receiver types must keep their declaration origin even when a caller
// introduces an unrelated type parameter with the same spelling.
func TestReceiverJoinDeclarationProvenance(t *testing.T) {
	for _, tc := range []struct{ name, source, current, expected string }{
		{"nominal-field-ternary", `class Other {int marker(){return 2;}} class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing extends Other> int run(boolean flag){return (flag ? held : held).marker();}}`, "Owner", "Thing"},
		{"nominal-field-pair-ternary", `class Other {int marker(){return 2;}} class Owner {static class Thing {int marker(){return 1;}} Thing left; Thing right; <Thing extends Other> int run(boolean flag){return (flag ? left : right).marker();}}`, "Owner", "Thing"},
		{"nominal-qualified-static-ternary", `class Other {int marker(){return 2;}} class Owner {static class Thing {int marker(){return 1;}} static Thing held; static <Thing extends Other> int run(boolean flag){return (flag ? Owner.held : Owner.held).marker();}}`, "Owner", "Thing"},
		{"nominal-inherited-field-ternary", `class Other {int marker(){return 2;}} class Base {static class Thing {int marker(){return 1;}} Thing held;} class Child extends Base {<Thing extends Other> int run(boolean flag){return (flag ? held : held).marker();}}`, "Child", "Thing"},
		{"nominal-return-ternary", `class Thing {int marker(){return 1;}} class Other {int marker(){return 2;}} class Owner {Thing acquire(){return null;} <Thing extends Other> int run(boolean flag){return (flag ? acquire() : acquire()).marker();}}`, "Owner", "Thing"},
		{"nominal-field-switch", `class Other {int marker(){return 2;}} class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing extends Other> int run(int flag){return (switch(flag){case 0 -> held; default -> held;}).marker();}}`, "Owner", "Thing"},
		{"method-binder-ternary-control", `class Thing {int marker(){return 1;}} class Bound {int marker(){return 2;}} class Owner {<Thing extends Bound> int run(boolean flag, Thing held){return (flag ? held : held).marker();}}`, "Owner", "Bound"},
		{"method-binder-switch-control", `class Thing {int marker(){return 1;}} class Bound {int marker(){return 2;}} class Owner {<Thing extends Bound> int run(int flag, Thing held){return (switch(flag){case 0 -> held; default -> held;}).marker();}}`, "Owner", "Bound"},
		{"outer-binder-ternary-control", `class Bound {int marker(){return 2;}} class Other {int marker(){return 3;}} class Owner<Thing extends Bound> {Thing held; <Thing extends Other> int run(boolean flag){return (flag ? held : held).marker();}}`, "Owner", "Bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope(tc.current)
			if ctx.currentClass == nil {
				t.Fatal("missing receiver consumer class")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 || methods[0].DeclarationNode == nil {
				t.Fatal("missing receiver consumer method")
			}
			ctx.localScope = methods[0]
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("missing marker invocation")
			}
			receiver := invocation.ChildByFieldName("object")
			if receiver == nil {
				t.Fatal("marker invocation must have explicit receiver")
			}
			target := resolveInvocationTarget(receiver, ctx, helper.File.Source)
			expected := helper.File.Symbols.FindClassScope(tc.expected)
			if expected == nil || target == nil || target.classScope != expected {
				typeName, known := inferExprJavaType(receiver, ctx, helper.File.Source)
				t.Fatalf("joined receiver %q inferred %q/%t selected %#v; want exact %s declaration", receiver.Content(helper.File.Source), typeName, known, target, tc.expected)
			}
		})
	}
}
