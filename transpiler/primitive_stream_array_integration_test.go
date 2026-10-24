package transpiler

import "testing"

func TestPrimitiveStreamOfSelectsFamilyArray(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.util.stream.*; public class PrimitiveArraySources { static void run() { int[] ints={1,2};long[] longs={3L,4L};double[] doubles={5.0,6.0};int a=IntStream.of(ints).sum();long b=LongStream.of(longs).sum();double c=DoubleStream.of(doubles).sum();IntStream.of((int[])null);LongStream.of(null);DoubleStream.of(null);int d=IntStream.of(1,2).sum();long e=LongStream.of(3,4).sum();double f=DoubleStream.of(5,6).sum();} }`)
	assertContains(t, out, "stdjava.StreamOfArray[int32](ints)")
	assertContains(t, out, "stdjava.StreamOfArray[int64](longs)")
	assertContains(t, out, "stdjava.StreamOfArray[float64](doubles)")
	assertContains(t, out, "stdjava.StreamOfArray[int32](stdjava.JavaArrayCast[*stdjava.PrimitiveArray[int32]]")
	assertContains(t, out, "stdjava.StreamOfArray[int64](nil)")
	assertContains(t, out, "stdjava.StreamOfArray[float64](nil)")
	assertContains(t, out, "stdjava.NewStream[int32](1, 2)")
	assertContains(t, out, "stdjava.NewStream[int64](int64(3), int64(4))")
	assertContains(t, out, "stdjava.NewStream[float64](float64(5), float64(6))")
}
