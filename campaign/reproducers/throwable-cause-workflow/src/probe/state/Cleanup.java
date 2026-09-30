package probe.state;

public final class Cleanup implements AutoCloseable {
    private final Trace trace;
    private final RuntimeException closeFailure;

    public Cleanup(Trace trace, RuntimeException closeFailure) {
        this.trace = trace;
        this.closeFailure = closeFailure;
    }

    @Override
    public void close() {
        trace.closed();
        if (closeFailure != null) {
            throw closeFailure;
        }
    }
}
