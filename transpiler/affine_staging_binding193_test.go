package transpiler

import (
	"fmt"
	"go/ast"
	"maps"
	"reflect"
	"strings"
	"testing"
)

// Existing APIs, original declarations and original executable companion only.
// The historical public parent remains active with all original assertions.
func TestCI193AffineOriginalTypeParameterStagingCompile(t *testing.T) {
	const original = `
final class TypeParameterGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
final class TypeParameterDoubleGrid {
    private final double[] values = new double[1];
    private final int size = 1;
    void set(int row, int column, double value) {
        this.values[row * this.size + column] = value;
    }
}
public class TypeParameterLoopProgram<panic> {
    public static TypeParameterGrid grid() { return new TypeParameterGrid(); }
    public static TypeParameterDoubleGrid doubleGrid() { return new TypeParameterDoubleGrid(); }
    public static TypeParameterLoopProgram<Integer> program() {
        return new TypeParameterLoopProgram<Integer>();
    }
    public int classRuntime(TypeParameterGrid grid, panic ignored) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1;
    }
    public static <stdjava> int methodRuntime(TypeParameterGrid grid, stdjava ignored) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1;
    }
    public static <float64> void methodType(TypeParameterDoubleGrid grid, float64 ignored) {
        for (int once = 0; once < 1; once++) {
            grid.set(0, 0, 1.0);
            break;
        }
    }
}
`
	out := renderGoFileFromJava(t, original)
	runGoTestInTempModule(t, out, `
package main
import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)
func TestTypeParameterFallback(t *testing.T) {
    grid := Grid()
    if got := Program().ClassRuntime(grid, stdjava.BoxInteger(0)); got != 0 {
        t.Fatalf("ClassRuntime() = %d, want 0", got)
    }
    if got := MethodRuntime(grid, stdjava.BoxInteger(0)); got != 0 {
        t.Fatalf("MethodRuntime() = %d, want 0", got)
    }
    MethodType(DoubleGrid(), stdjava.BoxInteger(0))
}
`)
}

func TestCI193HelpfulPrimitiveStagingBindingAdmission(t *testing.T) {
	for _, row := range []struct{ java, goName, literal string }{
		{"byte", "int8", "value"}, {"short", "int16", "value"},
		{"char", "rune", "'a'"}, {"int", "int32", "1"}, {"long", "int64", "1L"},
		{"float", "float32", "1.0f"}, {"double", "float64", "1.0"}, {"boolean", "bool", "true"},
	} {
		t.Run(row.java, func(t *testing.T) {
			for _, mode := range []string{"safe", "method", "class", "formal", "local", "nested", "unrelated"} {
				t.Run(mode, func(t *testing.T) {
					header, method, extra, body := "class Main", "static void run", "", "lane.call("+row.literal+");"
					if row.java == "byte" || row.java == "short" {
						extra = ", " + row.java + " value"
					}
					switch mode {
					case "method":
						method = "static <" + row.goName + "> void run"
					case "class":
						header = "class Main<" + row.goName + ">"
						method = "void run"
					case "formal":
						extra += ", Object " + row.goName
					case "local":
						body = "Object " + row.goName + " = null; " + body
					case "nested":
						body = "for (Object " + row.goName + " : new Object[]{null}) { " + body + " }"
					case "unrelated":
						body = "int unrelatedBinding = 0; " + body
					}
					source := fmt.Sprintf("class Target{void call(%s value){}}%s{%s(Target lane%s){%s}}", row.java, header, method, extra, body)
					ctx, n, s := helpfulNPEContext(t, source, "Main", "run", "call")
					resolution := helpfulNPEResolution(t, ctx, n, s)
					// Check the actual existing primitive lowering map used by captures.
					physical, ok := javaTypeStringToGoTypeExpr(row.java, nil, ctx).(*ast.Ident)
					if !ok || physical.Name != row.goName {
						t.Fatalf("actual physical type=%#v, want %s", physical, row.goName)
					}
					imports, aliases := maps.Clone(ctx.usedImports), maps.Clone(ctx.importAliases)
					diagnostics := len(Diagnostics())
					plan := helpfulSourceInvocationStagingPlan(n, n.ChildByFieldName("object"), resolution, ctx, s)
					if mode == "formal" || mode == "local" || mode == "nested" {
						if actual := localBindingName(row.goName, ctx); actual != row.goName {
							t.Fatalf("counter premise: actual binder %s want %s", actual, row.goName)
						}
					}
					safe := mode == "safe" || mode == "unrelated"
					if (plan != nil) != safe {
						t.Fatalf("%s/%s admission=%t want %t", row.java, mode, plan != nil, safe)
					}
					if !reflect.DeepEqual(imports, ctx.usedImports) || !reflect.DeepEqual(aliases, ctx.importAliases) || len(Diagnostics()) != diagnostics {
						t.Fatal("admission changed imports, aliases or diagnostics")
					}
				})
			}
		})
	}
}
func TestCI193HelpfulPrimitiveResultBindingAdmission(t *testing.T) {
	for _, row := range []struct{ java, goName, value string }{{"int", "int32", "1"}, {"double", "float64", "1.0"}, {"boolean", "bool", "true"}} {
		source := fmt.Sprintf("class Target{%s call(){return %s;}}class Main{static <%s> void run(Target lane){lane.call();}}", row.java, row.value, row.goName)
		ctx, n, s := helpfulNPEContext(t, source, "Main", "run", "call")
		if plan := helpfulSourceInvocationStagingPlan(n, n.ChildByFieldName("object"), helpfulNPEResolution(t, ctx, n, s), ctx, s); plan != nil {
			t.Fatalf("shadowed result type %s admitted", row.goName)
		}
	}
}

