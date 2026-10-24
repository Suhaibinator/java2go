package transpiler

import (
	"fmt"
	"strconv"
	"testing"
)

// These programs retain calls and updates around trivia: skipping comments must
// preserve Java's argument order, lazy branches, and array/lambda operands.
func TestCampaignExpressionCommentsJVM(t *testing.T) {
	cases := map[string]string{
		"FractionArgumentShape": `static int addSub(int fraction, boolean add) { return add ? fraction : -fraction; }
 public static int run() { int fraction=7; return addSub(fraction, true /* add */); }`,
		"SideEffectArguments": `static int trace; static int mark(int x){trace=trace*10+x;return x;}
 static int combine(int a,int b,int c){return a*100+b*10+c;}
 public static int run(){int value=combine(/* first */mark(1) /* after */, // between arguments
 /* second */mark(2), mark(3) /* trailing */);return trace*1000+value;}`,
		"UnaryParenthesizedCast": `static int trace;static int mark(int x){trace=trace*10+x;return x;}
 public static int run(){int a=- /* unary */ (/* paren */mark(1) /* after */);int b=(/* cast type */ int /* after type */) /* value */ (/* paren */mark(2));return trace*100+a*10+b;}`,
		"BinaryConditional": `static int trace;static int mark(int x){trace=trace*10+x;return x;}
 public static int run(){int a=mark(1) /* left */ + /* right */mark(2);boolean p=false && /* short circuit */mark(9)>0;int b=(/* condition */a>0 /* before question */ ? /* consequence */mark(3) /* before colon */ : /* alternative */mark(8));return trace*100+a*10+b+(p?1:0);}`,
		"ArrayDimensionsInitializer": `static int trace;static int mark(int x){trace=trace*10+x;return x;}
 public static int run(){int[][] a=new int[/* dimension one */mark(1) /* end */][/* dimension two */mark(2)];int[] b=new int[]{/* first */mark(3) /* after */, /* second */mark(4) /* trailing */};return trace*100+a.length*10+a[0].length+b[0]+b[1];}`,
		"LambdaArgumentsAndParameters": `static int trace;static int mark(int x){trace=trace*10+x;return x;}
 public static int run(){java.util.function.IntBinaryOperator op=(/* first */a /* after */, /* second */b /* trailing */) /* arrow */ -> /* body */ (/* paren */a /* left */ + /* right */b);int x=op.applyAsInt(/* first */mark(1),/* second */mark(2) /* trailing */);return trace*100+x;}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			source := "public class ExpressionComments {" + body + "}"
			want := campaignRuntimeJavaOracle(t, "ExpressionComments", source)
			value, err := strconv.ParseInt(want, 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			generated := renderGoFileFromJava(t, source)
			runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestExpressionComments(t *testing.T){if got:=Run();got!=int32(%d){t.Fatalf("JVM %%d != Go %%d",int32(%d),got)}}
`, value, value))
		})
	}
}
