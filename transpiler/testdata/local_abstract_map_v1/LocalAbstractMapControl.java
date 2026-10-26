package localmap;

import java.util.Map;
import java.util.Set;
import java.util.HashMap;

public class LocalAbstractMapControl {
    static final ThreadLocal<String> context = new ThreadLocal<>();
    static int gets, puts, entries;
    static boolean execution = true;
    static void check(Object lock) {
        execution &= Thread.currentThread() == thread && "context".equals(context.get()) && Thread.holdsLock(lock);
    }
    static Thread thread;
    static boolean nativeLocal() {
        class Local extends java.util.AbstractMap<String,String> {
            final Map<String,String> values = new HashMap<>();
            public String get(Object key) { gets++; check(this); return values.get(key); }
            public String put(String key, String value) { puts++; check(this); return values.put(key,value); }
            public Set<Map.Entry<String,String>> entrySet() { entries++; check(this); return values.entrySet(); }
        }
        Local local = new Local();
        Map<String,String> view = local;
        synchronized (local) {
            view.put("a","one");
            Map<String,String> input = new HashMap<>();
            input.put("b","two");
            view.putAll(input);
            boolean defaults = view.size() == 2 && !view.isEmpty() && view.containsKey("a") && view.containsValue("two");
            Set<String> keys = view.keySet();
            java.util.Collection<String> values = view.values();
            boolean views = keys == view.keySet() && values == view.values() && keys.contains("b") && values.contains("one");
            Map raw = view;
            int before = puts;
            boolean sourceBridge = false;
            try { raw.put("a",Integer.valueOf(7)); } catch (ClassCastException expected) { sourceBridge = puts == before && "one".equals(view.get("a")); }
            String computed = view.computeIfAbsent("c", key -> { check(local); return "three"; });
            String existing = view.putIfAbsent("a","replacement");
            String fallback = view.getOrDefault("absent","fallback");
            boolean callbacks = "three".equals(computed) && "one".equals(existing) && "fallback".equals(fallback);
            keys.remove("b");
            boolean live = !view.containsKey("b") && values.size() == 2;
            view.clear();
            return defaults && views && sourceBridge && callbacks && live && view.isEmpty() && gets > 0 && puts == 3 && entries > 0 && execution;
        }
    }
    static class AbstractMap<K,V> { int marker() { return 17; } }
    static boolean sourceLocal() {
        class Local extends AbstractMap<String,Integer> {}
        Local local = new Local();
        return local.marker() == 17 && !((Object)local instanceof java.util.Map);
    }
    public static void main(String[] args) {
        thread = Thread.currentThread();
        context.set("context");
        System.out.println("native-local=" + nativeLocal());
        System.out.println("source-shadow=" + sourceLocal());
    }
}
