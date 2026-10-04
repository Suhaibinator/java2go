package transpiler

import "testing"

func TestDeclaredStreamShellSourceProducerControls(t *testing.T) {
	for _, tc := range []struct{ name, decl, receiver, call string }{
		{"source-list-stream", `static class Stream<T>{} static class List<T>{Stream<T> stream(){return null;}}`, "List<java.lang.String>", "stream()"},
		{"source-list-parallel-stream", `static class Stream<T>{} static class List<T>{Stream<T> parallelStream(){return null;}}`, "List<java.lang.String>", "parallelStream()"},
		{"source-optional-stream", `static class Stream<T>{} static class Optional<T>{Stream<T> stream(){return null;}}`, "Optional<java.lang.String>", "stream()"},
		{"source-stream-map", `static class Stream<T>{Stream<java.lang.String> map(java.util.function.Function<T,java.lang.String> mapper){return null;}}`, "Stream<java.lang.String>", `map(item -> "mapped")`},
		{"source-stream-map-to-object", `static class Stream<T>{Stream<java.lang.String> mapToObj(java.util.function.Function<T,java.lang.String> mapper){return null;}}`, "Stream<java.lang.String>", `mapToObj(item -> "mapped")`},
		{"source-int-stream-boxed", `static class Stream<T>{} static class IntStream{Stream<java.lang.Integer> boxed(){return null;}}`, "IntStream", "boxed()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, "class SourceResultProbe {"+tc.decl+" static "+tc.receiver+" value; static Object run(){return value."+tc.call+";}}")
			invocation := findNode(helper.File.Ast, "method_invocation")
			if result, ok := inferIntrinsicMethodResultType(invocation, helper.Ctx, helper.File.Source); ok {
				t.Fatalf("source producer captured intrinsic %q", result)
			}
			result, known := inferExprJavaType(invocation, helper.Ctx, helper.File.Source)
			owner, _ := parseJavaTypeString(result)
			expected := helper.File.Symbols.FindClassScope("Stream")
			if expected == nil {
				t.Fatal("missing declared source result scope")
			}
			if !known || resolveClassScopeByQualifiedName(helper.Ctx, owner) != expected {
				t.Fatalf("source producer result = %q/%t, want declared source Stream identity", result, known)
			}
		})
	}
}

func TestNominalThisSuperReceiverIgnoresMethodBinderSpelling(t *testing.T) {
	for _, tc := range []struct{ name, source, current, expected string }{
		{"this", `class Thing {int count(){return 1;} <Thing> int run(){return this.count();}}`, "Thing", "Thing"},
		{"super", `class Base {int count(){return 1;}} class Child extends Base {<Base> int run(){return super.count();}}`, "Child", "Base"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope(tc.current)
			if ctx.currentClass == nil {
				t.Fatal("missing nominal current class")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("missing method binder declaration")
			}
			ctx.localScope = methods[0]
			invocation := findNode(helper.File.Ast, "method_invocation")
			target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, helper.File.Source)
			expected := helper.File.Symbols.FindClassScope(tc.expected)
			if expected == nil || target == nil || target.classScope != expected {
				t.Fatalf("nominal receiver target = %#v, want %s declaration", target, tc.expected)
			}
			if result, known := inferExprJavaType(invocation, ctx, helper.File.Source); !known || result != "int" {
				t.Fatalf("nominal method result = %q/%t, want int", result, known)
			}
		})
	}
}
