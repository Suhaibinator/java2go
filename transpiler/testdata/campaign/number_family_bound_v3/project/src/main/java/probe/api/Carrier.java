package probe.api;

public abstract class Carrier<T> {
    private T remembered;
    private int snapshots;
    protected final void remember(T value) { remembered = value; }
    public final T snapshot() { snapshots++; return remembered; }
    public final int snapshots() { return snapshots; }
    public abstract T get();
    public abstract void put(T value);
}
