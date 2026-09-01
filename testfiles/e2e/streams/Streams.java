import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class Streams {
    public static void main(String[] args) {
        List<Integer> nums = new ArrayList<Integer>();
        for (int i = 1; i <= 6; i++) {
            nums.add(i);
        }

        // filter evens, map *2, collect to list
        List<Integer> doubled = nums.stream()
                .filter(n -> n % 2 == 0)
                .map(n -> n * 2)
                .collect(Collectors.toList());
        System.out.println(doubled);

        // map type-changing Integer -> String, forEach
        nums.stream()
                .filter(n -> n <= 3)
                .map(n -> "n" + n)
                .forEach(s -> System.out.println(s));

        // reduce sum
        int sum = nums.stream().reduce(0, (a, b) -> a + b);
        System.out.println("sum " + sum);

        // count
        long evenCount = nums.stream().filter(n -> n % 2 == 0).count();
        System.out.println("evens " + evenCount);

        // anyMatch / allMatch
        System.out.println(nums.stream().anyMatch(n -> n > 5));
        System.out.println(nums.stream().allMatch(n -> n > 0));

        // Stream.of, sorted, limit
        List<Integer> top3 = Stream.of(5, 3, 9, 1, 7, 2)
                .sorted()
                .limit(3)
                .collect(Collectors.toList());
        System.out.println(top3);
    }
}
