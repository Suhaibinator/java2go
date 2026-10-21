package generic;

import model.State;

public class Identity {
    public static <T> T identity(T value) { State.calls++; State.mark("I"); return value; }
    public static <N extends java.lang.Number> N number(N value) { State.calls++; State.mark("N"); return value; }
    public static <N extends java.lang.Number> N choose(boolean first, N left, N right) {
        State.calls++; State.mark("C"); return first ? left : right;
    }
    @SuppressWarnings("unchecked")
    public static <T> T polluted(Object value) { State.calls++; State.mark("P"); return (T)value; }
    @SuppressWarnings("unchecked")
    public static <N extends java.lang.Number> N numberPolluted(Object value) {
        State.calls++; State.mark("B"); return (N)value;
    }
    public static <T> T fail(T value) { State.calls++; State.mark("F"); throw new IllegalStateException("body"); }
}
