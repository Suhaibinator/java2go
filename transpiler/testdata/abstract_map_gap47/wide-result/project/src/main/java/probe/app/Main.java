package probe.app;
import java.util.AbstractMap;
import probe.entry.Trace;
import probe.store.IntegerMap;
public final class Main {
    private static int consumers;
    private static Number numberReturn(AbstractMap<String,Integer> map, String key) { return map.get(key); }
    private static void acceptNumber(Number value) { consumers++; System.out.println("argument=" + value); }
    private static void state(String label, IntegerMap map) {
        System.out.println(label + ";size=" + map.size() + ";events=" + Trace.take());
    }
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        String key = "key-" + seed;
        Double value = Double.valueOf(seed + 0.5);
        IntegerMap map = new IntegerMap();
        AbstractMap<String,Integer> base = map;
        map.seed(key, value);
        Trace.take();
        System.out.println("seed=" + seed);
        Object broad = base.get(key);
        System.out.println("object=" + (broad == value));
        state("after-object", map);
        base.get(key);
        state("after-discard", map);
        Number assigned = base.get(key);
        System.out.println("number=" + assigned + ";identity=" + (assigned == value));
        state("after-number", map);
        acceptNumber(base.get(key));
        state("after-argument;consumers=" + consumers, map);
        Number returned = numberReturn(base, key);
        System.out.println("return=" + returned);
        state("after-return", map);
        Number cast = (Number) base.get(key);
        System.out.println("explicit=" + cast);
        state("after-explicit", map);
        try { Integer narrow = base.get(key); System.out.println("integer=" + narrow); }
        catch (ClassCastException expected) { System.out.println("integer=CCE"); }
        state("after-integer", map);
        try { Integer removed = base.remove(key); System.out.println("remove-integer=" + removed); }
        catch (ClassCastException expected) { System.out.println("remove-integer=CCE"); }
        state("after-remove-integer", map);
        map.seed(key, value);
        Trace.take();
        Number removed = base.remove(key);
        System.out.println("remove-number=" + removed + ";identity=" + (removed == value));
        state("after-remove-number", map);
        map.seed(key, Integer.valueOf(seed));
        Trace.take();
        Integer restored = base.get(key);
        System.out.println("restored=" + restored);
        state("after-restored", map);
        base.clear();
        state("after-clear", map);
    }
}
