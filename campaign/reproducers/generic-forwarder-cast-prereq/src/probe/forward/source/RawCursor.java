package probe.forward.source;

import java.util.Iterator;
import java.util.List;
import java.util.NoSuchElementException;
import probe.forward.state.Trace;

public final class RawCursor implements Iterator<Object> {
    private final List<Object> values;
    private final Trace trace;
    private int position;

    public RawCursor(List<Object> values, Trace trace) {
        this.values = values;
        this.trace = trace;
    }

    @Override
    public boolean hasNext() {
        trace.record("backing.hasNext@" + position);
        return position < values.size();
    }

    @Override
    public Object next() {
        if (position >= values.size()) {
            throw new NoSuchElementException();
        }
        Object value = values.get(position);
        position++;
        trace.record("backing.next#" + position);
        return value;
    }

    public int position() {
        return position;
    }
}
