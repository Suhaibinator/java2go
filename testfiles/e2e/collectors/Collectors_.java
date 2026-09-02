import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

public class Collectors_ {
    // Java's groupingBy, toMap and toSet return HashMap/HashSet, whose iteration
    // order is unspecified. The generated runtime is insertion-ordered, so it is
    // MORE deterministic than Java — printing one directly would compare against
    // output the JVM does not guarantee. Every such result is therefore rendered
    // through an explicitly sorted key list.
    public static void main(String[] args) {
        List<String> words = new ArrayList<String>();
        words.add("fig");
        words.add("pear");
        words.add("kiwi");
        words.add("plum");
        words.add("date");

        // toList keeps encounter order, so it prints directly.
        System.out.println(words.stream().filter(w -> w.length() == 4).collect(Collectors.toList()));

        // toSet has no order guarantee in Java; sort before printing.
        Set<Integer> lengths = words.stream().map(w -> w.length()).collect(Collectors.toSet());
        List<Integer> sortedLengths = new ArrayList<Integer>();
        for (Integer length : lengths) {
            sortedLengths.add(length);
        }
        Collections.sort(sortedLengths);
        System.out.println(sortedLengths);

        // joining, all three arities.
        System.out.println(words.stream().collect(Collectors.joining()));
        System.out.println(words.stream().collect(Collectors.joining(", ")));
        System.out.println(words.stream().collect(Collectors.joining(", ", "[", "]")));

        // counting, summing, averaging.
        System.out.println("count " + words.stream().collect(Collectors.counting()));
        System.out.println("sumInt " + words.stream().collect(Collectors.summingInt(w -> w.length())));
        System.out.println("sumLong " + words.stream().collect(Collectors.summingLong(w -> w.length())));
        System.out.println("avg " + words.stream().collect(Collectors.averagingInt(w -> w.length())));

        // toMap.
        Map<String, Integer> byWord = words.stream()
                .collect(Collectors.toMap(w -> w, w -> w.length()));
        List<String> wordKeys = new ArrayList<String>();
        for (String key : byWord.keySet()) {
            wordKeys.add(key);
        }
        Collections.sort(wordKeys);
        for (int i = 0; i < wordKeys.size(); i++) {
            System.out.println("word " + wordKeys.get(i) + "=" + byWord.get(wordKeys.get(i)));
        }

        // Duplicate keys are resolved by the merge function.
        Map<Integer, String> joinedByLength = words.stream()
                .collect(Collectors.toMap(w -> w.length(), w -> w, (a, b) -> a + "|" + b));
        List<Integer> joinedKeys = new ArrayList<Integer>();
        for (Integer key : joinedByLength.keySet()) {
            joinedKeys.add(key);
        }
        Collections.sort(joinedKeys);
        for (int i = 0; i < joinedKeys.size(); i++) {
            System.out.println("merged " + joinedKeys.get(i) + " " + joinedByLength.get(joinedKeys.get(i)));
        }

        // groupingBy, plain.
        Map<Integer, List<String>> byLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length()));
        List<Integer> lengthKeys = new ArrayList<Integer>();
        for (Integer key : byLength.keySet()) {
            lengthKeys.add(key);
        }
        Collections.sort(lengthKeys);
        for (int i = 0; i < lengthKeys.size(); i++) {
            System.out.println("group " + lengthKeys.get(i) + " " + byLength.get(lengthKeys.get(i)));
        }

        // groupingBy with a downstream collector.
        Map<Integer, Long> countByLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length(), Collectors.counting()));
        for (int i = 0; i < lengthKeys.size(); i++) {
            System.out.println("count " + lengthKeys.get(i) + " " + countByLength.get(lengthKeys.get(i)));
        }

        Map<Integer, String> namesByLength = words.stream()
                .collect(Collectors.groupingBy(w -> w.length(), Collectors.joining("/")));
        for (int i = 0; i < lengthKeys.size(); i++) {
            System.out.println("joined " + lengthKeys.get(i) + " " + namesByLength.get(lengthKeys.get(i)));
        }

        // mapping as a downstream collector.
        Map<Integer, List<String>> initialsByLength = words.stream()
                .collect(Collectors.groupingBy(
                        w -> w.length(),
                        Collectors.mapping(w -> w.substring(0, 1), Collectors.toList())));
        for (int i = 0; i < lengthKeys.size(); i++) {
            System.out.println("initials " + lengthKeys.get(i) + " " + initialsByLength.get(lengthKeys.get(i)));
        }

        // partitioningBy always returns both entries.
        Map<Boolean, List<String>> partitioned = words.stream()
                .collect(Collectors.partitioningBy(w -> w.length() > 3));
        System.out.println("false=" + partitioned.get(false) + " true=" + partitioned.get(true));

        Map<Boolean, Long> partitionCounts = words.stream()
                .collect(Collectors.partitioningBy(w -> w.length() > 3, Collectors.counting()));
        System.out.println("falseCount=" + partitionCounts.get(false) + " trueCount=" + partitionCounts.get(true));

        // Three-argument reduce: the accumulator is (U, T) -> U and the combiner
        // is (U, U) -> U, with U unrelated to the element type.
        String concatenated = words.stream().reduce("", (acc, w) -> acc + w.charAt(0), (a, b) -> a + b);
        System.out.println("reduced " + concatenated);
    }
}
