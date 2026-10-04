package transpiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestReferenceArrayNullAssignmentHasInferableInput(t *testing.T) {
	cases := []struct {
		name   string
		source string
		null   bool
		helper string
	}{
		{"object", `public class NullStoreProgram { static void clear(Object[] values) { values[0] = null; } }`, true, "ReferenceArrayAssign"},
		{"parenthesized", `public class NullStoreProgram { static void clear(Object[] values) { values[0] = ((null)); } }`, true, "ReferenceArrayAssign"},
		{"source-reference", `public class NullStoreProgram { static class Base {} static void clear(Base[] values) { values[0] = (null); } }`, true, "ReferenceArrayAssign"},
		{"generic-reference", `public class NullStoreProgram { static <T> void clear(T[] values) { values[0] = null; } }`, true, "ReferenceArrayAssign"},
		{"bounded-generic-reference", `public class NullStoreProgram { static class Base {} static <T extends Base> void clear(T[] values) { values[0] = null; } }`, true, "ReferenceArrayAssign"},
		{"nested-reference-array", `public class NullStoreProgram { static void clear(Object[][] values) { values[0] = null; } }`, true, "ReferenceArrayAssign"},
		{"string-reference", `public class NullStoreProgram { static void clear(String[] values) { values[0] = null; } }`, true, "ReferenceArrayAssign"},
		{"non-null-reference-control", `public class NullStoreProgram { static void write(Object[] values, Object value) { values[0] = value; } }`, false, "ReferenceArrayAssign"},
		{"primitive-control", `public class NullStoreProgram { static void write(int[] values, int value) { values[0] = value; } }`, false, "PrimitiveArrayAssign"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			generated := renderGoFileFromJava(t, tc.source)
			file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
			if err != nil {
				t.Fatalf("invalid generated Go: %v\n%s", err, generated)
			}
			calls := 0
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				var target ast.Expr
				var arguments []ast.Expr
				switch fun := call.Fun.(type) {
				case *ast.IndexExpr:
					target, arguments = fun.X, []ast.Expr{fun.Index}
				case *ast.IndexListExpr:
					target, arguments = fun.X, fun.Indices
				default:
					return true
				}
				selector, ok := target.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != tc.helper {
					return true
				}
				calls++
				if tc.null {
					if len(arguments) != 2 {
						t.Errorf("null store needs an explicit inferable Input type beside Result; got %d type arguments:\n%s", len(arguments), generated)
					} else if input, ok := arguments[1].(*ast.Ident); !ok || input.Name != "any" {
						t.Errorf("null store Input must accept every Java null reference view:\n%s", generated)
					}
				} else if len(arguments) != 1 {
					t.Errorf("non-null/primitive assignment changed its existing inference path:\n%s", generated)
				}
				return true
			})
			if calls != 1 {
				t.Fatalf("expected exactly one %s assignment, found %d:\n%s", tc.helper, calls, generated)
			}
		})
	}
}

func TestReferenceArrayNullAssignmentJVMParity(t *testing.T) {
	const source = `public class NullArrayStoreProgram {
 static class Base {} static class Child extends Base {}
 static int trace;
 static Object[] select(Object[] values) { trace=trace*10+1; return values; }
 static int index(int value) { trace=trace*10+2; return value; }
 static int badIndex() { trace=trace*10+2; throw new IllegalArgumentException(); }
 public static int run() {
  int score=0;
  Base[] typed=new Child[]{new Child()}; Object[] erased=typed;
  Object result=(erased[0]=null); if(result==null && typed[0]==null) score+=1;
  typed[0]=new Child(); Base assigned=(typed[0]=(null)); if(assigned==null && typed[0]==null) score+=2;
  Object[][] nested=new Object[][]{new Object[1]}; Object[] row=(nested[0]=null); if(row==null && nested[0]==null) score+=4;
  String[] words=new String[]{"seed"}; String text=(words[0]=null); if(text==null && words[0]==null) score+=8;
  trace=0; select(erased)[index(0)]=((null)); if(trace==12 && erased[0]==null) score+=16;
  Object[] missing=null; trace=0;
  try { select(missing)[index(0)]=null; } catch(NullPointerException expected) { if(trace==12) score+=32; }
  trace=0; try { select(erased)[index(9)]=null; } catch(ArrayIndexOutOfBoundsException expected) { if(trace==12 && erased[0]==null) score+=64; }
  trace=0; try { select(missing)[badIndex()]=null; } catch(IllegalArgumentException expected) { if(trace==12) score+=128; }
  return score;
 }
}`
	want := campaignRuntimeJavaOracle(t, "NullArrayStoreProgram", source)
	if want != "255" {
		t.Fatalf("JDK oracle did not exercise every null/result/covariance/order/bounds condition: %q", want)
	}
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("strconv";"testing")
func TestNullArrayStore(t *testing.T) { if got:=strconv.FormatInt(int64(Run()),10); got!=%q { t.Fatalf("JVM %%q != Go %%q",%q,got) } }
`, want, want))
}
