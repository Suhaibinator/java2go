package probe.error;
public final class Signal extends RuntimeException {
    public Signal(int seed) { super("signal" + seed); }
}
