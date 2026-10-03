package transpiler

import "testing"

func TestClassLiteralBuiltinOwnerIdentity63(t *testing.T) {
	for _, q := range []struct{ name, literal, parameter string }{
		{"qualified", "java.lang.Integer.class", "java.lang.Integer"},
		{"primitive", "int.class", "int"},
		{"source", "Parent.class", "Parent"},
	} {
		t.Run(q.name, func(t *testing.T) {
			source := `public class LiteralOwner63 {
    static class Class { int isInstance(Object x) { return 7; } }
    static class Parent {}
    static boolean run(Object x) { return ` + q.literal + `.isInstance(x); }
   }`
			helper := setupParseHelper(t, source)
			literal := findNode(helper.File.Ast, "class_literal")
			if literal == nil {
				t.Fatal("missing class literal")
			}
			got, ok := inferExprJavaType(literal, helper.Ctx, helper.File.Source)
			want := "java.lang.Class<" + q.parameter + ">"
			if !ok || got != want {
				t.Fatalf("class literal owner = %q/%t, want %q", got, ok, want)
			}
			if receiver, known := intrinsicReceiverTypeName(literal, helper.Ctx, helper.File.Source); !known || receiver != "Class" {
				t.Fatalf("genuine Class literal dispatch = %q/%t", receiver, known)
			}
		})
	}
	t.Run("source-instance-method", func(t *testing.T) {
		const source = `public class SourceOwner63 {
   static class Class { int isInstance(Object x) { return 7; } }
   static int run(Object x) { return new Class().isInstance(x); }
  }`
		helper := setupParseHelper(t, source)
		object := findNode(helper.File.Ast, "object_creation_expression")
		if object == nil {
			t.Fatal("missing source Class creation")
		}
		if receiver, known := intrinsicReceiverTypeName(object, helper.Ctx, helper.File.Source); known {
			t.Fatalf("source Class instance captured as intrinsic %q", receiver)
		}
	})
}
