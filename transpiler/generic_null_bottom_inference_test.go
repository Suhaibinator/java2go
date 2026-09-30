package transpiler

import "testing"

func TestGenericNullBottomInferenceControls(t *testing.T) {
	for _, tc := range []struct{ name, declaration, call, target, want string }{
		{"declared-string-bound", "<T extends java.lang.String> T identity(T value)", "identity(null)", "", "java.lang.String"},
		{"object-erasure", "<T> T identity(T value)", "identity(null)", "", "Object"},
		{"nonnull-witness", "<T> T identity(T value,T witness)", `identity(null,"witness")`, "", "java.lang.String"},
		{"assignment-target", "<T> T identity(T value)", "identity(null)", "java.lang.Integer", "java.lang.Integer"},
		{"explicit-argument", "<T> T identity(T value)", "NullInference.<java.lang.String>identity(null)", "", "java.lang.String"},
		{"primitive-boxing", "<T> T identity(T value)", "identity(17)", "", "java.lang.Integer"},
		{"array-witness", "<T> T identity(T[] value)", "identity(new java.lang.String[]{null})", "", "java.lang.String"},
		{"array-null", "<T> T identity(T[] value)", "identity(null)", "", "Object"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, "class NullInference {static "+tc.declaration+"{return null;} static Object run(){return "+tc.call+";}}")
			methods := helper.Ctx.currentClass.FindMethod().ByName("identity")
			if len(methods) != 1 {
				t.Fatal("missing identity declaration")
			}
			invocation := findNode(helper.File.Ast, "method_invocation")
			if invocation == nil {
				t.Fatal("missing invocation")
			}
			ctx := helper.Ctx.Clone()
			ctx.expectedType = tc.target
			ctx.expectedTypeRoot = invocation
			got := resolvedMethodInvocationTypeBindings(methods[0], invocation, ctx, helper.File.Source)["T"]
			if got != tc.want {
				t.Fatalf("inferred T=%q, want %q", got, tc.want)
			}
		})
	}
}
