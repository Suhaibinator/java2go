package transpiler

import "testing"

func TestIntegralStatisticsStrictJDK21Parity(t *testing.T) {
	const source = `import java.util.*;import java.util.stream.*;
public class IntegralStatisticsContract {
 public static String run(){
 IntSummaryStatistics ints=IntStream.of(Integer.MAX_VALUE,Integer.MAX_VALUE).summaryStatistics();
 LongSummaryStatistics longs=LongStream.of(9007199254740992L,1L,-9007199254740992L).summaryStatistics();
 LongSummaryStatistics overflow=LongStream.of(Long.MAX_VALUE,1L).summaryStatistics();
 IntSummaryStatistics empty=IntStream.empty().summaryStatistics();
 DoubleSummaryStatistics doubles=DoubleStream.of(1.5,2.5).summaryStatistics();
 return ints.getSum()+":"+ints.getAverage()+":"+longs.getSum()+":"+longs.getAverage()+":"+LongStream.of(9007199254740992L,1L,-9007199254740992L).average().getAsDouble()+":"+overflow.getSum()+":"+overflow.getAverage()+":"+LongStream.of(Long.MAX_VALUE,1L).average().getAsDouble()+":"+empty.getSum()+":"+empty.getAverage()+":"+doubles.getSum()+":"+doubles.getAverage();
 }
 public static void main(String[] args){System.out.print(run());}
}`
	want := stringCaseReferenceJavaOracle(t, "IntegralStatisticsContract", source)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>integral-statistics</artifactId><version>1</version></project>`,
		"src/main/java/IntegralStatisticsContract.java": source,
	}, "IntegralStatisticsContract", want)
}
