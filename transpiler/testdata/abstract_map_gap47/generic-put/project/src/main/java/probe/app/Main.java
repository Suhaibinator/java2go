package probe.app;
import probe.base.Base;
import probe.impl.StringMap;
import java.util.AbstractMap;
import java.util.Map;
import java.util.Set;
public final class Main {
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        String key = "key-" + seed;
        StringMap child = new StringMap();
        Base<String,String> base = child;
        AbstractMap<String,String> platform = child;
        Map<String,String> contract = child;
        Set<String> keys = base.keySet();
        System.out.println("seed=" + seed);
        System.out.println("child=" + child.put(key, "one"));
        System.out.println("base=" + base.put(key, "two"));
        System.out.println("platform=" + platform.put(key, "three"));
        System.out.println("contract=" + contract.put(key, "four"));
        System.out.println("inherited=" + base.inheritedPut(key, "five"));
        System.out.println("value=" + platform.get(key) + ";calls=" + child.putCalls());
        Map raw = child;
        try { raw.put(Integer.valueOf(seed), "bad-key"); System.out.println("raw-key=accepted"); }
        catch (ClassCastException expected) { System.out.println("raw-key=CCE"); }
        try { raw.put(key, Integer.valueOf(seed)); System.out.println("raw-value=accepted"); }
        catch (ClassCastException expected) { System.out.println("raw-value=CCE"); }
        System.out.println("after-raw=" + child.putCalls() + ";size=" + base.size() + ";value=" + base.get(key));
        try { child.superPut("super-" + seed, "unwritten"); System.out.println("super=accepted"); }
        catch (UnsupportedOperationException expected) { System.out.println("super=UOE"); }
        System.out.println("after-super=" + child.putCalls() + ";size=" + contract.size());
        System.out.println("cache=" + (keys == child.keySet()) + ";live=" + keys.contains(key));
        System.out.println("null-key=" + platform.put(null, "nil"));
        System.out.println("keys-remove=" + keys.remove(key) + ";size=" + child.size() + ";null=" + base.get(null));
        base.clear();
        System.out.println("cleared=" + keys.isEmpty() + ";calls=" + child.putCalls());
    }
}
