package prereq.factory.cache;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import org.apache.commons.lang3.mutable.MutableInt;
import prereq.factory.api.Adapter;
import prereq.factory.api.Factory;
import prereq.factory.api.Token;

public final class Context {
    private final List<Factory> factories = new ArrayList<>();
    private final Map<String, Adapter<?>> cache = new LinkedHashMap<>();
    private final MutableInt probes = new MutableInt();
    private final MutableInt misses = new MutableInt();
    private final MutableInt hits = new MutableInt();
    private final MutableInt insertions = new MutableInt();
    private final StringBuilder trace = new StringBuilder();

    public void add(Factory factory) {
        factories.add(factory);
    }

    public void probe(char route, String kind) {
        probes.increment();
        trace.append(route).append('(').append(kind).append(')');
    }

    @SuppressWarnings("unchecked")
    public <T> Adapter<T> lookup(Token<T> token) {
        if (cache.containsKey(token.kind())) {
            hits.increment();
            return (Adapter<T>) cache.get(token.kind());
        }
        misses.increment();
        for (Factory factory : factories) {
            Adapter<T> adapter = factory.create(this, token);
            if (adapter != null) {
                cache.put(token.kind(), adapter);
                insertions.increment();
                return adapter;
            }
        }
        return null;
    }

    public String snapshot() {
        return probes.intValue() + "," + misses.intValue() + "," + hits.intValue()
                + "," + insertions.intValue() + "," + cache.size() + ":" + trace;
    }
}
