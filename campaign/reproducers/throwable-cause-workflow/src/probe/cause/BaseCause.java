package probe.cause;

import probe.state.Trace;

public class BaseCause extends Throwable {
    private final Trace trace;

    public BaseCause(Trace trace) {
        super("stored-" + trace.id());
        this.trace = trace;
    }

    @Override
    public String toString() {
        return trace.describe();
    }
}
