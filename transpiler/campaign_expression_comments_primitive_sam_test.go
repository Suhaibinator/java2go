package transpiler

import (
	"fmt"
	"strconv"
	"testing"
)

// Preserve the primitive SAM blocker independently of all comment syntax.
func TestCampaignExpressionCommentsPrimitiveSAMControlJVM(t *testing.T) {
	const source = `public class ExpressionComments {static int trace;static int mark(int x){trace=trace*10+x;return x;}
 public static int run(){java.util.function.IntBinaryOperator op=(a , b )  ->  (a  + b);int x=op.applyAsInt(mark(1),mark(2) );return trace*100+x;}}`
	want := campaignRuntimeJavaOracle(t, "ExpressionComments", source)
	value, err := strconv.ParseInt(want, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestPrimitiveSAMControl(t *testing.T){if got:=Run();got!=int32(%d){t.Fatalf("JVM %%d != Go %%d",int32(%d),got)}}
`, value, value))
}
