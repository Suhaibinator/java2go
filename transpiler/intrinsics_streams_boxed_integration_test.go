package transpiler

import "testing"

func TestStreamsBoxedReferenceAndPrimitiveCallbacks(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.Arrays;
import java.util.List;
import java.util.OptionalInt;
import java.util.OptionalLong;
import java.util.OptionalDouble;
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.DoubleStream;
public class BoxedStreamsProgram {
    public static String run() {
        List<Integer> cached = IntStream.of(127, 127).boxed().toList();
        List<Integer> fresh = IntStream.of(128, 128).boxed().toList();
        int total = IntStream.rangeClosed(1, 3).boxed().map(n -> n + 1).mapToInt(n -> n).sum();
        int primitiveTotal = IntStream.of(1, 2).map(n -> n + 1).sum();
        Integer objectResult = IntStream.of(1).mapToObj(n -> n + 10).findFirst().get();
        OptionalInt first = IntStream.of(3, 4).findFirst();
        Integer boxedFirst = first.getAsInt();
        OptionalLong smallest = LongStream.of(7L, 8L).min();
        OptionalDouble largest = DoubleStream.of(1.0, 2.0).max();
        Integer original = new Integer(500);
        Integer[] values = new Integer[]{null, original};
        Integer fromArray = Arrays.stream(values).filter(n -> n != null).findFirst().get();
        return (cached.get(0) == cached.get(1)) + ":" + (fresh.get(0) == fresh.get(1))
                + ":" + total + ":" + primitiveTotal + ":" + objectResult
                + ":" + first.getAsInt() + ":" + smallest.getAsLong()
                + ":" + largest.getAsDouble() + ":" + (fromArray == original)
                + ":" + (boxedFirst == Integer.valueOf(3));
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestBoxedStreams(t *testing.T) {
    if got := Run(); got != "true:false:9:5:11:3:7:2.0:true:true" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestCollectorsBoxedDownstreamResults(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import java.util.stream.Stream;
public class BoxedCollectorsProgram {
    public static String run() {
        List<String> words = Stream.of("a", "bb", "ccc").toList();
        Map<Integer, Long> counts = words.stream().collect(
                Collectors.groupingBy(w -> w.length() % 2, Collectors.counting()));
        Map<Integer, Integer> sums = words.stream().collect(
                Collectors.groupingBy(w -> w.length() % 2, Collectors.summingInt(w -> w.length())));
        Map<Integer, Double> averages = words.stream().collect(
                Collectors.groupingBy(w -> w.length() % 2, Collectors.averagingInt(w -> w.length())));
        Map<Boolean, Long> partitions = words.stream().collect(
                Collectors.partitioningBy(w -> w.length() > 1, Collectors.counting()));
        List<Integer> mapped = words.stream().collect(
                Collectors.mapping(w -> w.length(), Collectors.toList()));
        var inferredCount = words.stream().collect(Collectors.counting());
        int total = words.stream().collect(Collectors.summingInt(w -> w.length()));
        return counts.get(1) + ":" + sums.get(1) + ":" + averages.get(1)
                + ":" + partitions.get(true) + ":" + mapped.get(1)
                + ":" + inferredCount.longValue() + ":" + total;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestBoxedCollectors(t *testing.T) {
    if got := Run(); got != "2:4:2.0:2:2:3:6" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestStreamsSyntheticWrapperTypesIgnoreSourceShadow(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.Stream;
public class Integer {
    public static String run() {
        var boxed = IntStream.of(2).boxed().findFirst().get();
        var mapped = IntStream.of(3).mapToObj(n -> n + 1).findFirst().get();
        var summed = Stream.of("aa").collect(Collectors.summingInt(s -> s.length()));
        return boxed.intValue() + ":" + mapped.intValue() + ":" + summed.intValue();
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestSyntheticWrappers(t *testing.T) {
    if got := Run(); got != "2:4:2" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}

func TestStreamsBoxedMethodReferencesAndNullUnboxing(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.Stream;
public class BoxedStreamReferencesProgram {
    public static String run() {
        List<Integer> lengths = Stream.of("a", "bbb").map(String::length).toList();
        int total = lengths.stream().mapToInt(Integer::intValue).sum();
        List<Integer> copies = lengths.stream().map(Integer::intValue).toList();
        int collected = Stream.of("a", "bbb").collect(Collectors.summingInt(String::length));
        long mixed = Stream.of(1, 2L, 3).mapToLong(Number::longValue).sum();
        var inferredList = IntStream.of(1, 2).boxed().toList();
        Integer first = inferredList.get(0);
        String caught = "no";
        try {
            Stream.of((Integer) null).mapToInt(n -> n).sum();
        } catch (NullPointerException ex) {
            caught = "yes";
        }
        return total + ":" + copies.get(1) + ":" + collected + ":" + first + ":" + caught + ":" + mixed;
    }
}
`)
	runGeneratedWithStdjava(t, out, `
package main
import "testing"
func TestBoxedStreamReferences(t *testing.T) {
    if got := Run(); got != "4:3:4:1:yes:6" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}
