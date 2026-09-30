import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Optional;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class StreamOps {
    public static void main(String[] args) {
        List<Integer> nums = new ArrayList<Integer>();
        nums.add(5);
        nums.add(3);
        nums.add(9);
        nums.add(3);
        nums.add(1);
        nums.add(9);

        // distinct keeps the first occurrence of each element.
        System.out.println(nums.stream().distinct().collect(Collectors.toList()));

        // skip and limit compose in encounter order.
        System.out.println(nums.stream().skip(2).limit(3).collect(Collectors.toList()));
        System.out.println(nums.stream().skip(0).collect(Collectors.toList()));
        System.out.println(nums.stream().skip(99).collect(Collectors.toList()));

        // sorted() natural, and sorted(Comparator) descending.
        System.out.println(nums.stream().sorted().collect(Collectors.toList()));
        System.out.println(nums.stream().sorted((a, b) -> b - a).collect(Collectors.toList()));

        // min / max with a comparator return an Optional.
        Optional<Integer> smallest = nums.stream().min((a, b) -> a - b);
        Optional<Integer> largest = nums.stream().max((a, b) -> a - b);
        System.out.println("min " + smallest.get() + " max " + largest.get());

        // findFirst on a non-empty and an empty stream.
        System.out.println("first " + nums.stream().findFirst().get());
        Optional<Integer> none = nums.stream().filter(n -> n > 100).findFirst();
        System.out.println("emptyPresent " + none.isPresent());
        System.out.println("orElse " + none.orElse(-1));

        // reduce with no identity returns an Optional; with one it returns a value.
        Optional<Integer> sumOpt = nums.stream().reduce((a, b) -> a + b);
        System.out.println("reduceOptional " + sumOpt.get());
        System.out.println("reduceIdentity " + nums.stream().reduce(0, (a, b) -> a + b));

        List<Integer> empty = new ArrayList<Integer>();
        System.out.println("emptyReduce " + empty.stream().reduce((a, b) -> a + b).isPresent());

        // peek observes elements without changing them.
        List<Integer> seen = new ArrayList<Integer>();
        List<Integer> peeked = nums.stream().filter(n -> n > 3).peek(n -> seen.add(n)).collect(Collectors.toList());
        System.out.println("peeked " + peeked + " seen " + seen);

        // flatMap concatenates the mapped streams.
        List<String> words = new ArrayList<String>();
        words.add("ab");
        words.add("cd");
        List<String> letters = words.stream()
                .flatMap(w -> Stream.of(w.substring(0, 1), w.substring(1, 2)))
                .collect(Collectors.toList());
        System.out.println(letters);

        // parallelStream and parallel() run sequentially here; collect order is
        // still guaranteed by Java, so this stays deterministic.
        System.out.println(nums.parallelStream().sorted().collect(Collectors.toList()));
        System.out.println(nums.stream().parallel().sequential().count());

        // Optional's own operations.
        Optional<String> name = Optional.of("java");
        System.out.println(name.filter(s -> s.length() == 4).isPresent());
        System.out.println(name.filter(s -> s.length() == 9).isPresent());
        System.out.println(name.map(s -> s.length()).get());
        System.out.println(name.flatMap(s -> Optional.of(s + "2go")).get());
        System.out.println(name.orElseGet(() -> "fallback"));
        Optional<String> absent = Optional.empty();
        System.out.println(absent.orElseGet(() -> "fallback"));

        Optional<String> missing = Optional.empty();
        try {
            missing.orElseThrow();
        } catch (java.util.NoSuchElementException e) {
            System.out.println("threw NoSuchElementException");
        }
        try {
            missing.orElseThrow(() -> new IllegalStateException("nothing here"));
        } catch (IllegalStateException e) {
            System.out.println("threw " + e.getMessage());
        }

        // Comparator factory feeding a stream sort.
        List<String> names = new ArrayList<String>();
        names.add("pear");
        names.add("fig");
        names.add("banana");
        Comparator<String> byLength = (a, b) -> a.length() - b.length();
        System.out.println(names.stream().sorted(byLength).collect(Collectors.toList()));
        System.out.println(names.stream().sorted(byLength.reversed()).collect(Collectors.toList()));
    }
}
