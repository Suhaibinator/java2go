package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func updaterTDDParse(t *testing.T, source string) (*ast.File, string) {
	t.Helper()
	generated := renderGoFileFromJava(t, source)
	file, err := parser.ParseFile(token.NewFileSet(), "updater_generated.go", generated, 0)
	if err != nil {
		t.Fatalf("invalid generated syntax: %v\n%s", err, generated)
	}
	return file, generated
}
func updaterTDDCallName(c *ast.CallExpr) string {
	fun := c.Fun
	switch x := fun.(type) {
	case *ast.IndexExpr:
		fun = x.X
	case *ast.IndexListExpr:
		fun = x.X
	}
	switch x := fun.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	}
	return ""
}
func updaterTDDCalls(node ast.Node) map[string]int {
	result := map[string]int{}
	ast.Inspect(node, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			result[updaterTDDCallName(c)]++
		}
		return true
	})
	return result
}

// Resolve the original Java declaration through the production symbol authority,
// then require its exact collision-safe execution companion in the emitted AST.
// No capitalization, prefix, or fallback-name matching is accepted here.
func updaterTDDStaticExecutionName(t *testing.T, file *ast.File, ownerName, originalName string) string {
	t.Helper()
	pkg := symbol.GlobalScope.FindPackage("")
	if pkg == nil {
		t.Fatal("missing selected source package")
	}
	owner := pkg.FindClassScope(ownerName)
	if owner == nil || owner.Class == nil || owner.Class.OriginalName != ownerName {
		t.Fatalf("missing exact source owner %s", ownerName)
	}
	var selected *symbol.Definition
	for _, method := range owner.Methods {
		if method.OriginalName == originalName && method.IsStatic && method.DeclarationNode != nil && len(method.Parameters) == 0 {
			if selected != nil {
				t.Fatalf("ambiguous static source declaration %s.%s", ownerName, originalName)
			}
			selected = method
		}
	}
	if selected == nil {
		t.Fatalf("missing zero-arity static source declaration %s.%s", ownerName, originalName)
	}
	var sourceFile *symbol.FileScope
	for _, candidate := range pkg.Files {
		if candidate != nil && candidate.FindClassScope(ownerName) == owner {
			sourceFile = candidate
		}
	}
	if sourceFile == nil {
		t.Fatal("selected source file missing")
	}
	name := symbol.GoIdentifier(executionImplementationName(selected, owner, Ctx{currentFile: sourceFile, currentClass: owner}))
	matches := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name {
			continue
		}
		if fn.Recv != nil || fn.Type.Params.NumFields() != 1 {
			t.Fatalf("selected static companion has wrong boundary: %s", name)
		}
		executionType, pointer := fn.Type.Params.List[0].Type.(*ast.StarExpr)
		if !pointer {
			t.Fatalf("selected static companion lacks caller Execution: %s", name)
		}
		selector, qualified := executionType.X.(*ast.SelectorExpr)
		if !qualified || selector.Sel.Name != "Execution" {
			t.Fatalf("selected static companion lacks caller Execution: %s", name)
		}
		matches++
	}
	if matches != 1 {
		t.Fatalf("selected source companion declaration count=%d: %s", matches, name)
	}
	return name
}

