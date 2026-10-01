package transpiler

import "testing"

// A known Java common base remains callable when the contributing nominal
// declarations differ. A caller binder cannot replace the selected base.
func TestRelatedNominalConditionalReceiverKeepsCommonBase(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"plain-related-nominal-conditional", `class Base {int marker(){return 1;}} class Child extends Base {} class Owner {int run(boolean flag, Base base, Child child){return (flag ? base : child).marker();}}`},
		{"shadowed-related-nominal-conditional", `class Base {int marker(){return 1;}} class Child extends Base {} class Other {int marker(){return 2;}} class Owner {Base base; Child child; <Base extends Other> int run(boolean flag){return (flag ? base : child).marker();}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope("Owner")
			expected := helper.File.Symbols.FindClassScope("Base")
			if ctx.currentClass == nil || expected == nil {
				t.Fatal("missing nominal consumer/common-base declarations")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 || methods[0].DeclarationNode == nil {
				t.Fatal("missing consumer method declaration")
			}
			ctx.localScope = methods[0]
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("missing marker invocation")
			}
			receiver := invocation.ChildByFieldName("object")
			if receiver == nil {
				t.Fatal("marker invocation must have explicit conditional receiver")
			}
			javaType, known := inferExprJavaType(receiver, ctx, helper.File.Source)
			if !known || javaType != "Base" {
				t.Fatalf("existing conditional Java type = %q/%t, want Base", javaType, known)
			}
			target := resolveInvocationTarget(receiver, ctx, helper.File.Source)
			if target == nil || target.classScope != expected {
				t.Fatalf("related conditional receiver %q selected %#v; want exact nominal Base declaration", receiver.Content(helper.File.Source), target)
			}
		})
	}
}
