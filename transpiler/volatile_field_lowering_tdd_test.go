package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func volatileTDDCallName(call *ast.CallExpr) string {
	fun := call.Fun
	switch indexed := fun.(type) {
	case *ast.IndexExpr:
		fun = indexed.X
	case *ast.IndexListExpr:
		fun = indexed.X
	}
	if selector, ok := fun.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	if identifier, ok := fun.(*ast.Ident); ok {
		return identifier.Name
	}
	return ""
}

func volatileTDDCalls(node ast.Node) map[string]int {
	calls := map[string]int{}
	ast.Inspect(node, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			calls[volatileTDDCallName(call)]++
		}
		return true
	})
	return calls
}

func volatileTDDParse(t *testing.T, source string) (*ast.File, string) {
	t.Helper()
	generated := renderGoFileFromJava(t, source)
	file, err := parser.ParseFile(token.NewFileSet(), "volatile_generated.go", generated, 0)
	if err != nil {
		t.Fatalf("invalid generated volatile syntax: %v\n%s", err, generated)
	}
	return file, generated
}

// This is a structural RED prerequisite, not a visibility oracle. The root's
// sealed lane must additionally type-check the generated program and execute
// publication, aliasing and ordering witnesses against the pinned JDK.
func TestVolatileFieldStorageAndAccessTDD(t *testing.T) {
	file, generated := volatileTDDParse(t, `public class VolatileStorageProbe {
  public volatile int count;
  public volatile long epoch;
  public volatile Object value;
  public static volatile long global;
  public int read(){return count;}
  public long readLong(){return epoch;}
  public Object readRef(){return value;}
  public void write(int n,long l,Object o){count=n;epoch=l;value=o;global=l;}
 }`)
	cells := 0
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok {
			return true
		}
		typ, ok := field.Type.(*ast.SelectorExpr)
		if ok && typ.Sel.Name == "VolatileFieldCell" {
			cells++
		}
		return true
	})
	// The static backing variable is a ValueSpec, not a struct Field.
	globals := 0
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		typ, ok := value.Type.(*ast.SelectorExpr)
		if ok && typ.Sel.Name == "VolatileFieldCell" {
			globals++
		}
		return true
	})
	if cells != 3 || globals != 1 {
		t.Fatalf("volatile fields need one canonical cell each: instance=%d static=%d\n%s", cells, globals, generated)
	}
	for _, name := range []string{"ReadJava2goExecution", "ReadLongJava2goExecution", "ReadRefJava2goExecution"} {
		found := false
		for _, declaration := range file.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if !ok || method.Name.Name != name {
				continue
			}
			found = true
			if got := volatileTDDCalls(method.Body)["VolatileLoad"]; got != 1 {
				t.Fatalf("%s needs one volatile load, got %d\n%s", name, got, generated)
			}
		}
		if !found {
			t.Fatalf("missing source method %s\n%s", name, generated)
		}
	}
	if strings.Contains(generated, "visibility/ordering not enforced") {
		t.Fatal("volatile declaration still merely documents missing semantics")
	}
	if got := volatileTDDCalls(file)["ReflectGeneratedVolatileFieldCellExecution"]; got != 3 {
		t.Fatalf("instance metadata must resolve each same cell, got %d\n%s", got, generated)
	}
}

func TestVolatileIncrementIsSeparateLoadAndStoreTDD(t *testing.T) {
	file, generated := volatileTDDParse(t, `public class VolatileCompoundProbe {
  public volatile int count;
  public int post(){return count++;}
  public int pre(){return ++count;}
  public int compound(){return count+=(count=9);}
  public void statement(){count++;count+=2;count=3;}
 }`)
	for _, name := range []string{"PostJava2goExecution", "PreJava2goExecution", "CompoundJava2goExecution"} {
		found := false
		for _, declaration := range file.Decls {
			method, ok := declaration.(*ast.FuncDecl)
			if !ok || method.Name.Name != name {
				continue
			}
			found = true
			calls := volatileTDDCalls(method.Body)
			stores := 1
			if strings.HasPrefix(name, "Compound") {
				stores = 2
			}
			if calls["VolatileLoad"] != 1 || calls["VolatileStore"] != stores {
				t.Fatalf("%s must save one read then write, calls=%v\n%s", name, calls, generated)
			}
			for _, atomic := range []string{"CompareAndSet", "GetAndAdd", "Swap", "PostIncrement", "PreIncrement"} {
				if calls[atomic] != 0 {
					t.Fatalf("%s acquired unintended atomic/pointer operation %s", name, atomic)
				}
			}
		}
		if !found {
			t.Fatalf("missing method %s\n%s", name, generated)
		}
	}
}