func TestAtomicUpdaterFactoryAndNativeFacadeTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.*;
 public class AtomicFactoryProbe {public volatile int count;public volatile long stamp;public volatile Object value;
 static AtomicIntegerFieldUpdater<AtomicFactoryProbe> i=AtomicIntegerFieldUpdater.newUpdater(AtomicFactoryProbe.class,"count");
 static AtomicLongFieldUpdater<AtomicFactoryProbe> l=AtomicLongFieldUpdater.newUpdater(AtomicFactoryProbe.class,"stamp");
 static AtomicReferenceFieldUpdater<AtomicFactoryProbe,Object> r=AtomicReferenceFieldUpdater.newUpdater(AtomicFactoryProbe.class,Object.class,"value");
 }`)
	calls := updaterTDDCalls(file)
	for _, name := range []string{"NewAtomicIntegerFieldUpdaterJavaString", "NewAtomicLongFieldUpdaterJavaString", "NewAtomicReferenceFieldUpdaterJavaString"} {
		if calls[name] != 1 {
			t.Fatalf("factory %s count=%d\n%s", name, calls[name], generated)
		}
	}
	if strings.Contains(generated, "stdjava.AtomicIntegerFieldUpdater[") || strings.Contains(generated, "stdjava.AtomicReferenceFieldUpdater[") {
		t.Fatalf("native physical facade incorrectly instantiated\n%s", generated)
	}
	if !strings.Contains(generated, `"AtomicFactoryProbe"`) {
		t.Fatalf("factory lost lexical caller TypeID\n%s", generated)
	}
}
func TestAtomicUpdaterExecutionMethodsTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.*;
 public class AtomicMethodsProbe {volatile int count;
 int use(AtomicIntegerFieldUpdater<AtomicMethodsProbe> u){u.set(this,3);u.lazySet(this,4);u.compareAndSet(this,4,5);u.weakCompareAndSet(this,5,6);u.getAndSet(this,7);u.getAndAdd(this,2);u.addAndGet(this,2);u.getAndIncrement(this);u.incrementAndGet(this);u.getAndDecrement(this);u.decrementAndGet(this);return u.get(this);}}
 `)
	calls := updaterTDDCalls(file)
	for _, name := range []string{"Set", "LazySet", "CompareAndSet", "WeakCompareAndSet", "GetAndSet", "GetAndAdd", "AddAndGet", "GetAndIncrement", "IncrementAndGet", "GetAndDecrement", "DecrementAndGet", "Get"} {
		if calls[name+"Execution"] != 1 {
			t.Fatalf("%s must dispatch once with Execution\n%s", name, generated)
		}
	}
	if calls["ReferenceRequireNonNull"] < 12 {
		t.Fatalf("nullable updater receiver boundary missing\n%s", generated)
	}
}
func TestAtomicUpdaterTypedReferenceResultTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.AtomicReferenceFieldUpdater;
 public class AtomicReferenceProbe {volatile String value;
 String read(AtomicReferenceFieldUpdater<AtomicReferenceProbe,String> u){return u.get(this);}
 String swap(AtomicReferenceFieldUpdater<AtomicReferenceProbe,String> u,String v){return u.getAndSet(this,v);}}`)
	if updaterTDDCalls(file)["ObjectView"] < 2 {
		t.Fatalf("erased reference result lost nominal typed/null projection\n%s", generated)
	}
	if strings.Contains(generated, "return u.Get(") {
		t.Fatalf("execution context lost\n%s", generated)
	}
}
func TestAtomicUpdaterSourceShadowDeclinesTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `public class AtomicSourceShadowProbe {
 static class AtomicIntegerFieldUpdater<T>{static AtomicIntegerFieldUpdater<Object> newUpdater(Class<Object> c,String n){return null;}int get(T value){return 71;}}
 int use(AtomicIntegerFieldUpdater<Object> u){return u.get(null);}}
 `)
	if strings.Contains(generated, "stdjava.AtomicIntegerFieldUpdater") || strings.Contains(generated, "NewAtomicIntegerFieldUpdater") {
		t.Fatalf("source owner hijacked\n%s", generated)
	}
}
func TestAtomicUpdaterForeignQualifierDeclinesTDD(t *testing.T) {
	for _, name := range []string{"foreign.AtomicIntegerFieldUpdater", "foreign.AtomicLongFieldUpdater", "foreign.AtomicReferenceFieldUpdater", "foreign.IntUnaryOperator", "foreign.UnaryOperator"} {
		if _, ok := stdjavaRuntimeTypeExpr(name, []string{"Object"}, nil, Ctx{}); ok {
			t.Fatalf("foreign owner mapped: %s", name)
		}
	}
}
func TestAtomicUpdaterCallbackFacadesTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.*;
 public class AtomicCallbackProbe {volatile int count;volatile long stamp;volatile Object value;
 int ints(AtomicIntegerFieldUpdater<AtomicCallbackProbe> u){return u.getAndUpdate(this,x->x+1)+u.accumulateAndGet(this,2,(a,b)->a+b);}
 long longs(AtomicLongFieldUpdater<AtomicCallbackProbe> u){return u.updateAndGet(this,x->x+1)+u.getAndAccumulate(this,2L,(a,b)->a+b);}
 Object refs(AtomicReferenceFieldUpdater<AtomicCallbackProbe,Object> u){return u.updateAndGet(this,x->x);}}
 `)
	calls := updaterTDDCalls(file)
	for _, name := range []string{"NewIntUnaryOperatorFuncAdapter", "NewIntBinaryOperatorFuncAdapter", "NewLongUnaryOperatorFuncAdapter", "NewLongBinaryOperatorFuncAdapter", "NewUnaryOperatorFuncAdapter"} {
		if calls[name] != 1 {
			t.Fatalf("execution SAM adapter %s missing\n%s", name, generated)
		}
	}
	if calls["ReferenceUnaryOperatorCallbackExecution"] != 1 {
		t.Fatalf("reference callback projection missing\n%s", generated)
	}
}
func TestAtomicUpdaterReceiverArgumentsBeforeNullCheckTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
 public class AtomicOrderProbe {volatile int count;static int trace;static AtomicOrderProbe target(){trace=trace*10+1;return null;}static int value(){trace=trace*10+2;return 4;}
 static void invoke(AtomicIntegerFieldUpdater<AtomicOrderProbe> u){u.set(target(),value());}}`)
	targetName := updaterTDDStaticExecutionName(t, file, "AtomicOrderProbe", "target")
	valueName := updaterTDDStaticExecutionName(t, file, "AtomicOrderProbe", "value")
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		fn, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		calls := updaterTDDCalls(fn.Body)
		if calls["SetExecution"] != 1 {
			return true
		}
		found = true
		target, value, check := -1, -1, -1
		for i, stmt := range fn.Body.List {
			c := updaterTDDCalls(stmt)
			if c[targetName] > 0 {
				target = i
			}
			if c[valueName] > 0 {
				value = i
			}
			if c["ReferenceRequireNonNull"] > 0 {
				check = i
			}
		}
		if target < 0 || value <= target || check <= value {
			t.Fatalf("receiver arguments must precede updater null check: %d,%d,%d\n%s", target, value, check, generated)
		}
		return true
	})
	if !found {
		t.Fatalf("staged call closure missing\n%s", generated)
	}
}
func TestAtomicUpdaterClassLiteralNoInitializationTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
 public class AtomicInitializationProbe {static class Cold{public static int initialized=side();public volatile int count;static int side(){return 4;}}
 static AtomicIntegerFieldUpdater<Cold> factory(){return AtomicIntegerFieldUpdater.newUpdater(Cold.class,"count");}}`)
	factoryName := updaterTDDStaticExecutionName(t, file, "AtomicInitializationProbe", "factory")
	owner := symbol.GlobalScope.FindPackage("").FindClassScope("AtomicInitializationProbe")
	cold := owner.FindClassScope("Cold")
	if cold == nil || cold.Enclosing != owner {
		t.Fatal("selected nested Cold owner missing")
	}
	coldEnsureName := symbol.GoIdentifier(classInitializationEnsureName(cold))
	found := false
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != factoryName {
			continue
		}
		found = true
		calls := updaterTDDCalls(fn.Body)
		if calls["NewAtomicIntegerFieldUpdaterJavaString"] != 1 {
			t.Fatalf("factory missing\n%s", generated)
		}
		if calls["ClassLiteral"] != 1 {
			t.Fatalf("factory did not retain the selected class literal\n%s", generated)
		}
		if calls[coldEnsureName] != 0 {
			t.Fatalf("metadata/factory initialized selected Cold owner\n%s", generated)
		}
		for name := range calls {
			if strings.Contains(name, "Cold") && strings.Contains(name, "Initialize") {
				t.Fatalf("metadata/factory initialized Cold\n%s", generated)
			}
		}
	}
	if !found {
		t.Fatalf("factory source method missing\n%s", generated)
	}
}

