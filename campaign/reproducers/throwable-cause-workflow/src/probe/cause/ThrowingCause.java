package probe.cause;

import probe.state.Trace;

public final class ThrowingCause extends Throwable {
    private final Trace trace;
    private final RuntimeException failure;

    public ThrowingCause(Trace trace, RuntimeException failure) {
        super("stored-" + trace.id());
        this.trace = trace;
        this.failure = failure;
    }

    @Override
    public String toString() {
        trace.describeThrowing(failure);
        throw new AssertionError("unreachable");
    }
}
