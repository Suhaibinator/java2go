package transpiler

import (
	"go/ast"
	"strings"
	"testing"
)

func anonymousVolatileSourceMethod(t *testing.T, file *ast.File, suffix, generated string) *ast.FuncDecl {
	t.Helper()
	var result *ast.FuncDecl
	for _, decl := range file.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if ok && strings.HasSuffix(method.Name.Name, suffix+"Java2goExecution") {
			if result != nil {
				t.Fatalf("ambiguous original method %s", suffix)
			}
			result = method
		}
	}
	if result == nil {
		t.Fatalf("missing original method %s\n%s", suffix, generated)
	}
	return result
}

func TestAnonymousVolatileFinalDeclarationTDD(t *testing.T) {
	cases := []struct {
		name, base, field string
		loads             int
		typ               string
	}{
		{"new_volatile", "", "volatile int flag=7;", 1, "int32"},
		{"ordinary_base_volatile_child", "int flag=40;", "volatile int flag=7;", 1, "int32"},
		{"volatile_base_ordinary_child", "volatile int flag=40;", "int flag=7;", 0, ""},
		{"different_volatile_types", "volatile long flag=40L;", "volatile int flag=7;", 1, "int32"},
		{"both_ordinary", "int flag=40;", "int flag=7;", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, "public class AnonymousFinalProbe {static class Base {"+c.base+"} public static int read(){return (new Base(){"+c.field+"}).flag;} }")
			method := anonymousVolatileSourceMethod(t, file, "Read", generated)
			calls := volatileTDDCalls(method.Body)
			if calls["VolatileLoad"] != c.loads {
				t.Fatalf("final anonymous declaration selects wrong storage kind: %v\n%s", calls, generated)
			}
			if c.loads != 0 {
				ast.Inspect(method.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || volatileTDDCallName(call) != "VolatileLoad" {
						return true
					}
					indexed, ok := call.Fun.(*ast.IndexExpr)
					if !ok {
						t.Fatal("volatile primitive load lacks exact storage type")
						return false
					}
					typ, ok := indexed.Index.(*ast.Ident)
					if !ok || typ.Name != c.typ {
						t.Fatalf("load used inherited type instead of anonymous field %s", c.typ)
					}
					return true
				})
			}
		})
	}
}

func TestAnonymousVolatileDirectOperationsTDD(t *testing.T) {
	cases := []struct {
		name, body    string
		loads, stores int
		base, field   string
	}{
		{"write", "return (new Base(){volatile int flag;}).flag=7;", 0, 1, "", "volatile int flag;"},
		{"compound", "return (new Base(){volatile int flag;}).flag+=7;", 1, 1, "", "volatile int flag;"},
		{"postfix", "return (new Base(){volatile int flag;}).flag++;", 1, 1, "", "volatile int flag;"},
		{"prefix", "return ++(new Base(){volatile int flag;}).flag;", 1, 1, "", "volatile int flag;"},
		{"statement_write", "(new Base(){volatile int flag;}).flag=7;return 1;", 0, 1, "", "volatile int flag;"},
		{"statement_update", "(new Base(){volatile int flag;}).flag++;return 1;", 1, 1, "", "volatile int flag;"},
		{"ordinary_base_volatile_child_write", "return (new Base(){FIELD}).flag=7;", 0, 1, "int flag;", "volatile int flag;"},
		{"ordinary_base_volatile_child_compound", "return (new Base(){FIELD}).flag+=7;", 1, 1, "int flag;", "volatile int flag;"},
		{"ordinary_base_volatile_child_postfix", "return (new Base(){FIELD}).flag++;", 1, 1, "int flag;", "volatile int flag;"},
		{"volatile_base_ordinary_child_write", "return (new Base(){FIELD}).flag=7;", 0, 0, "volatile long flag;", "int flag;"},
		{"volatile_base_ordinary_child_compound", "return (new Base(){FIELD}).flag+=7;", 0, 0, "volatile long flag;", "int flag;"},
		{"volatile_base_ordinary_child_postfix", "return (new Base(){FIELD}).flag++;", 0, 0, "volatile long flag;", "int flag;"},
		{"different_volatile_types_write", "return (new Base(){FIELD}).flag=7;", 0, 1, "volatile long flag;", "volatile int flag;"},
		{"different_volatile_types_compound", "return (new Base(){FIELD}).flag+=7;", 1, 1, "volatile long flag;", "volatile int flag;"},
		{"different_volatile_types_postfix", "return (new Base(){FIELD}).flag++;", 1, 1, "volatile long flag;", "volatile int flag;"},
		{"both_ordinary_write", "return (new Base(){FIELD}).flag=7;", 0, 0, "int flag;", "int flag;"},
		{"both_ordinary_compound", "return (new Base(){FIELD}).flag+=7;", 0, 0, "int flag;", "int flag;"},
		{"both_ordinary_postfix", "return (new Base(){FIELD}).flag++;", 0, 0, "int flag;", "int flag;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, "public class AnonymousOperationProbe {static class Base {"+c.base+"} public static int run(){"+strings.ReplaceAll(c.body, "FIELD", c.field)+"}}")
			method := anonymousVolatileSourceMethod(t, file, "Run", generated)
			calls := volatileTDDCalls(method.Body)
			if calls["VolatileLoad"] != c.loads || calls["VolatileStore"] != c.stores {
				t.Fatalf("anonymous operation requires separate exact accesses load=%d store=%d, got %v\n%s", c.loads, c.stores, calls, generated)
			}
			ast.Inspect(method.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := volatileTDDCallName(call)
				if name != "VolatileLoad" && name != "VolatileStore" {
					return true
				}
				indexed, ok := call.Fun.(*ast.IndexExpr)
				if !ok {
					t.Fatal("volatile operation lacks declared primitive storage type")
					return false
				}
				typ, ok := indexed.Index.(*ast.Ident)
				if !ok || typ.Name != "int32" {
					t.Fatalf("%s used inherited width instead of anonymous int field", name)
				}
				return true
			})
			for _, name := range []string{"CompareAndSet", "Swap", "GetAndAdd"} {
				if calls[name] != 0 {
					t.Fatalf("anonymous %s acquired atomic compound operation %s", c.name, name)
				}
			}
		})
	}
}

func TestAnonymousExplicitBaseCastTDD(t *testing.T) {
	cases := []struct {
		name, base, child, result string
		loads                     int
		typ                       string
	}{
		{"base_volatile_child_ordinary", "volatile long flag=40L;", "int flag=7;", "long", 1, "int64"},
		{"base_ordinary_child_volatile", "int flag=40;", "volatile int flag=7;", "int", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, "public class AnonymousCastProbe {static class Base {"+c.base+"} public static "+c.result+" read(){return ((Base)(new Base(){"+c.child+"})).flag;}}")
			method := anonymousVolatileSourceMethod(t, file, "Read", generated)
			if got := volatileTDDCalls(method.Body)["VolatileLoad"]; got != c.loads {
				t.Fatalf("explicit Base cast must select Base declaration: got %d\n%s", got, generated)
			}
			ast.Inspect(method.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || volatileTDDCallName(call) != "VolatileLoad" {
					return true
				}
				indexed, ok := call.Fun.(*ast.IndexExpr)
				if !ok {
					t.Fatal("cast volatile load lacks physical type")
					return false
				}
				typ, ok := indexed.Index.(*ast.Ident)
				if !ok || typ.Name != c.typ {
					t.Fatalf("explicit Base cast used anonymous width instead of %s", c.typ)
				}
				return true
			})
		})
	}
}