func TestAtomicUpdaterLexicalBinderDeclinesTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `public class AtomicBinderProbe<AtomicIntegerFieldUpdater>{AtomicIntegerFieldUpdater value;AtomicIntegerFieldUpdater identity(AtomicIntegerFieldUpdater v){return v;}}`)
	if strings.Contains(generated, "stdjava.AtomicIntegerFieldUpdater") {
		t.Fatalf("lexical type binder mapped to native updater\n%s", generated)
	}
}
func TestAtomicUpdaterValueQualifierDeclinesTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `public class AtomicQualifierProbe {static class Family{Object newUpdater(Class<?> c,String n){return null;}}static Family AtomicIntegerFieldUpdater;static Object use(){return AtomicIntegerFieldUpdater.newUpdater(Object.class,"value");}}`)
	if strings.Contains(generated, "NewAtomicIntegerFieldUpdaterJavaString") {
		t.Fatalf("value qualifier hijacked as static native owner\n%s", generated)
	}
}
func TestAtomicUpdaterSourceNamespaceDeclinesTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `package java.util.concurrent.atomic;public class AtomicIntegerFieldUpdater<T>{int get(T value){return 8;}int use(AtomicIntegerFieldUpdater<T> u,T value){return u.get(value);}}`)
	if strings.Contains(generated, "stdjava.AtomicIntegerFieldUpdater") {
		t.Fatalf("source namespace declaration replaced by facade\n%s", generated)
	}
}
