package probe.join.source;

import probe.join.state.Trace;

public final class TrackedText implements CharSequence {
    private final String id;
    private final String value;
    private final Trace trace;
    private final Runnable onString;
    private final RuntimeException failure;
    private final boolean returnsNull;
    private int conversions;

    public TrackedText(String id, String value, Trace trace, Runnable onString,
                       RuntimeException failure, boolean returnsNull) {
        this.id = id;
        this.value = value;
        this.trace = trace;
        this.onString = onString;
        this.failure = failure;
        this.returnsNull = returnsNull;
    }

    @Override
    public int length() {
        trace.add("length:" + id);
        return value.length();
    }

    @Override
    public char charAt(int index) {
        trace.add("charAt:" + id + ':' + index);
        return value.charAt(index);
    }

    @Override
    public CharSequence subSequence(int start, int end) {
        trace.add("subSequence:" + id + ':' + start + ':' + end);
        return value.subSequence(start, end);
    }

    @Override
    public String toString() {
        conversions++;
        trace.add("toString:" + id + '#' + conversions);
        if (onString != null) {
            onString.run();
        }
        if (failure != null) {
            throw failure;
        }
        return returnsNull ? null : value;
    }

    public int conversions() {
        return conversions;
    }
}
