package transpiler

import "testing"

// Reflection reads the same wildcard capture as collection and source-method
// results. Whitespace in a Java wildcard must not survive as a result type.
func TestReflectionConstructorWildcardReadType(t *testing.T) {
	for _, tc := range []struct {
		name, argument, want string
	}{
		{"upper-bound", "? extends Example", "Example"},
		{"spaced-upper-bound", "?  extends Example", "Example"},
		{"multiline-upper-bound", "?\n extends Example", "Example"},
		{"lower-bound", "? super Example", "java.lang.Object"},
		{"spaced-lower-bound", "?  super Example", "java.lang.Object"},
		{"unbounded", "?", "java.lang.Object"},
		{"concrete-source-Object", "Object", "Object"},
		{"upper-bound-source-Object", "?  extends Object", "Object"},
		{"concrete-source-superExample", "superExample", "superExample"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "import java.lang.reflect.Constructor; public class Probe { static class Example {} static class Object {} static class superExample {} Constructor<" + tc.argument + "> value; java.lang.Object read() throws java.lang.Exception {return value.newInstance();}}"
			helper := setupParseHelper(t, source)
			if helper.File.Ast.HasError() {
				t.Fatal("invalid Java wildcard fixture")
			}
			invocation := findNode(helper.File.Ast, "method_invocation")
			if invocation == nil {
				t.Fatal("missing constructor invocation")
			}
			got, known := reflectionDeclaredResultType(invocation, helper.Ctx, helper.File.Source)
			if !known || got != tc.want {
				t.Fatalf("reflection wildcard read = %q/%t, want %q", got, known, tc.want)
			}
		})
	}
}
