package workflow.state;
public final class Ledger {
    public final Object lock = new Object();
    public final int seed;
    public final Fault fault;
    private final ThreadLocal<Integer> caller = new ThreadLocal<Integer>();
    private Integer expected;
    private StringBuilder trace = new StringBuilder();
    public int reads, mismatches, finishes;
    public Ledger(int seed) { this.seed = seed; fault = new Fault(seed); }
    public static final class Fault extends RuntimeException {
        public final int code;
        public Fault(int code) { super(); this.code = code; }
    }
    public void install(Integer value) { expected = value; caller.set(value); }
    public void reset() { trace = new StringBuilder(); reads = 0; mismatches = 0; finishes = 0; }
    public void record(String mark) { trace.append(mark).append(':'); }
    public void touch() { reads++; if (caller.get() != expected) mismatches++; }
    public String pickString(String value) { touch(); record("R"); return value; }
    public CharSequence pickSequence(CharSequence value) { touch(); record("R"); return value; }
    public Integer start(Integer value) { touch(); record("A"); return value; }
    public int end(int value) { touch(); record("B"); return value; }
    public void finish(String mark) {
        synchronized (lock) { touch(); finishes++; record(mark); }
    }
    public String report() { return trace.toString() + "|" + reads + ":" + mismatches + ":" + finishes; }
    public void remove() { caller.remove(); }
    public boolean absent() { return caller.get() == null; }
}
