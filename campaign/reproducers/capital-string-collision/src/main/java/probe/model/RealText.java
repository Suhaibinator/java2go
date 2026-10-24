package probe.model;

public final class RealText {
    private final int seed;
    private int toStringCalls;

    public RealText(int seed) {
        this.seed = seed;
    }

    @Override
    public String toString() {
        toStringCalls++;
        return "real:" + seed + ":" + toStringCalls;
    }

    public int toStringCalls() {
        return toStringCalls;
    }
}
