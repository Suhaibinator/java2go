package probe.model;

import probe.api.Carrier;

public class NumberCarrier<N extends java.lang.Number> extends Carrier<N> {
    private N current;
    private int reads;
    private int writes;
    private int casts;
    @Override public N get() { reads++; return current; }
    @Override public void put(N value) { writes++; current = value; remember(value); }
    public int reads() { return reads; }
    public int writes() { return writes; }
    public String state() { return reads + ":" + writes + ":" + snapshots() + ":" + casts; }
    @SuppressWarnings("unchecked")
    public N cast(Object value) { casts++; return (N) value; }
    // This method binder is distinct from the class-owned N and has Object erasure.
    public <N> N shadow(N value) { return value; }
}
