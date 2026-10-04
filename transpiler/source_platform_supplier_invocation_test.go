package transpiler

import (
	"strings"
	"testing"
)

func TestSourcePlatformSupplierStaticInvocationSelection(t *testing.T) {
	source := `public class Invocation {
  public static <T> T supplied(T value, java.util.function.Supplier<String> message) { return value; }
  static class Source implements java.util.function.Supplier<String> { public String get() { return "message"; } }
  static class Base<E> implements java.util.function.Supplier<E> { public E get() { return null; } }
  static class Child extends Base<String> {}
  static String concrete() { return Invocation.supplied("kept", new Source()); }
  static String inherited() { return Invocation.supplied("kept", new Child()); }
 }`
	h := setupParseHelper(t, source)
	for _, name := range []string{"concrete", "inherited"} {
		t.Run(name, func(t *testing.T) {
			ctx := h.Ctx.Clone()
			for _, method := range ctx.currentClass.Methods {
				if method.OriginalName == name {
					ctx.localScope = method
				}
			}
			ctx.executionContextName = executionParameterName(ctx.localScope.DeclarationNode, h.File.Source, ctx)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			selected := findBestMethodInHierarchy(ctx.currentClass, "supplied", invocation.ChildByFieldName("arguments"), false, true, ctx, h.File.Source)
			if selected == nil || selected.def == nil {
				t.Fatal("declared Supplier<String> ancestry must keep the generic static declaration applicable")
			}
			if selected.def.Name != "Supplied" || !selected.def.IsStatic {
				t.Fatalf("lost the exact public static definition: %#v", selected.def)
			}
			if got := executionMethodCallName(selected.def, selected.owner, ctx); !strings.HasSuffix(got, "SuppliedJava2goExecution") {
				t.Fatalf("lost the caller execution entry: %s", got)
			}
		})
	}
}

func TestSourcePlatformSupplierInvocationRejectsUnprovenTypes(t *testing.T) {
	source := `public class InvocationNegative {
  public static <T> T supplied(T value, java.util.function.Supplier<String> message) { return value; }
  static class Wrong implements java.util.function.Supplier<Integer> { public Integer get() { return null; } }
  static class Shape { public String get() { return "shape"; } }
  static class Supplier<E> { public E get() { return null; } }
  static String wrong() { return InvocationNegative.supplied("kept", new Wrong()); }
  static String shape() { return InvocationNegative.supplied("kept", new Shape()); }
  static String shadow() { return InvocationNegative.supplied("kept", new Supplier<String>()); }
 }`
	h := setupParseHelper(t, source)
	for _, name := range []string{"wrong", "shape", "shadow"} {
		t.Run(name, func(t *testing.T) {
			ctx := h.Ctx.Clone()
			for _, method := range ctx.currentClass.Methods {
				if method.OriginalName == name {
					ctx.localScope = method
				}
			}
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if selected := findBestMethodInHierarchy(ctx.currentClass, "supplied", invocation.ChildByFieldName("arguments"), false, true, ctx, h.File.Source); selected != nil {
				t.Fatal("incompatible invariant arguments or source shape must not certify platform Supplier<String>")
			}
		})
	}
}
