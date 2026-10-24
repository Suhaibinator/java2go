package transpiler

import "testing"

func TestIntegralSummaryGetSumUsesLongAccessor(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.util.stream.*; public class IntegralStatistics { static void run() {long a=IntStream.of(1).summaryStatistics().getSum();long b=LongStream.of(1L).summaryStatistics().getSum();double c=DoubleStream.of(1).summaryStatistics().getSum();} }`)
	assertContains(t, out, ".GetSumLong()")
	assertContains(t, out, ".GetSum()")
}
