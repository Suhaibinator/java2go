package probe.model;

import probe.api.Carrier;
import probe.api.Factory;

public final class NamedFactory implements Factory {
    private int calls;
    @Override public <T> Carrier<T> retain(Carrier<T> value) { calls++; return value; }
    public int calls() { return calls; }
}
