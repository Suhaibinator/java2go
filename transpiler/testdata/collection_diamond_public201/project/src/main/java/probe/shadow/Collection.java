package probe.shadow;

public final class Collection {
    private final int value;
    public Collection(int seed) { value = seed * 2 + 7; }
    public int value() { return value; }
}
