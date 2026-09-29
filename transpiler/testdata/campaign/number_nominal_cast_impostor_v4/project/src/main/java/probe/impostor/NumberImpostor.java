package probe.impostor;

// This class has the full numeric method shape but no java.lang.Number ancestry.
// A reference checkcast must not invoke any of these conversion methods.
public final class NumberImpostor {
    private int byteCalls;
    private int shortCalls;
    private int intCalls;
    private int longCalls;
    private int floatCalls;
    private int doubleCalls;
    public byte byteValue() { byteCalls++; throw new AssertionError("impostor.byteValue"); }
    public short shortValue() { shortCalls++; throw new AssertionError("impostor.shortValue"); }
    public int intValue() { intCalls++; throw new AssertionError("impostor.intValue"); }
    public long longValue() { longCalls++; throw new AssertionError("impostor.longValue"); }
    public float floatValue() { floatCalls++; throw new AssertionError("impostor.floatValue"); }
    public double doubleValue() { doubleCalls++; throw new AssertionError("impostor.doubleValue"); }
    public String calls() {
        return byteCalls + ":" + shortCalls + ":" + intCalls + ":" + longCalls + ":" + floatCalls + ":" + doubleCalls;
    }
}
