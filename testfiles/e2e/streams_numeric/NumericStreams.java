import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.OptionalDouble;
import java.util.stream.Collectors;
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.Stream;

public class NumericStreams {
    public static void main(String[] args) {
        // range is half-open, rangeClosed includes the bound.
        System.out.println(IntStream.range(0, 5).sum());
        System.out.println(IntStream.rangeClosed(1, 5).sum());
        System.out.println(IntStream.range(5, 5).count());
        System.out.println(IntStream.range(5, 1).count());

        // Pipeline operations apply to a primitive stream unchanged.
        System.out.println(IntStream.rangeClosed(1, 10).filter(n -> n % 2 == 0).sum());
        System.out.println(IntStream.rangeClosed(1, 4).map(n -> n * n).boxed().collect(Collectors.toList()));

        // IntStream.of and LongStream.
        System.out.println(IntStream.of(3, 1, 2).sorted().boxed().collect(Collectors.toList()));
        System.out.println(LongStream.rangeClosed(1, 5).sum());

        // average returns an OptionalDouble.
        OptionalDouble mean = IntStream.rangeClosed(1, 4).average();
        System.out.println("avg " + mean.getAsDouble());
        System.out.println("emptyAvg " + IntStream.range(0, 0).average().isPresent());

        // min / max with no comparator.
        System.out.println("min " + IntStream.of(5, 2, 9).min().getAsInt());
        System.out.println("max " + IntStream.of(5, 2, 9).max().getAsInt());

        // mapToObj leaves the element type free; mapToInt pins it.
        List<String> labels = IntStream.rangeClosed(1, 3).mapToObj(n -> "n" + n).collect(Collectors.toList());
        System.out.println(labels);

        List<String> words = new ArrayList<String>();
        words.add("a");
        words.add("bbb");
        words.add("cc");
        System.out.println(words.stream().mapToInt(w -> w.length()).sum());
        System.out.println(words.stream().mapToLong(w -> w.length()).sum());

        // Widening conversions.
        System.out.println(IntStream.rangeClosed(1, 3).asLongStream().sum());
        System.out.println(IntStream.rangeClosed(1, 3).asDoubleStream().sum());

        // Arrays.stream over a primitive and a reference array.
        int[] numbers = new int[4];
        numbers[0] = 4;
        numbers[1] = 1;
        numbers[2] = 3;
        numbers[3] = 2;
        System.out.println(Arrays.stream(numbers).sum());
        System.out.println(Arrays.stream(numbers).sorted().boxed().collect(Collectors.toList()));

        String[] names = new String[3];
        names[0] = "pear";
        names[1] = "fig";
        names[2] = "kiwi";
        System.out.println(Arrays.stream(names).filter(s -> s.length() == 3).collect(Collectors.toList()));

        // Stream.concat keeps the element type of its arguments.
        System.out.println(Stream.concat(Stream.of("a", "b"), Stream.of("c")).collect(Collectors.toList()));

        // Stream.empty takes its element type from the assignment target.
        Stream<String> nothing = Stream.empty();
        System.out.println("empty " + nothing.count());

        // String.chars is an IntStream.
        System.out.println("chars " + "hello".chars().count());
        System.out.println("sumChars " + "abc".chars().sum());
        System.out.println("upper " + "abc".chars().map(c -> c - 32).count());

        // summaryStatistics collects everything in one pass.
        java.util.IntSummaryStatistics stats = IntStream.rangeClosed(1, 4).summaryStatistics();
        System.out.println("count " + stats.getCount());
        System.out.println("sum " + stats.getSum());
        System.out.println("min " + stats.getMin());
        System.out.println("max " + stats.getMax());
        System.out.println("average " + stats.getAverage());
    }
}
