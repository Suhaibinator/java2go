package prereq.factory.api;

public abstract class Adapter<T> {
    private int calls;

    protected final void recordCall() {
        calls++;
    }

    public final int calls() {
        return calls;
    }

    public abstract T apply(T input);
}