func TestVolatileInheritedAndStaticCellTDD(t *testing.T) {
	_, generated := volatileTDDParse(t, `public class VolatileInheritanceProbe {
  public static class Base {protected volatile long stamp;public static volatile Object shared;public long read(){return stamp;}}
  public static class Child extends Base {public long alter(){return ++stamp;}public Object sharedValue(){return Child.shared;}}
  public static long run(){Child c=new Child();Base b=c;b.stamp=17L;return c.alter()+b.read();}
 }`)
	// One declaration each. A subclass name must not allocate a second static
	// storage variable, and a projected Base view must not copy its instance cell.
	if got := strings.Count(normalizeSpaces(generated), "Stamp stdjava.VolatileFieldCell"); got != 1 {
		t.Fatalf("inherited declaration cell count %d\n%s", got, generated)
	}
	if !strings.Contains(generated, "VolatileLoad") || !strings.Contains(generated, "VolatileStore") {
		t.Fatalf("inherited/static accesses bypass volatile lowering\n%s", generated)
	}
	if strings.Contains(generated, "ChildShared stdjava.VolatileFieldCell") {
		t.Fatal("subclass allocated inherited static storage")
	}
}

func TestVolatileDeclaredFamiliesTDD(t *testing.T) {
	cases := []struct{ name, source string }{
		{"enum", `public class VolatileEnumProbe { enum Choice { ONE; volatile long stamp=7L; long read(){return stamp;} } public static long run(){Choice.ONE.stamp=9L;return Choice.ONE.read();} }`},
		{"local", `public class VolatileLocalProbe { public static long run(){class Local {volatile long stamp=7L;long read(){return stamp;}} Local local=new Local();local.stamp=9L;return local.read();} }`},
		{"anonymous", `public class VolatileAnonymousProbe {interface Carrier {long read();} public static long run(){Carrier value=new Carrier(){volatile long stamp=7L;public long read(){stamp=9L;return stamp;}};return value.read();} }`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetDiagnostics()
			setStrictMode(true)
			defer setStrictMode(false)
			file, generated := volatileTDDParse(t, c.source)
			if len(Diagnostics()) != 0 {
				t.Fatalf("volatile family retains unsupported diagnostics: %v", Diagnostics())
			}
			calls := volatileTDDCalls(file)
			if calls["VolatileLoad"] == 0 || calls["VolatileStore"] == 0 || calls["ReflectGeneratedVolatileFieldCellExecution"] == 0 {
				t.Fatalf("%s needs same-cell load/store/metadata: %v\n%s", c.name, calls, generated)
			}
			if !strings.Contains(normalizeSpaces(generated), "stdjava.VolatileFieldCell") {
				t.Fatalf("%s retains raw backing storage\n%s", c.name, generated)
			}
		})
	}
}

func TestThreadYieldNamespaceTDD(t *testing.T) {
	cases := []struct {
		name, source string
		calls        int
	}{
		{"canonical", `public class StandardYieldProbe {public static void run(){Thread.yield();java.lang.Thread.yield();}}`, 2},
		{"source_thread", `public class SourceYieldProbe {static class Thread {public static void yield(){}} public static void run(){Thread.yield();}}`, 0},
		{"source_instance", `public class InstanceYieldProbe {static class Worker {public void yield(){}} public static void run(){Worker Thread=new Worker();Thread.yield();}}`, 0},
		{"source_overload", `public class OverloadYieldProbe {static class Worker extends java.lang.Thread {public void yield(int value){}} public static void run(){new Worker().yield(3);}}`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file, generated := volatileTDDParse(t, c.source)
			calls := volatileTDDCalls(file)
			if calls["ThreadYieldExecution"] != c.calls {
				t.Fatalf("namespace %s routes wrong yield owner: %v\n%s", c.name, calls, generated)
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || volatileTDDCallName(call) != "ThreadYieldExecution" {
					return true
				}
				if len(call.Args) != 1 {
					t.Fatal("yield lost caller execution")
				}
				if identifier, ok := call.Args[0].(*ast.Ident); ok && identifier.Name == "nil" {
					t.Fatal("yield fabricated no caller execution")
				}
				return true
			})
		})
	}
}
