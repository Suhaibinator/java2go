package transpiler

import "testing"

// The lambda-shape table is keyed by (receiver type, method) rather than by
// method name alone. These tests pin the behavior that key buys: two types can
// register the same method name with different element types without colliding.
func TestLambdaShape_OptionalAndStreamMapDoNotCollide(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
public class ShapeProgram {
    public static void run() {
        Optional<String> name = Optional.of("hi");
        Optional<Integer> length = name.map(s -> s.length());

        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().map(n -> n * 2).forEach(n -> System.out.println(n));
    }
}
`
	out := renderGoFileFromJava(t, src)
	// Optional<String>.map takes a string parameter...
	assertContains(t, out, "stdjava.OptionalMap(name, func(s string)")
	// ...while Stream<Integer>.map takes an int32 one.
	assertContains(t, out, "stdjava.StreamMap(")
	assertContains(t, out, "func(n int32) int32")
}

// A block-bodied mapper used to defeat result inference entirely, because the
// old inference bailed on any `block` body. A block whose only statement is a
// return now infers like the expression form.
func TestLambdaShape_BlockBodiedMapperInfersResultType(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class BlockMapperProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().map(n -> {
            return "value" + n;
        }).forEach(s -> System.out.println(s));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "func(n int32) string")
}

// The expression form must keep working exactly as before.
func TestLambdaShape_ExpressionMapperInfersResultType(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class ExprMapperProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().map(n -> "value" + n).forEach(s -> System.out.println(s));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "func(n int32) string")
}

// A multi-statement block has no single expression to infer from, so the mapper
// falls back to the element type rather than guessing.
func TestLambdaShape_MultiStatementBlockFallsBackToElementType(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class MultiStatementProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().map(n -> {
            int doubled = n * 2;
            return doubled;
        }).forEach(n -> System.out.println(n));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "func(n int32) int32")
}

// reduce takes a BinaryOperator<T>, whose result is pinned to the element type
// regardless of what the body would infer to.
func TestLambdaShape_ReducePinsResultToElementType(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
public class ReduceProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        int total = xs.stream().reduce(0, (a, b) -> a + b);
        System.out.println(total);
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "func(a int32, b int32) int32")
}

func TestLambdaParameterNames(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lambda string
		want   []string
	}{
		{name: "bare identifier", lambda: "n -> n + 1", want: []string{"n"}},
		{name: "parenthesized single", lambda: "(n) -> n + 1", want: []string{"n"}},
		{name: "inferred pair", lambda: "(a, b) -> a + b", want: []string{"a", "b"}},
		{name: "formal parameters", lambda: "(Integer a, Integer b) -> a + b", want: []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("class C { void m() { f(" + tc.lambda + "); } }")
			helper := setupParseHelper(t, string(source))
			lambda := findNode(helper.File.Ast, "lambda_expression")
			if lambda == nil {
				t.Fatal("no lambda_expression in the parsed source")
			}
			got := lambdaParameterNames(lambda.ChildByFieldName("parameters"), source)
			if len(got) != len(tc.want) {
				t.Fatalf("lambdaParameterNames = %q, want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("lambdaParameterNames = %q, want %q", got, tc.want)
				}
			}
		})
	}
}
