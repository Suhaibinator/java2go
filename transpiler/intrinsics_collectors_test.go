package transpiler

import "testing"

// Every collector is compile-checked: the lowering picks a runtime function
// from the collector's identity and types its lambdas by hand, so a substring
// assertion would not prove the result is well typed.

func TestCollectors_ToListAndToSet(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;
public class ToListSetProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        List<String> asList = words.stream().collect(Collectors.toList());
        Set<String> asSet = words.stream().collect(Collectors.toSet());
        List<String> unmodifiable = words.stream().collect(Collectors.toUnmodifiableList());
    }
}
`)
	assertContains(t, out, ".ToList()")
	assertContains(t, out, "stdjava.StreamToSet(")
}

func TestCollectors_Joining(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;
public class JoiningProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        String plain = words.stream().collect(Collectors.joining());
        String separated = words.stream().collect(Collectors.joining(", "));
        String wrapped = words.stream().collect(Collectors.joining(", ", "[", "]"));
    }
}
`)
	assertContains(t, out, "stdjava.StreamJoining(")
}

func TestCollectors_CountingSummingAveraging(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;
public class NumericCollectorProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        long count = words.stream().collect(Collectors.counting());
        int summed = words.stream().collect(Collectors.summingInt(w -> w.length()));
        long summedLong = words.stream().collect(Collectors.summingLong(w -> w.length()));
        double averaged = words.stream().collect(Collectors.averagingInt(w -> w.length()));
    }
}
`)
	assertContains(t, out, "stdjava.StreamCounting(")
	assertContains(t, out, "stdjava.StreamSummingOf(")
	assertContains(t, out, "stdjava.StreamAveragingOf(")
}

// toMap's three lambdas have three different signatures, which is what the
// per-argument typing exists for.
func TestCollectors_ToMap(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
public class ToMapProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        Map<String, Integer> lengths = words.stream()
                .collect(Collectors.toMap(w -> w, w -> w.length()));
        Map<String, Integer> merged = words.stream()
                .collect(Collectors.toMap(w -> w, w -> w.length(), (a, b) -> a + b));
    }
}
`)
	assertContains(t, out, "stdjava.StreamToMap(")
	assertContains(t, out, "stdjava.StreamToMapMerging(")
	// The merge function takes two values, not two elements.
	assertContains(t, out, "func(a int32, b int32) int32")
}

func TestCollectors_GroupingByAndPartitioningBy(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
public class GroupingProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        Map<Integer, List<String>> byLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length()));
        Map<Boolean, List<String>> partitioned = words.stream()
                .collect(Collectors.partitioningBy(w -> w.length() > 3));
    }
}
`)
	assertContains(t, out, "stdjava.StreamGroupingBy(")
	assertContains(t, out, "stdjava.StreamPartitioningBy(")
}

// A downstream collector lowers to a function applied to each group's own
// stream, so the forms nest.
func TestCollectors_DownstreamCollectors(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
public class DownstreamProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        Map<Integer, Long> countByLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length(), Collectors.counting()));
        Map<Integer, String> joinedByLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length(), Collectors.joining("/")));
        Map<Boolean, Long> partitionCounts = words.stream()
                .collect(Collectors.partitioningBy(w -> w.length() > 3, Collectors.counting()));
    }
}
`)
	assertContains(t, out, "stdjava.StreamGroupingByDownstream(")
	assertContains(t, out, "stdjava.StreamPartitioningByDownstream(")
	assertContains(t, out, "func(__java2goGroup stdjava.Stream[string])")
}

// mapping transforms elements before the nested collector sees them.
func TestCollectors_Mapping(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
public class MappingProgram {
    public static void run() {
        List<String> words = new ArrayList<String>();
        Map<Integer, List<String>> firstLetters = words.stream()
                .collect(Collectors.groupingBy(
                        w -> w.length(),
                        Collectors.mapping(w -> w.substring(0, 1), Collectors.toList())));
    }
}
`)
	assertContains(t, out, "stdjava.StreamMap(")
	assertContains(t, out, "stdjava.StreamGroupingByDownstream(")
}

// K18: the three-argument reduce, whose accumulator is (U, T) -> U and whose
// combiner is (U, U) -> U, with U unrelated to the element type.
func TestCollectors_ThreeArgumentReduce(t *testing.T) {
	out := assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
public class ThreeArgReduceProgram {
    public static void run() {
        List<Integer> nums = new ArrayList<Integer>();
        String joined = nums.stream().reduce("", (acc, x) -> acc + x, (a, b) -> a + b);
        int total = nums.stream().reduce(0, (a, b) -> a + b);
    }
}
`)
	assertContains(t, out, "stdjava.StreamReduceCombining(")
	// The accumulator takes (U, T); the combiner takes (U, U).
	assertContains(t, out, "func(acc string, x int32) string")
	assertContains(t, out, "func(a string, b string) string")
}
