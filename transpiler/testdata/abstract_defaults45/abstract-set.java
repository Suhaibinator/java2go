package probe.set;

import java.util.AbstractCollection;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

public final class Main {
    static final StringBuilder events = new StringBuilder();
    static void event(String text) { if (events.length() != 0) events.append(','); events.append(text); }
    static void show(String name, Object value) { System.out.println(name + "=" + value + "|" + events); events.setLength(0); }
    static String label(Object value) {
        if (value == null) return "null";
        if (value instanceof Tagged) return ((Tagged) value).name;
        return String.valueOf(value);
    }
    static final class Tagged {
        final String name;
        Tagged(String name) { this.name = name; }
        public boolean equals(Object other) {
            event("eq(" + name + "," + label(other) + ")");
            return other instanceof Tagged && name.equals(((Tagged) other).name);
        }
        public int hashCode() { event("hash(" + name + ")"); return name.hashCode(); }
    }
    static final class Cursor implements Iterator<Object> {
        final String owner;
        final Iterator<Object> source;
        Cursor(String owner, Iterator<Object> source) { this.owner = owner; this.source = source; }
        public boolean hasNext() { event(owner + ".has"); return source.hasNext(); }
        public Object next() { event(owner + ".next"); return source.next(); }
        public void remove() { event(owner + ".remove"); source.remove(); }
    }
    static class Values extends AbstractCollection<Object> {
        final String name;
        final List<Object> state = new ArrayList<>();
        Values(String name) { this.name = name; }
        void seed(Object value) { state.add(value); }
        public int size() { event(name + ".size"); return state.size(); }
        public Iterator<Object> iterator() { event(name + ".iterator"); return new Cursor(name, state.iterator()); }
    }
    static class StateSet extends AbstractSet<Object> {
        final String name;
        final List<Object> state = new ArrayList<>();
        StateSet(String name) { this.name = name; }
        // Test setup supplies distinct elements; inherited add is intentionally unsupported.
        void seed(Object value) { state.add(value); }
        public int size() { event(name + ".size"); return state.size(); }
        public Iterator<Object> iterator() { event(name + ".iterator"); return new Cursor(name, state.iterator()); }
    }
    static final class RejectingSet extends StateSet {
        final boolean nullError;
        RejectingSet(String name, boolean nullError) { super(name); this.nullError = nullError; }
        public boolean contains(Object ignored) {
            event(name + ".reject");
            if (nullError) throw new NullPointerException("probe");
            throw new ClassCastException("probe");
        }
    }
    public static void main(String[] args) {
        StateSet set = new StateSet("set");
        set.seed(new Tagged("a")); set.seed(null); set.seed(new Tagged("b"));
        StateSet peer = new StateSet("peer");
        peer.seed(new Tagged("b")); peer.seed(new Tagged("a")); peer.seed(null);
        show("equals.self", set.equals(set));
        show("equals.peer", set.equals(peer));
        show("equals.null", set.equals(null));
        show("equals.nonset", set.equals(new Values("bag")));
        show("hash", set.hashCode());
        show("empty", set.isEmpty());

        Values small = new Values("small"); small.seed(new Tagged("b"));
        show("removeAll.small", set.removeAll(small));
        show("size.after.small", set.size());
        Values large = new Values("large"); large.seed(new Tagged("a")); large.seed(null); large.seed(new Tagged("q"));
        show("removeAll.large", set.removeAll(large));
        show("empty.after", set.isEmpty());
        set.seed(new Tagged("tail"));
        set.clear(); show("clear", set.state.size());

        RejectingSet badCast = new RejectingSet("cast", false); badCast.seed(new Tagged("a"));
        RejectingSet badNull = new RejectingSet("null", true); badNull.seed(new Tagged("a"));
        StateSet one = new StateSet("one"); one.seed(new Tagged("a"));
        show("equals.cast", badCast.equals(one));
        show("equals.npe", badNull.equals(one));
    }
}
