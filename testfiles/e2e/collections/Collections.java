import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Collections {
    public static void main(String[] args) {
        List<String> list = new ArrayList<String>();
        list.add("a");
        list.add("b");
        list.add("c");
        System.out.println(list.size());
        System.out.println(list.get(1));
        for (String s : list) {
            System.out.println("item " + s);
        }

        Map<String, Integer> map = new HashMap<String, Integer>();
        map.put("one", 1);
        map.put("two", 2);
        System.out.println(map.get("one"));
        System.out.println(map.containsKey("two"));
        System.out.println(map.size());
    }
}
