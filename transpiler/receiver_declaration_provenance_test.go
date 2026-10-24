package transpiler

import "testing"

// Source-only proposal. Exact owner assertions distinguish nominal declarations
// from genuine binders even when both expose the same method name and result.
func TestReceiverDeclarationProvenanceUnderCallerBinder(t *testing.T) {
	for _, tc := range []struct{ name, source, current, expected string }{
		{"owner-instance-field", `class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing> int run(){return held.marker();}}`, "Owner", "Thing"},
		{"owner-this-field", `class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing> int run(){return this.held.marker();}}`, "Owner", "Thing"},
		{"owner-other-value-field", `class Thing {int marker(){return 1;}} class Holder {Thing held;} class Owner {<Thing> int run(Holder holder){return holder.held.marker();}}`, "Owner", "Thing"},
		{"owner-static-qualified-field", `class Owner {static class Thing {int marker(){return 1;}} static Thing held; static <Thing> int run(){return Owner.held.marker();}}`, "Owner", "Thing"},
		{"enclosing-instance-field", `class Outer {static class Thing {int marker(){return 1;}} Thing held; class Inner {<Thing> int run(){return held.marker();}}}`, "Inner", "Thing"},
		{"inherited-instance-field", `class Base {static class Thing {int marker(){return 1;}} Thing held;} class Child extends Base {<Thing> int run(){return held.marker();}}`, "Child", "Thing"},
		{"array-element-field", `class Owner {static class Thing {int marker(){return 1;}} Thing[] held; <Thing> int run(){return held[0].marker();}}`, "Owner", "Thing"},
		{"parenthesized-field", `class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing> int run(){return (held).marker();}}`, "Owner", "Thing"},
		{"field-wrong-bound-selection", `class Other {int marker(){return 2;}} class Owner {static class Thing {int marker(){return 1;}} Thing held; <Thing extends Other> int run(){return held.marker();}}`, "Owner", "Thing"},
		{"default-package-method-return", `class Thing {int marker(){return 1;}} class Owner {Thing acquire(){return null;} <Thing> int run(){return acquire().marker();}}`, "Owner", "Thing"},
		{"default-package-inherited-return", `class Thing {int marker(){return 1;}} class Base {Thing acquire(){return null;}} class Child extends Base {<Thing> int run(){return acquire().marker();}}`, "Child", "Thing"},
		{"nested-method-return-control", `class Owner {static class Thing {int marker(){return 1;}} Thing acquire(){return null;} <Thing> int run(){return acquire().marker();}}`, "Owner", "Thing"},
		{"package-method-return-control", `package provenance; class Thing {int marker(){return 1;}} class Owner {Thing acquire(){return null;} <Thing> int run(){return acquire().marker();}}`, "Owner", "Thing"},
		{"renamed-method-binder-control", `class Owner {static class Thing {int marker(){return 1;}} Thing held; <Unrelated> int run(){return held.marker();}}`, "Owner", "Thing"},
		{"actual-method-binder-control", `class Thing {int marker(){return 1;}} class Bound {int marker(){return 2;}} class Owner {<Thing extends Bound> int run(Thing held){return held.marker();}}`, "Owner", "Bound"},
		{"outer-class-binder-control", `class Bound {int marker(){return 2;}} class Other {int marker(){return 3;}} class Owner<Thing extends Bound> {Thing held; <Thing extends Other> int run(){return held.marker();}}`, "Owner", "Bound"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope(tc.current)
			if ctx.currentClass == nil {
				t.Fatal("missing receiver consumer class declaration")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 || methods[0].DeclarationNode == nil {
				t.Fatal("missing receiver consumer method declaration")
			}
			ctx.localScope = methods[0]
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("missing marker invocation")
			}
			receiver := invocation.ChildByFieldName("object")
			if receiver == nil {
				t.Fatal("marker invocation must have an explicit receiver expression")
			}
			target := resolveInvocationTarget(receiver, ctx, helper.File.Source)
			expected := helper.File.Symbols.FindClassScope(tc.expected)
			if expected == nil || target == nil || target.classScope != expected {
				typeName, known := inferExprJavaType(receiver, ctx, helper.File.Source)
				t.Fatalf("receiver %q inferred %q/%t selected %#v; want exact %s declaration", receiver.Content(helper.File.Source), typeName, known, target, tc.expected)
			}
		})
	}
}
