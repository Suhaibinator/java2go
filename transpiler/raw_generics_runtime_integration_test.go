package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestRawGenericStaticParameterAndInheritedInnerTypeRuntime(t *testing.T) {
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previousGlobal })

	src := `
public class RawGenericProgram<T> {
    private Node head;

    public class Node {
        T value;
        Node next;
        Node(T value) { this.value = value; }
    }

    RawGenericProgram(T value) {
        this.head = new Node(value);
    }

    T get() {
        if (this.head != null) {
            return this.head.value;
        }
        return null;
    }

    String render() {
        String result = "value=";
        result += this.get();
        return result;
    }

    static void assertValue(RawGenericProgram list, int expected) {
        if ((Integer) list.get() != expected) {
            throw new AssertionError("wrong value");
        }
    }

    public static String run() {
        RawGenericProgram<Integer> list = new RawGenericProgram<>(7);
        assertValue(list, 7);
        return list.render();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func TestRawGenericRuntime(t *testing.T) {
    if got := Run(); got == nil || !got.Equals(stdjava.JavaStringFromHostUTF8("value=7")) {
        t.Fatalf("Run() = %v, want value=7", got)
    }
}
`)

	// Run the unchanged Java program's generated Go before checking its ABI.
	// Canonical erased storage replaced the older per-T Go representation.
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	shape := func(node ast.Node) string {
		var buffer bytes.Buffer
		if err := format.Node(&buffer, fileSet, node); err != nil {
			t.Fatal(err)
		}
		return buffer.String()
	}
	types := map[string]*ast.TypeSpec{}
	functions := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					types[typ.Name.Name] = typ
				}
			}
		case *ast.FuncDecl:
			functions[declaration.Name.Name] = declaration
		}
	}
	for _, name := range []string{"RawGenericProgram", "RawGenericProgramNode"} {
		typ := types[name]
		if typ == nil || !typ.Assign.IsValid() || typ.TypeParams == nil || len(typ.TypeParams.List) != 1 || shape(typ.Type) != name+"Java2goErased" {
			t.Fatalf("%s must retain its source generic alias over shared erased storage", name)
		}
	}
	for _, expected := range []struct{ owner, field, typ string }{
		{"RawGenericProgramJava2goErased", "head", "*RawGenericProgramNodeJava2goErased"},
		{"RawGenericProgramNodeJava2goErased", "next", "*RawGenericProgramNodeJava2goErased"},
		{"RawGenericProgramNodeJava2goErased", "rawGenericProgram", "*RawGenericProgramJava2goErased"},
		{"RawGenericProgramNodeJava2goErased", "value", "any"},
	} {
		owner := types[expected.owner]
		if owner == nil {
			t.Fatalf("missing erased owner %s", expected.owner)
		}
		structure, ok := owner.Type.(*ast.StructType)
		if !ok {
			t.Fatalf("%s must own its fields", expected.owner)
		}
		found := false
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				if name.Name == expected.field {
					found = shape(field.Type) == expected.typ
				}
			}
		}
		if !found {
			t.Fatalf("%s.%s must retain %s", expected.owner, expected.field, expected.typ)
		}
	}
	get := functions["getJava2goExecution"]
	assertValue := functions["assertValueJava2goExecution"]
	render := functions["renderJava2goExecution"]
	if get == nil || get.Type.Results == nil || len(get.Type.Results.List) != 1 || shape(get.Type.Results.List[0].Type) != "any" ||
		assertValue == nil || assertValue.Type.TypeParams != nil || len(assertValue.Type.Params.List) != 3 || shape(assertValue.Type.Params.List[1].Type) != "*RawGenericProgram[any]" || render == nil {
		t.Fatal("get and raw static parameter must retain the erased reference ABI")
	}
	nilResult, integerCast, concat := false, false, false
	ast.Inspect(get.Body, func(node ast.Node) bool {
		if returned, ok := node.(*ast.ReturnStmt); ok && len(returned.Results) == 1 && shape(returned.Results[0]) == "*new(any)" {
			nilResult = true
		}
		return true
	})
	ast.Inspect(assertValue.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && shape(call) == `stdjava.UnboxInteger(stdjava.ObjectView[*stdjava.Integer](list.getJava2goExecution(__java2goExecution), stdjava.TypeID("java.lang.Integer")))` {
			integerCast = true
		}
		return true
	})
	ast.Inspect(render.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && shape(call) == `stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, old), stdjava.JavaStringTextOperandExecution(__java2goExecution, rhs))` {
			concat = true
		}
		return true
	})
	if !nilResult || !integerCast || !concat {
		t.Fatalf("nil reference, owned Integer checkcast/unbox and execution-aware concat must remain: nil=%v cast=%v concat=%v", nilResult, integerCast, concat)
	}
}
