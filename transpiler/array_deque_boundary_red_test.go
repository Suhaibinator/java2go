package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestArrayDequeBoundaryCompiler224(t *testing.T) {
	t.Run("typed_array_overload", func(t *testing.T) {
		source := `import java.util.ArrayDeque;class Probe {
public static String tagString(){return "STRING";}public static String tagObject(){return "OBJECT";}
public static String pick(String[] value){return tagString();}public static String pick(Object[] value){return tagObject();}
public static String result(ArrayDeque<String> d){return pick(d.toArray(new String[0]));}}`
		generated := renderGoFileFromJava(t, source)
		file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
		if err != nil {
			t.Fatalf("fixture did not produce Go syntax: %v", err)
		}
		functions := map[string]*ast.FuncDecl{}
		entry := ""
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				functions[fn.Name.Name] = fn
				if fn.Recv == nil && strings.HasSuffix(strings.ToLower(fn.Name.Name), "resultjava2goexecution") {
					if entry != "" {
						t.Fatal("ambiguous result fixture")
					}
					entry = fn.Name.Name
				}
			}
		}
		if entry == "" {
			t.Fatal("missing result fixture")
		}
		seen := map[string]bool{}
		tags := map[string]bool{}
		var visit func(string)
		visit = func(name string) {
			if seen[name] {
				return
			}
			seen[name] = true
			lower := strings.ToLower(name)
			if strings.Contains(lower, "tagstring") {
				tags["STRING"] = true
			}
			if strings.Contains(lower, "tagobject") {
				tags["OBJECT"] = true
			}
			if fn := functions[name]; fn != nil {
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if ident, ok := call.Fun.(*ast.Ident); ok {
							visit(ident.Name)
						}
					}
					return true
				})
			}
		}
		visit(entry)
		if tags["OBJECT"] && !tags["STRING"] {
			t.Fatal("typed toArray overload selected Object[] instead of String[]")
		}
		if !tags["STRING"] || tags["OBJECT"] {
			t.Fatalf("unexpected overload fixture route: %v", tags)
		}
	})
	t.Run("generator_overload_refusal", func(t *testing.T) {
		strictRoutingState(t)
		var observed any
		func() {
			defer func() { observed = recover() }()
			renderGoFileFromJava(t, `import java.util.ArrayDeque;import java.util.function.IntFunction;class Probe {String[] result(ArrayDeque<String> d,IntFunction<String[]> generator){return d.toArray(generator);}}`)
		}()
		if observed == nil && len(Diagnostics()) == 0 {
			t.Fatal("IntFunction toArray overload mislowered as array")
		}
		matched := false
		for _, diagnostic := range Diagnostics() {
			text := diagnostic.String()
			if strings.Contains(text, "toArray") && strings.Contains(text, "IntFunction") {
				matched = true
			}
		}
		if observed == nil || !matched {
			t.Fatal("unexpected generator refusal fixture outcome")
		}
	})
}
