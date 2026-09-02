package transpiler

import "testing"

func TestNumericStreams_Sources(t *testing.T) {
	src := `
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.Stream;
public class SourceProgram {
    public static void run() {
        int a = IntStream.range(0, 5).sum();
        int b = IntStream.rangeClosed(1, 5).sum();
        long c = LongStream.range(0, 5).sum();
        int d = IntStream.of(3, 1, 2).sum();
        Stream<String> joined = Stream.concat(Stream.of("a"), Stream.of("b"));
        Stream<String> nothing = Stream.empty();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.IntStreamRange(0, 5)")
	assertContains(t, out, "stdjava.IntStreamRangeClosed(1, 5)")
	assertContains(t, out, "stdjava.LongStreamRange(0, 5)")
	// The element type is spelled out so untyped constants do not infer a
	// host-sized Go int.
	assertContains(t, out, "stdjava.NewStream[int32](3, 1, 2)")
	assertContains(t, out, "stdjava.StreamConcat(")
	assertContains(t, out, "stdjava.StreamEmpty[string]()")
}

func TestNumericStreams_ArraysStreamPassesTheComponentType(t *testing.T) {
	src := `
import java.util.Arrays;
public class ArraysStreamProgram {
    public static void run() {
        int[] numbers = new int[3];
        String[] names = new String[2];
        int total = Arrays.stream(numbers).sum();
        long named = Arrays.stream(names).count();
    }
}
`
	out := renderGoFileFromJava(t, src)
	// A reference array erases its elements to any at runtime, so the component
	// type has to be passed explicitly.
	assertContains(t, out, "stdjava.StreamOfArray[int32](numbers)")
	assertContains(t, out, "stdjava.StreamOfArray[string](names)")
}

func TestNumericStreams_Conversions(t *testing.T) {
	src := `
import java.util.ArrayList;
import java.util.List;
import java.util.stream.IntStream;
public class ConversionProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        int lengths = words.stream().mapToInt(w -> w.length()).sum();
        long wide = words.stream().mapToLong(w -> w.length()).sum();
        long boxedCount = IntStream.rangeClosed(1, 3).boxed().count();
        long asLong = IntStream.rangeClosed(1, 3).asLongStream().sum();
        double asDouble = IntStream.rangeClosed(1, 3).asDoubleStream().sum();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamBoxed(")
	assertContains(t, out, "stdjava.StreamAsLongStream(")
	assertContains(t, out, "stdjava.StreamAsDoubleStream(")
	// mapToLong pins the closure to int64, so a body of type int32 needs the
	// widening conversion Java applies implicitly.
	assertContains(t, out, "func(w string) int64")
	assertContains(t, out, "return int64(stdjava.StringLength(")
}

// A primitive stream's terminal returns an OptionalInt/OptionalDouble, whose
// accessors differ from Optional's but map onto the same runtime value.
func TestNumericStreams_PrimitiveOptionalAccessors(t *testing.T) {
	src := `
import java.util.OptionalDouble;
import java.util.stream.IntStream;
public class PrimitiveOptionalProgram {
    public static void run() {
        int smallest = IntStream.of(5, 2).min().getAsInt();
        OptionalDouble mean = IntStream.of(5, 2).average();
        double value = mean.getAsDouble();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamMin(")
	assertContains(t, out, "stdjava.StreamAverage(")
	assertContains(t, out, ".Get()")
	assertNotContains(t, out, ".getAsInt()")
	assertNotContains(t, out, ".getAsDouble()")
}

// String.chars returns an IntStream in Java, so stream operations must chain
// onto it.
func TestNumericStreams_StringCharsIsAStream(t *testing.T) {
	src := `
public class CharsProgram {
    public static void run() {
        long count = "hello".chars().count();
        int total = "abc".chars().sum();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StringCharsStream(")
	assertContains(t, out, ".Count()")
	assertContains(t, out, "stdjava.StreamSum(")
}

func TestNumericStreams_SummaryStatistics(t *testing.T) {
	src := `
import java.util.IntSummaryStatistics;
import java.util.stream.IntStream;
public class StatsProgram {
    public static void run() {
        IntSummaryStatistics stats = IntStream.rangeClosed(1, 4).summaryStatistics();
        long count = stats.getCount();
        int min = stats.getMin();
        double average = stats.getAverage();
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "stdjava.StreamSummaryStatistics(")
	assertContains(t, out, ".GetCount()")
	assertContains(t, out, ".GetMin()")
	assertContains(t, out, ".GetAverage()")
}
