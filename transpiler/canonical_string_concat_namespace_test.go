package transpiler

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestCanonicalStringConcatDeclarationNamespace(t *testing.T) {
	for _, tc := range []struct {
		name, imports, members, binder, expression string
		shadowsString                            bool
	}{
		{"ordinary", "", "", "", `"value:" + number`, false},
		{"source-shadow", "", "static class String {}", "", `"value:" + number`, true},
		{"foreign-import", "import foreign.String;", "", "", `"value:" + number`, true},
		{"binder-shadow", "", "", "<String>", `"value:" + number`, true},
		{"nested-source-shadow", "", "static class String {}", "", `("value:" + number) + number`, true},
	} {
		source := tc.imports + " public class TextNamespace { " + tc.members +
			" static " + tc.binder + " java.lang.String run(int number) { return " + tc.expression + "; } }"
		helper := setupParseHelper(t, source)
		methods := helper.Ctx.currentClass.FindMethod().ByName("run")
		if len(methods) != 1 {
			t.Fatalf("%s: missing caller declaration", tc.name)
		}
		helper.Ctx.localScope = methods[0]
		expression := findNode(helper.File.Ast, "binary_expression")
		literal := findNode(helper.File.Ast, "string_literal")
		if expression == nil || literal == nil {
			t.Fatalf("%s: missing concatenation or literal", tc.name)
		}
		for _, value := range []*sitter.Node{literal, expression} {
			got, known := inferExprJavaType(value, helper.Ctx, helper.File.Source)
			if !known || got != "java.lang.String" {
				t.Fatalf("%s: inferred %s type %q/%t, want canonical JDK declaration", tc.name, value.Type(), got, known)
			}
		}
		if tc.shadowsString && isBuiltinJavaString("String", helper.Ctx) {
			t.Fatalf("%s: source/import/binder spelling was captured as JDK String", tc.name)
		}
		owner, known := intrinsicReceiverTypeName(expression, helper.Ctx, helper.File.Source)
		if !known || owner != "String" {
			t.Fatalf("%s: concatenation owner = %q/%t, want canonical String", tc.name, owner, known)
		}
	}
}
