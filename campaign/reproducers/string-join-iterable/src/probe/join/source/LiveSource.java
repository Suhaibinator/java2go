package probe.join.source;

import java.util.Iterator;
import java.util.List;
import probe.join.state.Trace;

public final class LiveSource implements Iterable<CharSequence> {
    private final List<CharSequence> values;
    private final Trace trace;
    private int iterators;

    public LiveSource(List<CharSequence> values, Trace trace) {
        this.values = values;
        this.trace = trace;
    }

    @Override
    public Iterator<CharSequence> iterator() {
        iterators++;
        trace.add("iterator#" + iterators);
        if (iterators != 1) {
            throw new IllegalStateException("source iterated twice");
        }
        Iterator<CharSequence> backing = values.iterator();
        return new Iterator<CharSequence>() {
            @Override
            public boolean hasNext() {
                trace.add("hasNext");
                return backing.hasNext();
            }

            @Override
            public CharSequence next() {
                trace.add("next");
                return backing.next();
            }

            @Override
            public void remove() {
                trace.add("remove");
                backing.remove();
            }
        };
    }
}
