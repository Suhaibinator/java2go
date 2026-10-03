package transpiler

import (
	"testing"
)

func TestGenericInvocationReceiverDeclaration72(t *testing.T) {
	for _, tc := range []struct{ name, imports, types, field, binder, call, result, intrinsic string }{
		{"duration-collision", "import java.time.Duration;", "", "Duration", "Duration", "identity(value)", "java.time.Duration", "Duration"},
		{"duration-renamed-control", "import java.time.Duration;", "", "Duration", "Value", "identity(value)", "java.time.Duration", "Duration"},
		{"duration-explicit", "import java.time.Duration;", "", "Duration", "Duration", "ReceiverResult72.<Duration>identity(value)", "java.time.Duration", "Duration"},
		{"list-collision", "import java.util.ArrayList;", "", "ArrayList<String>", "ArrayList", "identity(value)", "java.util.ArrayList<java.lang.String>", "ArrayList"},
		{"source-shadow", "", "static class Duration { long getSeconds(){return 7;} }", "Duration", "Duration", "identity(value)", "ReceiverResult72.Duration", ""},
		{"qualified-JDK-source-shadow", "", "static class Duration { long getSeconds(){return 7;} }", "java.time.Duration", "Duration", "identity(value)", "java.time.Duration", "Duration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			method := "getSeconds"
			if tc.intrinsic == "ArrayList" {
				method = "size"
			}
			source := tc.imports + " public class ReceiverResult72 {" + tc.types + " static " + tc.field + " value; static <" + tc.binder + "> " + tc.binder + " identity(" + tc.binder + " input){return input;} static long run(){return " + tc.call + "." + method + "();}}"
			helper := setupParseHelper(t, source)
			outer := findNode(helper.File.Ast, "method_invocation")
			if outer == nil {
				t.Fatal("missing outer invocation")
			}
			inner := outer.ChildByFieldName("object")
			if inner == nil || inner.Type() != "method_invocation" {
				t.Fatalf("expected chained generic invocation, got %v", inner)
			}
			got, known := inferExprJavaType(inner, helper.Ctx, helper.File.Source)
			if !known || got != tc.result {
				t.Fatalf("generic invocation type = %q/%t, want declaration %q", got, known, tc.result)
			}
			intrinsic, present := intrinsicReceiverTypeName(inner, helper.Ctx, helper.File.Source)
			if tc.intrinsic == "" {
				if present {
					t.Fatalf("source result captured as runtime intrinsic %q", intrinsic)
				}
			} else if !present || intrinsic != tc.intrinsic {
				t.Fatalf("receiver intrinsic = %q/%t, want %q", intrinsic, present, tc.intrinsic)
			}
		})
	}
}

func TestGenericInvocationResultRetainsCallerBinder72(t *testing.T) {
	source := `public class CallerResult72 {
		static <Value> Value identity(Value input) { return input; }
		static <Value> Value run(Value value) { return identity(value); }
	}`
	helper := setupParseHelper(t, source)
	methods := helper.Ctx.currentClass.FindMethod().ByName("run")
	if len(methods) != 1 {
		t.Fatal("missing caller method")
	}
	helper.Ctx.localScope = methods[0]
	invocation := findNode(helper.File.Ast, "method_invocation")
	got, known := inferExprJavaType(invocation, helper.Ctx, helper.File.Source)
	want := methods[0].TypeParameters[0].EmittedName()
	if !known || got != want {
		t.Fatalf("inferred caller binder = %q/%t, want %q", got, known, want)
	}
	if intrinsic, known := intrinsicReceiverTypeName(invocation, helper.Ctx, helper.File.Source); !known || intrinsic != "Object" {
		t.Fatalf("caller binder erasure = %q/%t, want Object", intrinsic, known)
	}
}

func TestGenericInvocationResultUnresolvedBinder72(t *testing.T) {
	helper := setupParseHelper(t, `public class UnresolvedResult72 {
		static <Value> Value empty() { return null; }
		static Object run() { return empty(); }
	}`)
	invocation := findNode(helper.File.Ast, "method_invocation")
	if got, known := inferUserMethodReturnType(invocation, helper.Ctx, helper.File.Source); known {
		t.Fatalf("unresolved declaration binder leaked as %q", got)
	}
}