func TestCI193HelpfulUnrelatedScopeKeepsDirectStaging(t *testing.T) {
	for _, row := range []struct{ name, prefix, body string }{
		{"other-formal", "class Other{void unused(int int32){}}", "lane.call(1);"},
		{"other-local", "class Other{void unused(){int int32=0;}}", "lane.call(1);"},
		{"disjoint-block", "", "{int int32=0;} lane.call(1);"},
		{"later-local", "", "lane.call(1); int int32=0;"},
		{"other-method", "", "lane.call(1);"},
	} {
		t.Run(row.name, func(t *testing.T) {
			extra := ""
			if row.name == "other-method" {
				extra = "void unused(int int32){}"
			}
			raw := row.prefix + "class Target{void call(int value){}}class Main{" + extra + "static void run(Target lane){" + row.body + "}}"
			ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
			out := helpfulNPEText(t, ParseExpr(n, s, ctx))
			if !strings.Contains(out, "RequireInvocationReceiver(") {
				t.Fatalf("unrelated declaration suppressed required direct staging: %s", out)
			}
		})
	}
}
func TestCI193HelpfulPrimitiveArrayStagingAdmission(t *testing.T) {
	for _, row := range []struct{ name, target, body string }{
		{"result", "double[] call(){return null;}", "lane.call();"},
		{"null-formal", "void call(double[] value){}", "lane.call(null);"},
		{"parenthesized-null", "void call(double[] value){}", "lane.call((null));"},
	} {
		t.Run(row.name, func(t *testing.T) {
			raw := "class Target{" + row.target + "}class Main{static <float64> void run(Target lane){" + row.body + "}}"
			ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
			out := helpfulNPEText(t, ParseExpr(n, s, ctx))
			if strings.Contains(out, "RequireInvocationReceiver(") {
				t.Fatalf("primitive-array type parameter capture admitted unsafe new staging: %s", out)
			}
		})
	}
}
func TestCI193HelpfulAlreadyStagedChecksRemain(t *testing.T) {
	for _, raw := range []string{
		"class Target{int call(int value){return value;}}class Child extends Target{int call(int value){return 7;}}class Main{static <int32> void run(Target lane){lane.call(1);}}",
		"interface Target{int call(int value);}class Main{static <int32> void run(Target lane){lane.call(1);}}",
	} {
		ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
		out := helpfulNPEText(t, ParseExpr(n, s, ctx))
		if !strings.Contains(out, "RequireInvocationReceiver(") {
			t.Fatalf("direct-admission refusal suppressed check on an already staged path: %s", out)
		}
	}
}

