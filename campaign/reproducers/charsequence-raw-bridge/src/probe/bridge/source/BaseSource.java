package probe.bridge.source;

import java.util.Iterator;
import java.util.List;
import probe.bridge.state.Trace;

public class BaseSource<T> implements Iterable<T> {
    private final List<T> values;
    private final Trace trace;
    private int iteratorCount;

    public BaseSource(List<T> values, Trace trace) {
        this.values = values;
        this.trace = trace;
    }

    @Override
    public Iterator<T> iterator() {
        iteratorCount++;
        trace.record("iterator#" + iteratorCount);
        Iterator<T> backing = values.iterator();
        return new Iterator<T>() {
            @Override
            public boolean hasNext() {
                trace.record("hasNext");
                return backing.hasNext();
            }

            @Override
            public T next() {
                trace.record("next");
                return backing.next();
            }
        };
    }
}
