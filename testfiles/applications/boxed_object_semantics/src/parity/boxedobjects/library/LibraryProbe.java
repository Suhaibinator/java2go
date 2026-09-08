package parity.boxedobjects.library;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.stream.Collectors;
import java.util.stream.IntStream;

public final class LibraryProbe {
    private LibraryProbe() {}

    public static String run() {
        Integer first = new Integer(1000);
        Integer equal = new Integer(1000);
        Map<Integer, Integer> values = new HashMap<Integer, Integer>();
        Integer missing = values.get(first);
        values.put(first, new Integer(300));
        Integer replaced = values.put(equal, new Integer(400));
        boolean retainedKey = false;
        for (Integer key : values.keySet()) {
            retainedKey = key == first;
        }
        String map = "map=" + values.size() + ":" + (missing == null)
                + ":" + replaced + ":" + values.get(equal) + ":" + retainedKey
                + ":" + values.containsKey(Long.valueOf(1000L))
                + ":" + values.containsValue(new Integer(400));
        values.put(null, null);
        map = map + ":" + values.containsKey(null) + ":" + (values.get(null) == null)
                + ":" + (values.get(new Integer(999)) == null)
                + ":" + (values.remove(Long.valueOf(1000L)) == null)
                + ":" + values.remove(equal);

        List<Integer> list = new ArrayList<Integer>();
        list.add(new Integer(65));
        list.add(new Integer(1000));
        list.add(equal);
        String lists = "list=" + list.contains(new Integer(65))
                + ":" + list.contains(Character.valueOf('A'))
                + ":" + list.contains(Long.valueOf(65L))
                + ":" + list.remove(new Integer(1000))
                + ":" + list.remove(0) + ":" + (list.get(0) == equal)
                + ":" + list.remove(Long.valueOf(1000L));

        Set<Integer> set = new HashSet<Integer>();
        boolean addedFirst = set.add(first);
        boolean addedEqual = set.add(equal);
        boolean retainedSetValue = false;
        for (Integer item : set) {
            retainedSetValue = item == first;
        }
        String sets = "set=" + addedFirst + ":" + addedEqual + ":" + set.size()
                + ":" + retainedSetValue + ":" + set.contains(equal)
                + ":" + set.contains(Long.valueOf(1000L))
                + ":" + set.remove(equal) + ":" + set.isEmpty();

        ConcurrentHashMap<Integer, Integer> concurrent = new ConcurrentHashMap<Integer, Integer>();
        Integer concurrentMissing = concurrent.get(first);
        concurrent.put(first, new Integer(500));
        Integer concurrentReplaced = concurrent.put(equal, new Integer(600));
        String concurrentMap = "concurrent=" + (concurrentMissing == null)
                + ":" + concurrentReplaced + ":" + concurrent.size()
                + ":" + concurrent.get(equal)
                + ":" + concurrent.containsKey(Long.valueOf(1000L))
                + ":" + concurrent.remove(new Integer(1000));

        List<Integer> numbers = IntStream.rangeClosed(1, 3).boxed().collect(Collectors.toList());
        int sum = numbers.stream().mapToInt(Integer::intValue).sum();
        Long count = numbers.stream().collect(Collectors.counting());
        Integer collectedSum = numbers.stream().collect(Collectors.summingInt(value -> value));
        Map<Integer, Long> grouped = numbers.stream()
                .collect(Collectors.groupingBy(value -> value % 2, Collectors.counting()));
        Map<Boolean, Long> partitions = numbers.stream()
                .collect(Collectors.partitioningBy(value -> value > 1, Collectors.counting()));
        List<Integer> duplicates = new ArrayList<Integer>();
        duplicates.add(first);
        duplicates.add(equal);
        List<Integer> distinct = duplicates.stream().distinct().collect(Collectors.toList());
        String streams = "stream=" + sum + ":" + count + ":" + collectedSum
                + ":" + grouped.get(0) + ":" + grouped.get(1)
                + ":" + partitions.get(Boolean.FALSE) + ":" + partitions.get(Boolean.TRUE)
                + ":" + distinct.size() + ":" + (distinct.get(0) == first);

        Integer absent = null;
        Optional<Integer> empty = Optional.ofNullable(absent);
        Optional<Integer> present = Optional.of(first);
        Optional<Integer> mappedEmpty = present.map(value -> absent);
        boolean rejectedNull = false;
        try {
            Optional.of(absent);
        } catch (NullPointerException expected) {
            rejectedNull = true;
        }
        String optionals = "optional=" + empty.isPresent() + ":" + present.isPresent()
                + ":" + (present.get() == first) + ":" + mappedEmpty.isPresent()
                + ":" + rejectedNull;
        return map + "/" + lists + "/" + sets + "/" + concurrentMap + "/" + streams + "/" + optionals;
    }
}
