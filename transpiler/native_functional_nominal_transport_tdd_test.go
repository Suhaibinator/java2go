package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"strings"
	"testing"
)

func nominalTransportType(n ast.Node) string {
	var b bytes.Buffer
	if err := format.Node(&b, token.NewFileSet(), n); err != nil {
		panic(err)
	}
	return b.String()
}
func nominalTransportRequireCheckedCast(t *testing.T, file *ast.File, types ...string) {
	t.Helper()
	found := map[string]int{}
	unchecked := map[string]int{}
	ast.Inspect(file, func(n ast.Node) bool {
		if a, ok := n.(*ast.TypeAssertExpr); ok && a.Type != nil {
			typ := nominalTransportType(a.Type)
			for _, wanted := range types {
				if typ == wanted {
					unchecked[typ]++
				}
			}
		}
		if c, ok := n.(*ast.CallExpr); ok && updaterTDDCallName(c) == "ObjectView" && len(c.Args) == 2 {
			var typ ast.Expr
			switch f := c.Fun.(type) {
			case *ast.IndexExpr:
				typ = f.Index
			case *ast.IndexListExpr:
				if len(f.Indices) == 1 {
					typ = f.Indices[0]
				}
			}
			if typ != nil {
				written := nominalTransportType(typ)
				for _, wanted := range types {
					if written == wanted {
						found[written]++
					}
				}
			}
		}
		return true
	})
	for _, wanted := range types {
		if unchecked[wanted] != 0 {
			t.Fatalf("unchecked structural SAM assertion before nominal source view: %s (%d)", wanted, unchecked[wanted])
		}
		if found[wanted] == 0 {
			t.Fatalf("checked nominal SAM projection missing: %s", wanted)
		}
	}
}
func TestNativeFunctionalAliasNominalTransportTDD(t *testing.T) {
	source, err := os.ReadFile("testdata/operator_functional_prerequisite/MultipleSamViewsProbe.java")
	if err != nil {
		t.Fatal(err)
	}
	file, _ := updaterTDDParse(t, string(source))
	nominalTransportRequireCheckedCast(t, file, "stdjava.IntUnaryOperator", "stdjava.IntBinaryOperator", "stdjava.Function[*stdjava.JavaString, *stdjava.JavaString]", "stdjava.Consumer[*stdjava.JavaString]")
}
func TestNativeFunctionalErasedReturnNominalTransportTDD(t *testing.T) {
	file, _ := updaterTDDParse(t, `import java.util.function.*;public class ErasedControl{static class Both implements IntUnaryOperator,IntBinaryOperator{public int applyAsInt(int a){return a+1;}public int applyAsInt(int a,int b){return a+b;}} Object erased(Both value){return value;} IntUnaryOperator unary(Both value){return (IntUnaryOperator)erased(value);} IntBinaryOperator binary(Both value){return (IntBinaryOperator)erased(value);}}`)
	nominalTransportRequireCheckedCast(t, file, "stdjava.IntUnaryOperator", "stdjava.IntBinaryOperator")
}
func TestNativeFunctionalUpdaterReturnNominalTransportTDD(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.*;import java.util.concurrent.atomic.AtomicReferenceFieldUpdater;public class UpdaterControl{static class Cell{volatile Object value;} IntUnaryOperator unary(AtomicReferenceFieldUpdater<Cell,Object> updater,Cell cell){return (IntUnaryOperator)updater.getAndUpdate(cell,x->x);} IntBinaryOperator binary(AtomicReferenceFieldUpdater<Cell,Object> updater,Cell cell){return (IntBinaryOperator)updater.getAndUpdate(cell,x->x);}}`)
	if updaterTDDCalls(file)["GetAndUpdateExecution"] != 2 {
		t.Fatalf("updater transport did not exercise both actual runtime results\n%s", generated)
	}
	nominalTransportRequireCheckedCast(t, file, "stdjava.IntUnaryOperator", "stdjava.IntBinaryOperator")
}
func TestNativeFunctionalSourceIdentityExecutionTransportTDD(t *testing.T) {
	source, err := os.ReadFile("testdata/operator_functional_prerequisite/MultipleSamViewsProbe.java")
	if err != nil {
		t.Fatal(err)
	}
	file, _ := updaterTDDParse(t, string(source))
	views := 0
	ast.Inspect(file, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || !strings.HasSuffix(updaterTDDCallName(c), "SourceView") {
			return true
		}
		views++
		if len(c.Args) != 2 {
			t.Fatal("source view needs original ObjectInfo and caller Execution closure")
		}
		info, ok := c.Args[0].(*ast.SelectorExpr)
		if !ok || info.Sel.Name != "ObjectInfo" {
			t.Fatal("source view lost shared ObjectInfo")
		}
		closure, ok := c.Args[1].(*ast.FuncLit)
		if !ok || closure.Type.Params == nil || len(closure.Type.Params.List) == 0 || len(closure.Type.Params.List[0].Names) != 1 || closure.Type.Params.List[0].Names[0].Name != "execution" {
			t.Fatal("source view omitted consuming Execution")
		}
		forwarded := 0
		ast.Inspect(closure.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selected, ok := call.Fun.(*ast.SelectorExpr)
			if ok && strings.Contains(selected.Sel.Name, "Java2goExecution") && len(call.Args) > 0 {
				if arg, ok := call.Args[0].(*ast.Ident); ok && arg.Name == "execution" {
					forwarded++
				}
			}
			return true
		})
		if forwarded != 1 {
			t.Fatal("view must forward consuming Execution once to declaration-selected implementation")
		}
		return false
	})
	if views != 4 {
		t.Fatalf("expected all four original declared native views, got %d", views)
	}
}
func TestNativeFunctionalShadowNominalTransportTDD(t *testing.T) {
	_, generated := updaterTDDParse(t, `public class ShadowTransport{interface IntUnaryOperator{int applyAsInt(int x);} IntUnaryOperator read(Object value){return (IntUnaryOperator)value;}}`)
	if strings.Contains(generated, "stdjava.IntUnaryOperator") || strings.Contains(generated, `"java.util.function.IntUnaryOperator"`) {
		t.Fatal("source shadow was authorized as canonical native SAM")
	}
}
