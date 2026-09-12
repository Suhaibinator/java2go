package parity.collections;

import java.util.*;
import java.util.stream.*;
import java.util.concurrent.ConcurrentHashMap;

public class CollectionContracts {
    static class Key implements Comparable<Key> {
        int id;
        String label;
        Key(int id, String label) { this.id = id; this.label = label; }
        public boolean equals(Object other) {
            return other instanceof Key && id == ((Key) other).id;
        }
        public int hashCode() { return 7; }
        public int compareTo(Key other) { return Integer.compare(id, other.id); }
    }
    static class Identity { int id = 1; }

    public static String run() {
        Key first = new Key(1, "first");
        Key equal = new Key(1, "replacement");
        Key collision = new Key(2, "second");
        Map<Key, String> map = new LinkedHashMap<>();
        map.put(first, "old");
        String previous = map.put(equal, "new");
        map.put(collision, "collision");
        String result = previous + ":" + map.size() + ":" + map.get(first)
            + ":" + map.get(new Key(2, "lookup"));
        boolean retained = false;
        for (Key key : map.keySet()) { if (key == first) retained = true; }
        result += ":" + retained;
        result += ":" + map.remove(equal) + ":" + map.size() + ":" + map.containsKey(collision);
        Set<Key> set = new HashSet<>();
        result += ":" + set.add(first) + ":" + set.add(equal) + ":" + set.add(collision);
        result += ":" + set.contains(equal) + ":" + set.remove(equal) + ":" + set.size();
        List<Key> list = new ArrayList<>();
        list.add(first);
        result += ":" + list.contains(equal) + ":" + list.indexOf(equal) + ":" + list.remove(equal);
        List<Identity> identities = new ArrayList<>();
        identities.add(new Identity());
        result += ":" + identities.contains(new Identity());
        List<Key> events = Arrays.asList(first, equal, collision);
        result += ":" + events.stream().distinct().count();
        result += ":" + events.stream().collect(Collectors.toSet()).size();
        Map<Key, Long> counts = events.stream().collect(Collectors.groupingBy(k -> k, Collectors.counting()));
        result += ":" + counts.size() + ":" + counts.get(equal);
        Map<Key, String> merged = events.stream().collect(Collectors.toMap(k -> k, k -> k.label, (a, b) -> a + "+" + b));
        result += ":" + merged.size() + ":" + merged.get(equal);
        TreeSet<Integer> sorted = new TreeSet<>();
        sorted.add(3); sorted.add(1); sorted.add(2);
        result += ":" + sorted;
        ConcurrentHashMap<Key, String> concurrent = new ConcurrentHashMap<>();
        concurrent.put(first, "old");
        result += ":" + concurrent.put(equal, "new");
        concurrent.put(collision, "collision");
        result += ":" + concurrent.size() + ":" + concurrent.get(new Key(1, "lookup"));
        result += ":" + concurrent.remove(equal) + ":" + concurrent.containsKey(collision);
        return result;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
