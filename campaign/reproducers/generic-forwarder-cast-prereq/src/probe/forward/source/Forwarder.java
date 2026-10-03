package probe.forward.source;

import java.util.Iterator;
import probe.forward.state.Trace;

public final class Forwarder<T> implements Iterator<T> {
    private final Iterator<T> backing;
    private final Trace trace;
    private int returned;

    private Forwarder(Iterator<T> backing, Trace trace) {
        this.backing = backing;
        this.trace = trace;
    }

    @SuppressWarnings("unchecked")
    public static <T> Forwarder<T> fromRaw(Iterator<?> backing, Trace trace) {
        return new Forwarder<>((Iterator<T>) backing, trace);
    }

    @Override
    public boolean hasNext() {
        return backing.hasNext();
    }

    @Override
    public T next() {
        T value = backing.next();
        returned++;
        trace.record("forwarder.after-backing.next#" + returned);
        return value;
    }
}
