package probe.app;
import java.util.AbstractMap;
import java.io.Serializable;
import probe.entry.Trace;
import probe.store.IntegerMap;
public final class Main {
    private static int consumers;
    private static void accept(Serializable value) { consumers++; }
    private static void state(String label, IntegerMap map) {
        System.out.println(label + ";size=" + map.size() + ";events=" + Trace.take());
    }
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        String key = "key-" + seed;
        IntegerMap map = new IntegerMap();
        AbstractMap<String,Integer> base = map;
        Object marker = new Object();
        map.seed(key, marker);
        Trace.take();
        System.out.println("seed=" + seed);
        Object broad = base.get(key);
        System.out.println("object=" + (broad == marker));
        state("after-object", map);
        try { Comparable<?> comparable = base.get(key); System.out.println("plain-comparable=accepted"); }
        catch (ClassCastException expected) { System.out.println("plain-comparable=CCE"); }
        state("after-plain-comparable", map);
        try { Serializable serial = base.get(key); System.out.println("plain-serializable=accepted"); }
        catch (ClassCastException expected) { System.out.println("plain-serializable=CCE"); }
        state("after-plain-serializable", map);
        try { accept(base.get(key)); System.out.println("plain-argument=accepted"); }
        catch (ClassCastException expected) { System.out.println("plain-argument=CCE"); }
        state("after-plain-argument;consumers=" + consumers, map);
        base.clear();
        map.seed(key, null);
        Trace.take();
        Comparable<?> nullComparable = base.get(key);
        System.out.println("null-comparable=" + (nullComparable == null));
        state("after-null-comparable", map);
        Serializable nullSerializable = base.get(key);
        System.out.println("null-serializable=" + (nullSerializable == null));
        state("after-null-serializable", map);
        accept(base.get(key));
        state("after-null-argument;consumers=" + consumers, map);
        base.clear();
        Double genuine = Double.valueOf(seed + 0.5);
        map.seed(key, genuine);
        Trace.take();
        Comparable<?> genuineComparable = base.get(key);
        System.out.println("genuine-comparable=" + (genuineComparable == genuine));
        state("after-genuine-comparable", map);
        Serializable genuineSerializable = base.get(key);
        System.out.println("genuine-serializable=" + (genuineSerializable == genuine));
        state("after-genuine-serializable", map);
        accept(base.get(key));
        state("after-genuine-argument;consumers=" + consumers, map);
        Object restoredBroad = base.get(key);
        System.out.println("genuine-object=" + (restoredBroad == genuine));
        state("after-genuine-object", map);
    }
}
