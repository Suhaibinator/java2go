package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func renderGoFileFromJava(t *testing.T, src string) string {
	t.Helper()
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	return buf.String()
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestGenericsIntegration_GenericClassAndNestedTypes(t *testing.T) {
	src := `
package gen.integration;
public class Pair<K extends Number, V> {
    K key;
    V value;
    public Pair(K k, V v) {
        this.key = k;
        this.value = v;
    }
    public K getKey() { return this.key; }
    public V getValue() { return this.value; }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "type Pair[K *Number, V any] struct") {
		t.Errorf("Expected generic struct with 2 type params, got:\n%s", out)
	}
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "key K") || !strings.Contains(flat, "value V") {
		t.Errorf("Expected fields to use type params K/V, got:\n%s", out)
	}
	if !strings.Contains(out, "func NewPair[K *Number, V any]") {
		t.Errorf("Expected generic constructor function with type params, got:\n%s", out)
	}
	if !strings.Contains(out, "func (pr *Pair[K, V]) GetKey()") {
		t.Errorf("Expected method receiver to use instantiated type params, got:\n%s", out)
	}
}

func TestGenericsIntegration_NestedGenericTypeExpressions(t *testing.T) {
	src := `
package gen.integration2;
import java.util.List;
import java.util.Map;
public class Container {
    Map<String, List<Integer>> m;
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "m *Map[string, *List[*Integer]]") {
		t.Errorf("Expected nested generic field type '*Map[string, *List[*Integer]]', got:\n%s", out)
	}
}

func TestGenericsIntegration_DiamondExplicitAndRawConstructors(t *testing.T) {
	src := `
package gen.integration3;
public class Box<T> {
    T value;
    public Box() {}
    public static void test() {
        Box<String> inferred = new Box<>();
        Box<Integer> explicit = new Box<Integer>();
        Box raw = new Box();
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "NewBox[string]") {
		t.Errorf("Expected diamond operator to infer 'string' type arg, got:\n%s", out)
	}
	if !strings.Contains(out, "NewBox[*Integer]") && !strings.Contains(out, "NewBox[Integer]") {
		t.Errorf("Expected explicit type args on constructor call, got:\n%s", out)
	}
	if strings.Contains(out, "raw := NewBox[") || strings.Contains(out, "raw = NewBox[") {
		t.Errorf("Expected raw 'new Box()' to omit type args, got:\n%s", out)
	}
}

func TestGenericsIntegration_InstanceGenericMethodHelper_EndToEnd(t *testing.T) {
	src := `
package gen.integration4;
public class Box<T> {
    public <R> R identity(R value) { return value; }

    public static Foo callFoo(Box<Foo> box, Foo value) {
        return box.identity(value);
    }

    public static <X> X callGeneric(Box<X> box, X value) {
        return box.identity(value);
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "type BoxIdentityHelper") || !strings.Contains(out, "func NewBoxIdentityHelper") {
		t.Errorf("Expected helper type + constructor for instance generic method, got:\n%s", out)
	}
	if !strings.Contains(out, "NewBoxIdentityHelper[*Foo, *Foo]") && !strings.Contains(out, "NewBoxIdentityHelper[*Foo,*Foo]") {
		t.Errorf("Expected helper call for concrete Foo to use pointer type args, got:\n%s", out)
	}
	if !strings.Contains(out, "NewBoxIdentityHelper[X, X]") && !strings.Contains(out, "NewBoxIdentityHelper[X,X]") {
		t.Errorf("Expected helper call for generic X to use type param args, got:\n%s", out)
	}
}

func TestGenericsIntegration_ExplicitTypeArgumentsOnGenericFunctionCall(t *testing.T) {
	src := `
package gen.integration5;
public class Utils {
    static <T> T id(T value) { return value; }

    public static void test() {
        Foo f = null;
        Foo g = Utils.<Foo>id(f);
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "func id[T any]") {
		t.Errorf("Expected generic function declaration for id, got:\n%s", out)
	}
	if !strings.Contains(out, "id[*Foo]") && !strings.Contains(out, "id[Foo]") {
		t.Errorf("Expected explicit type args to be applied at call site, got:\n%s", out)
	}
}

func TestGenericsIntegration_NestedClassTypeParameters(t *testing.T) {
	src := `
package gen.integration6;
public class Outer<T> {
    public class Inner<U> {
        T t;
        U u;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "type Outer[T any] struct") {
		t.Errorf("Expected Outer to be generic, got:\n%s", out)
	}
	if !strings.Contains(out, "type OuterInner[T any, U any] struct") {
		t.Errorf("Expected Inner to inherit parent type params and add its own, got:\n%s", out)
	}
}
