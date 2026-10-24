package probe.collection;

import java.util.AbstractCollection;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;
import java.util.List;

public final class Main {
    static final StringBuilder events = new StringBuilder();
    static void event(String text) {
        if (events.length() != 0) events.append(',');
        events.append(text);
    }
    static void show(String name, Object result) {
        System.out.println(name + "=" + result + "|" + events);
        events.setLength(0);
    }
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
        public int hashCode() { return name.hashCode(); }
    }
    static final class Cursor implements Iterator<Object> {
        final Iterator<Object> source;
        Cursor(Iterator<Object> source) { this.source = source; }
        public boolean hasNext() { event("has"); return source.hasNext(); }
        public Object next() { event("next"); return source.next(); }
        public void remove() { event("remove"); source.remove(); }
    }
    static final class Bag extends AbstractCollection<Object> {
        final List<Object> state = new ArrayList<>();
        void seed(Object value) { state.add(value); }
        public int size() { event("size"); return state.size(); }
        public Iterator<Object> iterator() { event("iterator"); return new Cursor(state.iterator()); }
    }
    static Object argument() { event("argument"); return new Tagged("q"); }
    @SuppressWarnings({"unchecked", "rawtypes"})
    public static void main(String[] args) {
        Bag bag = new Bag();
        bag.seed(new Tagged("a")); bag.seed(null); bag.seed(new Tagged("b")); bag.seed(new Tagged("b"));
        show("empty.before", bag.isEmpty());
        show("contains.b", bag.contains(new Tagged("b")));
        show("contains.null", bag.contains(null));
        show("contains.absent", bag.contains(new Tagged("q")));
        show("remove.b", bag.remove(new Tagged("b")));
        show("size.after.b", bag.size());
        show("remove.null", bag.remove(null));
        show("remove.absent", bag.remove(new Tagged("q")));
        bag.clear(); show("clear", bag.state.size());
        show("empty.after", bag.isEmpty());

        Bag polluted = new Bag();
        polluted.seed("good"); polluted.seed(Integer.valueOf(9)); polluted.seed("tail");
        Collection<String> typed = (Collection<String>) (Collection) polluted;
        Iterator<String> cursor = typed.iterator();
        Object first = cursor.next();
        boolean failed = false;
        try { String bad = cursor.next(); bad.length(); }
        catch (ClassCastException expected) { failed = true; }
        String recovered = cursor.next();
        show("consumer.cast", first + ":" + failed + ":" + recovered + ":" + cursor.hasNext());

        Collection<Object> missing = null;
        boolean nullFailed = false;
        try { missing.contains(argument()); }
        catch (NullPointerException expected) { nullFailed = true; }
        show("null.argument", nullFailed);
    }
}
