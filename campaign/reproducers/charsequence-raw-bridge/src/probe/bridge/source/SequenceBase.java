package probe.bridge.source;

import probe.bridge.state.Trace;

public class SequenceBase implements CharSequence {
    private final String label;
    private final String value;
    private final Trace trace;
    private final Runnable conversionAction;
    private int conversions;

    public SequenceBase(String label, String value, Trace trace, Runnable conversionAction) {
        this.label = label;
        this.value = value;
        this.trace = trace;
        this.conversionAction = conversionAction;
    }

    @Override
    public int length() {
        trace.record("length:" + label);
        return value.length();
    }

    @Override
    public char charAt(int index) {
        trace.record("charAt:" + label + ':' + index);
        return value.charAt(index);
    }

    @Override
    public CharSequence subSequence(int start, int end) {
        trace.record("subSequence:" + label + ':' + start + ':' + end);
        return value.subSequence(start, end);
    }

    @Override
    public String toString() {
        conversions++;
        trace.record("toString:" + label + '#' + conversions);
        if (conversionAction != null) {
            conversionAction.run();
        }
        return value;
    }

    public int conversions() {
        return conversions;
    }
}
