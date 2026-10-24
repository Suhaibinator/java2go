package probe.map;

import java.util.AbstractMap;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Collection;
import java.util.ConcurrentModificationException;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;

public final class Main {
    static final StringBuilder events = new StringBuilder();
    static void event(String text) { if (events.length() != 0) events.append(','); events.append(text); }
    static void show(String name, Object value) { System.out.println(name + "=" + value + "|" + events); events.setLength(0); }
    static boolean same(Object left, Object right) { return left == null ? right == null : left.equals(right); }
    static int hash(Object value) { return value == null ? 0 : value.hashCode(); }

    // A real application implementation of the canonical JDK Map.Entry interface.
    static final class Entry implements Map.Entry<String, String> {
        final String key;
        String value;
        Entry(String key, String value) { this.key = key; this.value = value; }
        public String getKey() { event("key(" + key + ")"); return key; }
        public String getValue() { event("value(" + key + ")"); return value; }
        public String setValue(String next) { event("set(" + key + ")"); String old = value; value = next; return old; }
        public boolean equals(Object other) {
            event("entry.equals(" + key + ")");
            if (!(other instanceof Map.Entry)) return false;
            Map.Entry<?, ?> entry = (Map.Entry<?, ?>) other;
            return same(key, entry.getKey()) && same(value, entry.getValue());
        }
        public int hashCode() { event("entry.hash(" + key + ")"); return hash(key) ^ hash(value); }
    }
    static final class Cursor implements Iterator<Map.Entry<String, String>> {
        final String owner;
        final Iterator<Map.Entry<String, String>> source;
        Cursor(String owner, Iterator<Map.Entry<String, String>> source) { this.owner = owner; this.source = source; }
        public boolean hasNext() { event(owner + ".has"); return source.hasNext(); }
        public Map.Entry<String, String> next() { event(owner + ".next"); return source.next(); }
        public void remove() { event(owner + ".remove"); source.remove(); }
    }
    static final class Entries extends AbstractSet<Map.Entry<String, String>> {
        final String owner;
        final List<Map.Entry<String, String>> state = new ArrayList<>();
        Entries(String owner) { this.owner = owner; }
        public int size() { event(owner + ".size"); return state.size(); }
        public Iterator<Map.Entry<String, String>> iterator() { event(owner + ".iterator"); return new Cursor(owner, state.iterator()); }
    }
    static class StateMap extends AbstractMap<String, String> {
        final String name;
        final Entries entries;
        StateMap(String name) { this.name = name; this.entries = new Entries(name); }
        void seed(String key, String value) { entries.state.add(new Main.Entry(key, value)); }
        public Set<Map.Entry<String, String>> entrySet() { event(name + ".entrySet"); return entries; }
    }
    static final class RejectingMap extends StateMap {
        final boolean nullError;
        RejectingMap(String name, boolean nullError) { super(name); this.nullError = nullError; }
        public String get(Object ignored) {
            event(name + ".reject");
            if (nullError) throw new NullPointerException("probe");
            throw new ClassCastException("probe");
        }
    }
    public static void main(String[] args) {
        StateMap map = new StateMap("map");
        map.seed("a", "x"); map.seed("nullable", null); map.seed(null, "n"); map.seed("b", "y");
        show("size", map.size());
        show("empty", map.isEmpty());
        show("contains.a", map.containsKey("a"));
        show("contains.null", map.containsKey(null));
        show("get.null", map.get(null));
        show("get.nullable", map.get("nullable"));
        show("get.absent", map.get("absent"));
        show("containsValue.null", map.containsValue(null));
        show("remove.a", map.remove("a"));
        show("remove.absent", map.remove("absent"));

        StateMap peer = new StateMap("peer");
        peer.seed("b", "y"); peer.seed(null, "n"); peer.seed("nullable", null);
        show("equals.self", map.equals(map));
        show("equals.peer", map.equals(peer));
        show("hash", map.hashCode());
        peer.remove("nullable"); events.setLength(0);
        show("equals.absent.null", map.equals(peer));

        RejectingMap badCast = new RejectingMap("cast", false);
        badCast.seed("nullable", null); badCast.seed(null, "n"); badCast.seed("b", "y");
        RejectingMap badNull = new RejectingMap("npe", true);
        badNull.seed("nullable", null); badNull.seed(null, "n"); badNull.seed("b", "y");
        show("equals.cast", map.equals(badCast));
        show("equals.npe", map.equals(badNull));

        Set<String> keys = map.keySet();
        Collection<String> values = map.values();
        show("views.cached", (keys == map.keySet()) + ":" + (values == map.values()));
        Iterator<String> removal = values.iterator();
        String removed = removal.next();
        removal.remove();
        boolean repeat = false;
        try { removal.remove(); } catch (IllegalStateException expected) { repeat = true; }
        show("values.remove", removed + ":" + repeat + ":" + keys.contains("nullable"));

        Map.Entry<String, String> retained = map.entrySet().iterator().next();
        String old = retained.setValue("updated");
        show("entry.replace", old + ":" + map.get(null));
        Iterator<Map.Entry<String, String>> stale = map.entrySet().iterator();
        stale.next(); map.seed("tail", "z");
        boolean concurrent = false;
        try { stale.next(); } catch (ConcurrentModificationException expected) { concurrent = true; }
        show("cursor.stale", concurrent);

        boolean unsupported = false;
        try { map.put("blocked", "value"); }
        catch (UnsupportedOperationException expected) { unsupported = true; }
        show("put.default", unsupported);
        map.clear(); show("clear", map.entries.state.size());
        show("views.after", keys.isEmpty() + ":" + values.isEmpty() + ":" + map.isEmpty());
    }
}
