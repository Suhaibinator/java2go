package transpiler

import "testing"

func TestStreams_PipelineDispatch(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
import java.util.stream.Collectors;
public class StreamProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        List<Integer> out = xs.stream().filter(n -> n > 0).map(n -> n * 2).collect(Collectors.toList());
        long c = xs.stream().count();
        boolean any = xs.stream().anyMatch(n -> n == 1);
        int sum = xs.stream().reduce(0, (a, b) -> a + b);
        xs.stream().forEach(n -> System.out.println(n));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamOfSlice(xs.Slice())")
	// predicate -> bool result, mapper -> element-type result.
	assertContains(t, out, "Filter(func(n int32) bool")
	assertContains(t, out, "stdjava.StreamMap(")
	assertContains(t, out, "func(n int32) int32")
	assertContains(t, out, ".ToList()")
	assertContains(t, out, ".Count()")
	assertContains(t, out, "AnyMatch(func(n int32) bool")
	// reduce: two element-typed params, element-typed result.
	assertContains(t, out, "stdjava.StreamReduce(")
	assertContains(t, out, "func(a int32, b int32) int32")
	// forEach: void consumer, no result type.
	assertContains(t, out, "ForEach(func(n int32) {")
}

func TestStreams_StreamOfStatic(t *testing.T) {
	src := `
import java.util.stream.Stream;
public class StreamOfProgram {
    public static long run() {
        return Stream.of("a", "b", "c").count();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, `stdjava.NewStream("a", "b", "c")`)
	assertContains(t, out, ".Count()")
}

func TestStreams_SortedAndLimit(t *testing.T) {
	src := `
import java.util.List;
import java.util.ArrayList;
public class SortedProgram {
    public static void run() {
        List<Integer> xs = new ArrayList<Integer>();
        xs.stream().sorted().limit(2).forEach(n -> System.out.println(n));
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamSorted(stdjava.StreamOfSlice(xs.Slice()))")
	assertContains(t, out, ".Limit(2)")
}
