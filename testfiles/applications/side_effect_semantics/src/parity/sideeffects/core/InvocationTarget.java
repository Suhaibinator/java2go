package parity.sideeffects.core;

public final class InvocationTarget {
    private final TraceLog trace;

    public InvocationTarget(TraceLog trace) {
        this.trace = trace;
    }

    public int combine(int left, int right) {
        trace.add("m");
        return left * 10 + right;
    }

    public static int staticCombine(TraceLog trace, int left, int right) {
        trace.add("s");
        return left * 10 + right;
    }
}
