package transpiler

import (
	"go/ast"
	"strings"
	"testing"
)

func TestOperatorParentNativeFacadesTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.*;public class ParentControl { Function<String,String> unary(UnaryOperator<String> u){Object o=u;return (Function<String,String>)o;} BiFunction<String,String,String> binary(BinaryOperator<String> b){Object o=b;return (BiFunction<String,String,String>)o;} }`)
	calls := updaterTDDCalls(file)
	if calls["ObjectView"] < 2 || !strings.Contains(generated, "stdjava.BiFunction[") {
		t.Fatalf("operator parent must have a nominal object view and native interface ABI\n%s", generated)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if strings.HasPrefix(fn.Name.Name, "Binary") && fn.Type.Results != nil {
			if _, plain := fn.Type.Results.List[0].Type.(*ast.FuncType); plain {
				t.Fatalf("BiFunction parent remains native callback instead of Java object\n%s", generated)
			}
		}
	}
}
func TestOperatorMultipleNativeViewsTDD(t *testing.T) {
	for _, edges := range []string{"IntUnaryOperator,IntBinaryOperator", "IntBinaryOperator,IntUnaryOperator"} {
		_, generated := updaterTDDParse(t, `import java.util.function.*;public class MultipleControl implements `+edges+` {public int applyAsInt(int a){return a+1;}public int applyAsInt(int a,int b){return a+b;} static int use(MultipleControl source){Object o=source;IntUnaryOperator u=(IntUnaryOperator)o;IntBinaryOperator b=(IntBinaryOperator)o;return u.applyAsInt(1)+b.applyAsInt(2,3);}}`)
		for _, required := range []string{"NewIntUnaryOperatorSourceView", "NewIntBinaryOperatorSourceView", "Java2goReferenceView", `"java.util.function.IntUnaryOperator"`, `"java.util.function.IntBinaryOperator"`} {
			if !strings.Contains(generated, required) {
				t.Fatalf("declared SAM lost representable ABI or nominal edge: %s\n%s", required, generated)
			}
		}
	}
}
func TestOperatorInheritedNativeEdgesTDD(t *testing.T) {
	for _, edges := range []string{"Transform,Consumer<String>", "Consumer<String>,Transform"} {
		_, generated := updaterTDDParse(t, `import java.util.function.*;interface Transform extends Function<String,String>{} public class InheritedControl implements `+edges+`{public String apply(String a){return a;}public void accept(String a){} Object both(){return this;}}`)
		for _, required := range []string{"NewFunctionSourceView", "NewConsumerSourceView", `"java.util.function.Function"`, `"java.util.function.Consumer"`} {
			if !strings.Contains(generated, required) {
				t.Fatalf("inherited closure or multiple-edge physical view missing: %s\n%s", required, generated)
			}
		}
	}
}
func TestOperatorNativeNamespaceDeclinesTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `public class ShadowControl {interface IntUnaryOperator{int applyAsInt(int a);} static class Mine implements IntUnaryOperator {public int applyAsInt(int a){return a;}} int use(IntUnaryOperator u){return u.applyAsInt(4);}}`)
	if strings.Contains(generated, "stdjava.IntUnaryOperator") || strings.Contains(generated, "NewIntUnaryOperatorSourceView") {
		t.Fatalf("source SAM shadow borrowed external nominal edge\n%s", generated)
	}
	for _, name := range []string{"foreign.BiFunction", "foreign.Consumer", "foreign.UnaryOperator", "foreign.IntUnaryOperator"} {
		if _, mapped := stdjavaRuntimeTypeExpr(name, nil, nil, Ctx{}); mapped {
			t.Fatalf("foreign canonical owner mapped: %s", name)
		}
	}
	_, generated = updaterTDDParse(t, `public class BinderControl<IntUnaryOperator>{IntUnaryOperator value;IntUnaryOperator read(){return value;}}`)
	if strings.Contains(generated, "stdjava.IntUnaryOperator") {
		t.Fatalf("lexical type binder borrowed operator native ABI\n%s", generated)
	}
}
func TestOperatorCollisionViewExecutionTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.IntUnaryOperator;public class CollisionControl implements IntUnaryOperator {public int applyAsIntJava2goExecution(int x){return -1;}public int applyAsInt(int x){return x+1;}}`)
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || updaterTDDCallName(call) != "NewIntUnaryOperatorSourceView" {
			return true
		}
		found = true
		if len(call.Args) != 2 {
			t.Fatal("native view must bind canonical info and one source closure")
		}
		closure, ok := call.Args[1].(*ast.FuncLit)
		if !ok || closure.Type.Params.NumFields() != 2 {
			t.Fatal("view closure must receive consuming Execution")
		}
		valid := false
		ast.Inspect(closure.Body, func(n ast.Node) bool {
			invoke, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := invoke.Fun.(*ast.SelectorExpr)
			if ok && strings.HasPrefix(selector.Sel.Name, "ApplyAsIntJava2goExecution") && selector.Sel.Name != "ApplyAsIntJava2goExecution" {
				valid = true
			}
			return true
		})
		if !valid {
			t.Fatalf("view did not resolve collision-safe actual implementation\n%s", generated)
		}
		return false
	})
	if !found {
		t.Fatalf("source callback has no explicit execution view\n%s", generated)
	}
}
func TestOperatorDefaultCallerExecutionTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.*;public class DefaultsControl { Function<String,Integer> transform(Function<String,String> f){return f.andThen(String::length);} BiFunction<String,String,Integer> transform2(BiFunction<String,String,String> f){return f.andThen(String::length);} IntUnaryOperator ints(IntUnaryOperator u){return u.andThen(x->x*2);} }`)
	calls := updaterTDDCalls(file)
	for _, required := range []string{"FunctionAndThenExecution", "BiFunctionAndThenExecution", "IntUnaryOperatorAndThenExecution"} {
		if calls[required] != 1 {
			t.Fatalf("default method caller-execution dispatch missing %s\n%s", required, generated)
		}
	}
	if calls["FunctionCallbackExecution"] != 0 {
		t.Fatalf("composition captured construction Execution through native callback boundary\n%s", generated)
	}
}
