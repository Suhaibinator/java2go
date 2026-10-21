package emptyset;

import java.io.Serializable;
import java.util.AbstractCollection;
import java.util.Collection;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.Set;

public class EmptySetControl {
    static final ThreadLocal<String> local = new ThreadLocal<>();
    static class IsEmptyProbe extends AbstractCollection<java.lang.Object> {
        final java.lang.Object lock;
        final Thread thread;
        final boolean result;
        int calls;
        boolean context;
        IsEmptyProbe(java.lang.Object lock, boolean result) {
            this.lock = lock;
            this.thread = Thread.currentThread();
            this.result = result;
        }
        public boolean isEmpty() {
            calls++;
            context = Thread.currentThread() == thread && Thread.holdsLock(lock)
                    && "kept".equals(local.get());
            return result;
        }
        public int size() { throw new AssertionError("size must not be called"); }
        public Iterator<java.lang.Object> iterator() { throw new AssertionError("iterator must not be called"); }
    }
    static class EqualsBomb {
        public boolean equals(java.lang.Object other) { throw new AssertionError("equals must not be called"); }
        public int hashCode() { throw new AssertionError("hash must not be called"); }
    }
    @SuppressWarnings({"rawtypes", "unchecked"})
    public static void main(String[] args) {
        Set<String> strings = Collections.emptySet();
        Set<Integer> integers = Collections.<Integer>emptySet();
        java.lang.Object first = strings;
        java.lang.Object second = integers;
        boolean identity = first == second;
        synchronized (first) { identity &= Thread.holdsLock(second); }
        boolean nominal = first instanceof Set && first instanceof Collection
                && first instanceof Iterable && first instanceof Serializable
                && !(first instanceof List) && !(first instanceof Map)
                && first.getClass().getName().equals("java.util.Collections$EmptySet");
        Set raw = strings;
        boolean addNull = false;
        boolean addValue = false;
        try { raw.add(null); } catch (UnsupportedOperationException expected) { addNull = true; }
        try { raw.add(new java.lang.Object()); } catch (UnsupportedOperationException expected) { addValue = true; }
        boolean mutation = addNull && addValue && !raw.remove(null)
                && !raw.remove(new EqualsBomb()) && !raw.contains(null);
        raw.clear();
        mutation &= raw.size() == 0 && raw.isEmpty() && first == raw;
        Iterator<String> cursor = strings.iterator();
        java.lang.Object otherCursor = integers.iterator();
        boolean nextFailure = false;
        boolean removeFailure = false;
        try { cursor.next(); } catch (NoSuchElementException expected) { nextFailure = true; }
        try { cursor.remove(); } catch (IllegalStateException expected) { removeFailure = true; }
        int count = 0;
        for (String value : strings) { count++; }
        boolean iteration = cursor == otherCursor && !cursor.hasNext()
                && nextFailure && removeFailure && count == 0;
        IsEmptyProbe yes = new IsEmptyProbe(first, true);
        IsEmptyProbe no = new IsEmptyProbe(first, false);
        local.set("kept");
        boolean callbacks;
        synchronized (first) {
            callbacks = strings.containsAll(yes) && !strings.containsAll(no);
        }
        local.remove();
        boolean nullFailure = false;
        try { strings.containsAll(null); } catch (NullPointerException expected) { nullFailure = true; }
        callbacks &= yes.calls == 1 && no.calls == 1 && yes.context && no.context && nullFailure;
        System.out.println("identity=" + identity);
        System.out.println("nominal=" + nominal);
        System.out.println("mutation=" + mutation);
        System.out.println("iteration=" + iteration);
        System.out.println("callbacks=" + callbacks);
    }
}
