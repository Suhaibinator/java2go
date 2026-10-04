package probe.model;

public final class CapitalText {
    private final int seed;
    private int capitalCalls;
    private int hashCalls;

    public CapitalText(int seed) {
        this.seed = seed;
    }

    // This is an ordinary Java method. It does not override Object.toString().
    public String String() {
        capitalCalls++;
        return "capital:" + seed + ":" + capitalCalls;
    }

    @Override
    public int hashCode() {
        hashCalls++;
        return seed + 42;
    }

    public int capitalCalls() {
        return capitalCalls;
    }

    public int hashCalls() {
        return hashCalls;
    }
}
