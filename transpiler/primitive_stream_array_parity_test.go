package transpiler

import "testing"

func TestPrimitiveStreamArrayStrictJDK21Parity(t *testing.T) {
	const source = `import java.util.stream.*;
public class PrimitiveStreamArrayContract {
 static int calls;static int[] values(){calls++;return new int[]{7,2,-3};}
 public static String run(){
 int[] ints={Integer.MAX_VALUE,1};long[] longs={Long.MAX_VALUE,1L};double[] doubles={1.5,2.5};
 String out=IntStream.of(ints).sum()+":"+LongStream.of(longs).sum()+":"+DoubleStream.of(doubles).sum()+":"+IntStream.of(values()).sum()+":"+calls+":"+IntStream.of(new int[0]).count();
 int[] absent=null;try{IntStream.of(absent);out+=":missing";}catch(NullPointerException ex){out+=":intnull";}
 try{LongStream.of(null);out+=":missing";}catch(NullPointerException ex){out+=":longnull";}
 try{DoubleStream.of(null);out+=":missing";}catch(NullPointerException ex){out+=":doublenull";}
 return out+":"+IntStream.of(4).sum()+":"+LongStream.of(5,6).sum()+":"+DoubleStream.of(7,8).sum();
 }
 public static void main(String[] args){System.out.print(run());}
}`
	want := stringCaseReferenceJavaOracle(t, "PrimitiveStreamArrayContract", source)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>primitive-array</artifactId><version>1</version></project>`,
		"src/main/java/PrimitiveStreamArrayContract.java": source,
	}, "PrimitiveStreamArrayContract", want)
}

func TestPrimitiveStreamArrayShadowsStrictJDK21Parity(t *testing.T) {
	const source = `class IntStream {static IntStream of(int[] values){return new IntStream();}int sum(){return 17;}}
class LongStream {static LongStream of(long[] values){return new LongStream();}long sum(){return 18L;}}
class DoubleStream {static DoubleStream of(double[] values){return new DoubleStream();}double sum(){return 19.0;}}
public class PrimitiveStreamArrayShadows {
 static <IntStream> int binder(IntStream ignored){return java.util.stream.IntStream.of(new int[]{4,5}).sum();}
 public static String run(){return IntStream.of(new int[]{1}).sum()+":"+LongStream.of(new long[]{2L}).sum()+":"+DoubleStream.of(new double[]{3.0}).sum()+":"+binder("bound")+":"+java.util.stream.LongStream.of(new long[]{6L,7L}).sum()+":"+java.util.stream.DoubleStream.of(new double[]{8.0,9.0}).sum();}
 public static void main(String[] args){System.out.print(run());}
}`
	want := stringCaseReferenceJavaOracle(t, "PrimitiveStreamArrayShadows", source)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>primitive-array-shadows</artifactId><version>1</version></project>`,
		"src/main/java/PrimitiveStreamArrayShadows.java": source,
	}, "PrimitiveStreamArrayShadows", want)
}
