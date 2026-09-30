package transpiler

import "testing"

// These cases all passed the substring-asserting integration tests while
// generating Go that did not compile. They are pinned here through an actual
// build against the stdjava runtime.

func TestCompiles_ComparatorComparingFamily(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
public class ComparingProgram {
    static class Person {
        int age;
        String name;
        int getAge() { return age; }
        String getName() { return name; }
    }
    public static void run() {
        List<Person> people = new ArrayList<Person>();
        people.sort(Comparator.comparing((Person p) -> p.getName()));
        people.sort(Comparator.comparingInt((Person p) -> p.getAge()));
    }
}
`)
}

// An argument-free comparator factory has nothing to infer its element type
// from, so it must be spelled out from the call it is an argument to.
func TestCompiles_ComparatorOrderFactoriesAsArguments(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.List;
public class OrderFactoryProgram {
    public static void run() {
        List<Integer> nums = new ArrayList<Integer>();
        Collections.sort(nums, Comparator.naturalOrder());
        Collections.sort(nums, Comparator.reverseOrder());
        nums.sort(Comparator.naturalOrder());
    }
}
`)
}

// A call chained onto a Comparator-returning expression must resolve to the
// generated Go method name, not keep its Java spelling.
func TestCompiles_ChainedComparatorCalls(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.Comparator;
public class ChainedComparatorProgram {
    public static void run() {
        Comparator<Integer> ascending = (a, b) -> a - b;
        int result = ascending.reversed().compare(1, 2);
    }
}
`)
}

// Java widens implicitly at a ToLongFunction's return; Go does not, and the
// conversion has to reach returns nested inside control flow too.
func TestCompiles_NestedReturnInPrimitiveMapper(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.stream.IntStream;
public class NestedReturnProgram {
    public static void run() {
        long total = IntStream.range(0, 5).mapToLong(x -> {
            if (x > 2) {
                return x * 2;
            }
            return x;
        }).sum();
    }
}
`)
}

// summaryStatistics has to carry a result type for a chained accessor, and its
// long and double forms are distinct Java classes.
func TestCompiles_SummaryStatisticsSpellings(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.IntSummaryStatistics;
import java.util.stream.IntStream;
public class StatsSpellingProgram {
    public static void run() {
        long chained = IntStream.rangeClosed(1, 4).summaryStatistics().getSum();
        IntSummaryStatistics declared = IntStream.rangeClosed(1, 4).summaryStatistics();
        int smallest = declared.getMin();
        double average = declared.getAverage();
    }
}
`)
}

// The surface covered by the byte-exact e2e fixtures must also compile in
// isolation, so a fixture rewrite cannot quietly stop exercising it.
func TestCompiles_StreamAndOptionalSurface(t *testing.T) {
	assertGeneratedCompiles(t, `
import java.util.ArrayList;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.Stream;
public class SurfaceProgram {
    public static void run() {
        List<Integer> nums = new ArrayList<Integer>();
        List<Integer> distinct = nums.stream().distinct().skip(1).collect(Collectors.toList());
        int first = nums.stream().findFirst().get();
        int smallest = nums.stream().min((a, b) -> a - b).get();
        int folded = nums.stream().reduce((a, b) -> a + b).get();
        List<String> flat = nums.stream().flatMap(n -> Stream.of("a", "b")).collect(Collectors.toList());
        long ranged = IntStream.rangeClosed(1, 3).asLongStream().sum();

        Optional<String> name = Optional.of("java");
        String viaFlatMap = name.flatMap(s -> Optional.of(s + "!")).get();
        String fallback = name.orElseGet(() -> "none");
        String required = name.orElseThrow();
    }
}
`)
}
