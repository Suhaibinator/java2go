package parity.objectmodel.model;

public final class RecursiveChain<T> {
    private final T value;
    private final RecursiveChain<T> next;

    public RecursiveChain(T value, RecursiveChain<T> next) {
        this.value = value;
        this.next = next;
    }

    public int size() {
        return next == null ? 1 : 1 + next.size();
    }

    public T tail() {
        return next == null ? value : next.tail();
    }
}