func TestCI193HelpfulPrimitiveArrayProjectionControls(t *testing.T) {
	for _, row := range []struct{ java, goName string }{{"byte", "int8"}, {"short", "int16"}, {"char", "rune"}, {"int", "int32"}, {"long", "int64"}, {"float", "float32"}, {"double", "float64"}, {"boolean", "bool"}} {
		for _, mode := range []string{"result", "null", "rank2"} {
			t.Run(row.java+"/"+mode, func(t *testing.T) {
				target, body := row.java+"[] call(){return null;}", "lane.call();"
				if mode == "null" {
					target = "void call(" + row.java + "[] value){}"
					body = "lane.call(null);"
				}
				if mode == "rank2" {
					target = row.java + "[][] call(){return null;}"
				}
				raw := "class Target{" + target + "}class Main{static <" + row.goName + "> void run(Target lane){" + body + "}}"
				ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
				resolved := helpfulNPEResolution(t, ctx, n, s)
				metadata := helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), resolved, ctx, s)
				if metadata == nil {
					t.Fatal("metadata API lost source-owned diagnostic")
				}
				plan := helpfulSourceInvocationStagingPlan(n, n.ChildByFieldName("object"), resolved, ctx, s)
				if (plan != nil) != (mode == "rank2") {
					t.Fatalf("%s %s admission=%t", row.java, mode, plan != nil)
				}
			})
		}
	}
}

// Go value bindings start after their own declarator, unlike Java's initializer scope.
func TestCI193HelpfulLexicalInitializerBoundary(t *testing.T) {
	for _, row := range []struct {
		name, target, body string
		admit              bool
	}{
		{"own-initializer", "Object call(int value){return null;}", "Object int32=lane.call(1);", true},
		{"previous-declarator", "Object call(int value){return null;}", "Object int32=null, other=lane.call(1);", false},
		{"later-declarator", "Object call(int value){return null;}", "Object other=lane.call(1), int32=null;", true},
		{"for-own-initializer", "Object call(int value){return null;}", "for(Object int32=lane.call(1);;){break;}", true},
		{"enhanced-iterable", "Object[] call(int value){return null;}", "for(Object int32:lane.call(1)){}", true},
		{"enhanced-body", "Object call(int value){return null;}", "for(Object int32:new Object[]{null}){lane.call(1);}", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			raw := "class Target{" + row.target + "}class Main{static void run(Target lane){" + row.body + "}}"
			ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
			if actual := localBindingName("int32", ctx); actual != "int32" {
				t.Fatalf("counter premise: actual binder %s", actual)
			}
			out := helpfulNPEText(t, ParseExpr(n, s, ctx))
			if got := strings.Contains(out, "RequireInvocationReceiver("); got != row.admit {
				t.Fatalf("staging=%t want %t: %s", got, row.admit, out)
			}
		})
	}
}

// A cast already demands the primitive type in the body, so existing hygiene
// renames these value binders. Staging must use that actual emitted spelling.
func TestCI193HelpfulCastRenamedPrimitiveBinderAdmits(t *testing.T) {
	for _, row := range []struct{ java, goName string }{{"byte", "int8"}, {"short", "int16"}} {
		for _, mode := range []string{"formal", "local", "nested"} {
			t.Run(row.java+"/"+mode, func(t *testing.T) {
				extra, body := "", "lane.call(("+row.java+")1);"
				switch mode {
				case "formal":
					extra = ", Object " + row.goName
				case "local":
					body = "Object " + row.goName + "=null;" + body
				case "nested":
					body = "for(Object " + row.goName + ":new Object[]{null}){" + body + "}"
				}
				raw := "class Target{void call(" + row.java + " value){}}class Main{static void run(Target lane" + extra + "){" + body + "}}"
				ctx, n, s := helpfulNPEContext(t, raw, "Main", "run", "call")
				if actual := localBindingName(row.goName, ctx); actual == row.goName {
					t.Fatalf("cast did not hygienize actual binder %s", actual)
				}
				if plan := helpfulSourceInvocationStagingPlan(n, n.ChildByFieldName("object"), helpfulNPEResolution(t, ctx, n, s), ctx, s); plan == nil {
					t.Fatal("already hygienic binding refused staging")
				}
			})
		}
	}
}
