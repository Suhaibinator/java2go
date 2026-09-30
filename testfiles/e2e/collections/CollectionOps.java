import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CollectionOps {
    public static void main(String[] args) {
        List<Integer> nums = new ArrayList<Integer>();
        for (int i = 1; i <= 5; i++) {
            nums.add(i * i);
        }
        System.out.println(nums.size());

        int sum = 0;
        for (int n : nums) {
            sum += n;
        }
        System.out.println("sum " + sum);
        System.out.println("first " + nums.get(0));
        System.out.println("last " + nums.get(nums.size() - 1));
        System.out.println(nums.contains(16));
        System.out.println(nums.indexOf(9));

        Map<String, Integer> counts = new HashMap<String, Integer>();
        String[] words = new String[] { "a", "b", "a", "c", "a", "b" };
        for (String w : words) {
            counts.put(w, counts.getOrDefault(w, 0) + 1);
        }
        System.out.println("a=" + counts.get("a"));
        System.out.println("b=" + counts.get("b"));
        System.out.println("c=" + counts.get("c"));
        System.out.println("keys " + counts.size());
    }
}
