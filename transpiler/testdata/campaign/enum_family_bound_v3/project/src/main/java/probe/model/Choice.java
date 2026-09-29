package probe.model;

public enum Choice {
    FIRST,
    SPECIAL {
        @Override public synchronized String toString() {
            synchronized (this) {
                record();
                return "special:" + name() + ":" + calls();
            }
        }
    };
    private int calls;
    protected final void record() { calls++; }
    public final int calls() { return calls; }
    @Override public synchronized String toString() {
        record();
        return "ordinary:" + name() + ":" + calls;
    }
}
