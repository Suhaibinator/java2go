package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func TestArrayDequeCanonicalCompiler224(t *testing.T) {
	for _, entry := range []struct {
		name, source string
		want         []string
	}{
		{"constructors_and_protocols", `import java.util.ArrayDeque;import java.util.Deque;import java.util.Queue;
class Probe {String run(){ArrayDeque<String> d=new ArrayDeque<String>(-17);Deque<String> q=d;Queue<String> queue=d;q.addFirst("first");queue.offer("last");return q.removeLast();}}`, []string{"NewArrayDequeWithCapacity[", "DequeAddFirstExecution", "QueueOfferExecution", "DequeRemoveLastExecution", "ObjectView[*stdjava.JavaString]"}},
		{"constructor_references", `import java.util.ArrayDeque;import java.util.function.Supplier;
interface Make {ArrayDeque<String> create();} class Probe {Supplier<ArrayDeque<String>> nativeFactory(){return ArrayDeque::new;}Make sourceFactory(){return ArrayDeque::new;}}`, []string{"NewArrayDeque[", "Supplier", "Make"}},
		{"wildcard_collection_factory", `import java.util.ArrayDeque;import java.util.Collection;interface Make {Collection<?> create();}class Probe {Make factory(){return ArrayDeque::new;}}`, []string{"NewArrayDeque[", "stdjava.JavaIterable"}},
		{"raw_concrete_views", `import java.util.ArrayDeque;class Probe {ArrayDeque raw(Object value){return (ArrayDeque)value;}ArrayDeque<String> typed(Object value){return (ArrayDeque<String>)value;}}`, []string{"ReferenceCheckCast[*stdjava.ArrayDeque]"}},
		{"qualified_owner_under_binder", `class Probe<ArrayDeque>{java.util.ArrayDeque<String> value=new java.util.ArrayDeque<String>();}`, []string{"*stdjava.ArrayDeque", "NewArrayDeque["}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, entry.source)
			if entry.name == "constructors_and_protocols" || entry.name == "raw_concrete_views" {
				file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
				if err != nil {
					t.Fatal(err)
				}
				if entry.name == "constructors_and_protocols" {
					assertArrayDequeInferredAliases224(t, file)
				} else {
					assertArrayDequeNominalCasts224(t, file)
				}
			}
			for _, text := range entry.want {
				if !strings.Contains(generated, text) {
					t.Fatalf("missing canonical lowering %s: %s", text, generated)
				}
			}
		})
	}
	for _, entry := range []struct{ name, source string }{
		{"source_owner", `class ArrayDeque {ArrayDeque(){}}class Probe {ArrayDeque value=new ArrayDeque();}`},
		{"type_binder", `class Probe<ArrayDeque,Deque,Queue>{ArrayDeque a;Deque d;Queue q;}`},
		{"foreign_import", `import foreign.ArrayDeque;import foreign.Deque;import foreign.Queue;class Probe{ArrayDeque a;Deque d;Queue q;}`},
		{"source_interfaces", `interface Deque {}interface Queue {}class Probe{Deque d;Queue q;}`},
	} {
		t.Run(entry.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, entry.source)
			for _, text := range []string{"stdjava.ArrayDeque", "stdjava.JavaDeque", "stdjava.JavaQueue", "stdjava.NewArrayDeque"} {
				if strings.Contains(generated, text) {
					t.Fatalf("source/binder/import borrowed native %s: %s", text, generated)
				}
			}
		})
	}
}

// Go may infer local types; verify the constructor/alias data flow itself.
func assertArrayDequeInferredAliases224(t *testing.T, file *ast.File) {
	t.Helper()
	bindings := map[string][]ast.Expr{}
	ast.Inspect(file, func(node ast.Node) bool {
		if assignment, ok := node.(*ast.AssignStmt); ok && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
			if name, ok := assignment.Lhs[0].(*ast.Ident); ok {
				bindings[name.Name] = append(bindings[name.Name], assignment.Rhs[0])
			}
		}
		return true
	})
	for _, name := range []string{"d", "q", "queue"} {
		if len(bindings[name]) != 1 {
			t.Fatalf("expected one local binding for %s, got %d", name, len(bindings[name]))
		}
	}
	call, ok := bindings["d"][0].(*ast.CallExpr)
	if !ok {
		t.Fatal("deque allocation is not a call")
	}
	factory, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		t.Fatal("deque allocation lost logical E")
	}
	if !arrayDequeStdjavaSelector224(factory.X, "NewArrayDequeWithCapacity") {
		t.Fatal("deque local does not infer canonical erased factory carrier")
	}
	element, ok := factory.Index.(*ast.StarExpr)
	if !ok || !arrayDequeStdjavaSelector224(element.X, "JavaString") {
		t.Fatal("deque factory lost logical Java String element")
	}
	for _, name := range []string{"q", "queue"} {
		alias, ok := bindings[name][0].(*ast.Ident)
		if !ok || alias.Name != "d" {
			t.Fatalf("%s does not preserve original deque carrier identity", name)
		}
	}
}
func arrayDequeStdjavaSelector224(expr ast.Expr, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "stdjava"
}
func arrayDequeErasedCarrier224(expr ast.Expr) bool {
	pointer, ok := expr.(*ast.StarExpr)
	return ok && arrayDequeStdjavaSelector224(pointer.X, "ArrayDeque")
}
func assertArrayDequeNominalCasts224(t *testing.T, file *ast.File) {
	t.Helper()
	casts, returnViews := 0, 0
	ast.Inspect(file, func(node ast.Node) bool {
		if function, ok := node.(*ast.FuncDecl); ok && function.Type.Results != nil {
			for _, result := range function.Type.Results.List {
				if arrayDequeErasedCarrier224(result.Type) {
					returnViews++
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		generic, ok := call.Fun.(*ast.IndexExpr)
		if !ok || !arrayDequeStdjavaSelector224(generic.X, "ReferenceCheckCast") {
			return true
		}
		if !arrayDequeErasedCarrier224(generic.Index) || len(call.Args) != 2 {
			t.Fatal("Java cast lost its erased physical carrier")
		}
		descriptor, ok := call.Args[1].(*ast.CallExpr)
		if !ok || !arrayDequeStdjavaSelector224(descriptor.Fun, "TypeID") || len(descriptor.Args) != 1 {
			t.Fatal("Java cast lost its nominal descriptor")
		}
		literal, ok := descriptor.Args[0].(*ast.BasicLit)
		if !ok {
			t.Fatal("Java cast descriptor is not canonical")
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil || value != "java.util.ArrayDeque" {
			t.Fatal("Java cast acquired a foreign nominal owner")
		}
		casts++
		return true
	})
	// Public and caller-execution entry points retain the same erased carrier
	// for both raw and parameterized Java views. The explicit Java cast must use
	// ReferenceCheckCast's nominal/unknown-value ClassCastException shield.
	if casts != 2 || returnViews != 4 {
		t.Fatalf("raw/typed erased Java cast contract: casts=%d return views=%d", casts, returnViews)
	}
}
