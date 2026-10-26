package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func volatileFamilyRootExpr(t *testing.T, expr ast.Expr) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// These controls describe declaration and native API contracts; the unchanged
// seven generated application packages remain the type-check/execution gate.
func TestVolatileGeneratedFamilyRootsTDD(t *testing.T) {
	reflection := []struct{ name, source string }{
		{"qualified", `public class LongRouteProbe { static long run(java.lang.reflect.Field f,Object o){f.setLong(o,11L);return f.getLong(o);} }`},
		{"imported", `import java.lang.reflect.Field; public class LongRouteProbe { static long run(Field f,Object o){f.setLong(o,11L);return f.getLong(o);} }`},
		{"local", `public class LongRouteProbe {static long run(java.lang.reflect.Field f,Object o){class Worker {long read(){f.setLong(o,11L);return f.getLong(o);}}return new Worker().read();}}`},
		{"enum", `public class LongRouteProbe {enum Worker {ONE;long read(java.lang.reflect.Field f,Object o){f.setLong(o,11L);return f.getLong(o);}} static long run(java.lang.reflect.Field f,Object o){return Worker.ONE.read(f,o);}}`},
		{"anonymous", `public class LongRouteProbe {interface Reader {long read();}static long run(java.lang.reflect.Field f,Object o){return new Reader(){public long read(){f.setLong(o,11L);return f.getLong(o);}}.read();}}`},
	}
	for _, c := range reflection {
		t.Run("reflection_"+c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, c.source)
			calls := volatileTDDCalls(file)
			if calls["GetLongExecution"] != 1 || calls["SetLongExecution"] != 1 {
				t.Fatalf("canonical typed Field routes lost: %v\n%s", calls, generated)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := volatileTDDCallName(call)
				if name != "GetLongExecution" && name != "SetLongExecution" {
					return true
				}
				want := 2
				if name == "SetLongExecution" {
					want = 3
				}
				if len(call.Args) != want || volatileFamilyRootExpr(t, call.Args[0]) == "nil" {
					t.Fatal("typed Field route lost caller Execution")
				}
				return true
			})
		})
	}
	guards := []struct{ name, source string }{
		{"source_field", `public class LongOwnerProbe {static class Field {public long getLong(Object o){return 13L;}public void setLong(Object o,long n){}}static long run(){Field f=new Field();f.setLong(null,13L);return f.getLong(null);}}`},
		{"lexical_field", `public class LongOwnerProbe {static class Member {public long getLong(Object o){return 13L;}public void setLong(Object o,long n){}}static class Box<Field extends Member>{Field f;long read(){f.setLong(null,13L);return f.getLong(null);}}}`},
	}
	for _, c := range guards {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, c.source)
			calls := volatileTDDCalls(file)
			if calls["GetLongExecution"] != 0 || calls["SetLongExecution"] != 0 || calls["GetLongJava2goExecution"] == 0 || calls["SetLongJava2goExecution"] == 0 {
				t.Fatalf("source/binder Field lost declaration dispatch: %v\n%s", calls, generated)
			}
		})
	}
	storage := []struct{ name, source string }{
		{"local_storage", `public class CellLayoutProbe {static long run(){class Worker {volatile long epoch=3L;volatile int count=2;long ordinary=4L;long read(){epoch=5L;return epoch+count+ordinary;}}return new Worker().read();}}`},
		{"local_hierarchy", `public class CellLayoutProbe {static long run(){class Parent {volatile long epoch=3L;long ordinary=4L;}class Worker extends Parent {long read(){epoch=5L;return epoch+ordinary;}}return new Worker().read();}}`},
	}
	for _, c := range storage {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, c.source)
			seen := map[string]string{}
			ast.Inspect(file, func(n ast.Node) bool {
				field, ok := n.(*ast.Field)
				if !ok {
					return true
				}
				for _, name := range field.Names {
					if name.Name == "epoch" || name.Name == "count" || name.Name == "ordinary" {
						seen[name.Name] = volatileFamilyRootExpr(t, field.Type)
					}
				}
				return true
			})
			if seen["epoch"] != "stdjava.VolatileFieldCell" || seen["ordinary"] != "int64" || (c.name == "local_storage" && seen["count"] != "stdjava.VolatileFieldCell") {
				t.Fatalf("declaration backing layout %v\n%s", seen, generated)
			}
		})
	}
	generic := []struct{ name, source, descriptor string }{
		{"owner_unbounded", `public class ErasureProbe<T>{volatile T value;T read(){return value;}}`, "stdjava.ObjectTypeID"},
		{"owner_bound", `public class ErasureProbe<T extends ErasureProbe.Base>{static class Base {} volatile T value;T read(){return value;}}`, `stdjava.TypeID("ErasureProbe$Base")`},
		{"consuming_method_shadow", `public class ErasureProbe<T extends ErasureProbe.Base>{static class Base {} volatile T value;static <T> Object read(ErasureProbe<Base> owner){return owner.value;}}`, `stdjava.TypeID("ErasureProbe$Base")`},
		{"dependent_bound", `public class ErasureProbe<U extends ErasureProbe.Base,T extends U>{static class Base {} volatile T value;T read(){return value;}}`, `stdjava.TypeID("ErasureProbe$Base")`},
		{"method_local_bound", `public class ErasureProbe {static class Base {} static <T extends Base> Object read(T initial){class Worker {volatile T value;Worker(){value=initial;}T read(){return value;}}return new Worker().read();}}`, `stdjava.TypeID("ErasureProbe$Base")`},
		{"method_anonymous_bound", `public class ErasureProbe {static class Base {} interface Reader {Object read();} static <T extends Base> Object read(T initial){return new Reader(){volatile T value=initial;public Object read(){return value;}}.read();}}`, `stdjava.TypeID("ErasureProbe$Base")`},
		{"array_bound", `public class ErasureProbe<T extends ErasureProbe.Base>{static class Base {} volatile T[] value;T[] read(){return value;}}`, `stdjava.ArrayTypeID(stdjava.TypeID("ErasureProbe$Base"))`},
	}
	for _, c := range generic {
		t.Run("generic_"+c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, c.source)
			loads := 0
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || volatileTDDCallName(call) != "VolatileLoad" {
					return true
				}
				loads++
				if len(call.Args) != 2 || volatileFamilyRootExpr(t, call.Args[1]) != c.descriptor {
					t.Fatalf("load needs declaring bound descriptor %s\n%s", c.descriptor, generated)
				}
				return true
			})
			if loads == 0 || strings.Contains(generated, `stdjava.TypeID("T")`) {
				t.Fatalf("missing load or un-erased binder\n%s", generated)
			}
		})
	}
}
