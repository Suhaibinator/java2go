package transpiler

import (
	"fmt"
	"go/ast"
	"testing"
)

func TestMathExtrema181PhysicalWidthsWholeProgram(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;
import static java.lang.Math.max;
import static java.lang.Math.min;
class Math {static int max(int a,int b){return -17;}static double min(double a,double b){return -19.0;}}
public class Main {
 static int trace;
 static int mi(int x){trace=trace*10+x;return x;}
 static long ml(int x){trace=trace*10+x;return x;}
 static float mf(int x){trace=trace*10+x;return x;}
 static double md(int x){trace=trace*10+x;return x;}
 static int ri(){return java.lang.Math.max(1,2);}
 static long rl(){return java.lang.Math.max(9007199254740993L,9007199254740992L);}
 static float rf(){return java.lang.Math.max(1.0f,2.0f);}
 static double rd(){return java.lang.Math.min(1.0,2.0);}
 static int ti(int x){return x;}
 static long tl(long x){return x;}
 static float tf(float x){return x;}
 static double td(double x){return x;}
 public static void main(String[] args){
  int i=max(1,2);
  long l=max(9007199254740993L,9007199254740992L);
  float f=max(1.0f,2.0f);
  double d=min(1.0,2.0);
  System.out.println("assign:"+(i==2)+":"+(l==9007199254740993L)+":"+(f==2.0f)+":"+(d==1.0));
  System.out.println("return:"+(ri()==2)+":"+(rl()==9007199254740993L)+":"+(rf()==2.0f)+":"+(rd()==1.0));
  System.out.println("args:"+(ti(max(1,2))+ti(java.lang.Math.min(1,2))==3)+":"+(tl(max(1L,2L))+tl(java.lang.Math.min(1L,2L))==3L)+":"+(tf(max(1.0f,2.0f))+tf(java.lang.Math.min(1.0f,2.0f))==3.0f)+":"+(td(max(1.0,2.0))+td(java.lang.Math.min(1.0,2.0))==3.0));
  trace=0;float mixed=max(mi(1),mf(2));double wide=java.lang.Math.min(ml(3),md(4));
  System.out.println("effects:"+trace+":"+(mixed==2.0f)+":"+(wide==3.0));
  Integer boxed=7;Long longBox=9L;int boxedI=max(boxed,2);long boxedL=java.lang.Math.min(longBox,10L);
  System.out.println("boxed:"+(boxedI==7)+":"+(boxedL==9L));
  System.out.println("source:"+(Math.max(1,2)==-17)+":"+(Math.min(1.0,2.0)==-19.0));
 }
}`,
	}, "probe.Main", "assign:true:true:true:true\nreturn:true:true:true:true\nargs:true:true:true:true\neffects:1234:true:true\nboxed:true:true\nsource:true:true\n")
}

func TestMathExtrema181PrimitiveTypeArgsAndBoxedFallback(t *testing.T) {
	for _, tc := range []struct{ left, right, want string }{
		{"int", "int", "int32"}, {"long", "int", "int64"}, {"int", "long", "int64"},
		{"float", "float", "float32"}, {"long", "float", "float32"}, {"float", "double", "float64"},
		{"double", "double", "float64"}, {"byte", "short", "int32"}, {"char", "int", "int32"},
		{"Integer", "int", ""}, {"Long", "long", ""}, {"boolean", "boolean", ""},
		{"String", "int", ""}, {"int[]", "int[]", ""}, {"T", "T", ""},
	} {
		for _, invocation := range []string{"max(a,b)", "java.lang.Math.max(a,b)"} {
			t.Run(tc.left+"-"+tc.right+"/"+invocation, func(t *testing.T) {
				source := fmt.Sprintf("import static java.lang.Math.max; class Use<T> { void call(%s a,%s b){%s;} }", tc.left, tc.right, invocation)
				helper := setupParseHelper(t, source)
				ctx := helper.Ctx.Clone()
				ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
				node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
				ctx.expectedType = "java.lang.String"
				ctx.expectedTypeRoot = node
				got := mathExtremaPrimitiveTypeArgs(node, ctx, []byte(source))
				if tc.want == "" {
					if len(got) != 0 {
						t.Fatalf("nonprimitive/invalid operands changed the old fallback: %v", got)
					}
					return
				}
				if len(got) != 1 {
					t.Fatalf("known primitive Java width must produce one type argument, got %v", got)
				}
				width, ok := got[0].(*ast.Ident)
				if !ok || width.Name != tc.want {
					t.Fatalf("Java primitive width is %q, got %#v", tc.want, got[0])
				}
			})
		}
	}
}
