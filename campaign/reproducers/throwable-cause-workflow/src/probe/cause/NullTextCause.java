package probe.cause;

import probe.state.Trace;

public final class NullTextCause extends Throwable {
    private final Trace trace;

    public NullTextCause(Trace trace) {
        super("stored-" + trace.id());
        this.trace = trace;
    }

    @Override
    public String toString() {
        return trace.describeNull();
    }
}
