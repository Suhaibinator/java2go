package parity.collections;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

public class CollectionValues {
    static class Rank implements Comparable<Rank> {
        int rank;
        Rank(int rank) { this.rank = rank; }
        public int compareTo(Rank other) { return Integer.compare(rank, other.rank); }
    }
    public static String run() {
        List<Integer> one = Arrays.asList(1, 2);
        List<Integer> equal = Arrays.asList(1, 2);
        List<Integer> reverse = Arrays.asList(2, 1);
        Map<List<Integer>, String> index = new HashMap<>();
        index.put(one, "list");
        String result = one.equals(equal) + ":" + one.equals(reverse)
            + ":" + one.hashCode() + ":" + index.get(equal);
        Set<Integer> left = new HashSet<>(); left.add(1); left.add(2);
        Set<Integer> right = new HashSet<>(); right.add(2); right.add(1);
        result += ":" + left.equals(right) + ":" + left.hashCode() + ":" + left.equals(one);
        Map<String, Integer> a = new HashMap<>(); a.put("x", null);
        Map<String, Integer> b = new HashMap<>(); b.put("y", null);
        result += ":" + a.equals(b);
        b.clear(); b.put("x", null);
        result += ":" + a.equals(b) + ":" + a.hashCode();
        TreeSet<Rank> ranks = new TreeSet<>();
        result += ":" + ranks.add(new Rank(2)) + ":" + ranks.add(new Rank(1)) + ":" + ranks.add(new Rank(2));
        for (Rank rank : ranks) { result += ":" + rank.rank; }
        TreeMap<Integer, String> ordered = new TreeMap<>();
        ordered.put(3, "c"); ordered.put(1, "a"); ordered.put(2, "b");
        result += ":" + ordered;
        ConcurrentHashMap<String, Integer> concurrent = new ConcurrentHashMap<>();
        concurrent.put("x", 1);
        a.put("x", 1);
        result += ":" + concurrent.equals(a) + ":" + a.equals(concurrent) + ":" + concurrent.hashCode();
        return result;
    }
    public static void main(String[] args) { System.out.println(run()); }
}
