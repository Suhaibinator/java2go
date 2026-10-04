package probe.error;
import probe.state.Journal;
public final class CallbackCause extends Exception {
    private final Journal journal;
    private final String rendering;
    private final Signal failure;
    private final Thread caller;
    private int calls;
    private boolean held;
    private boolean sameThread;
    public CallbackCause(Journal journal, String rendering, Signal failure) {
        super((String) null); this.journal = journal; this.rendering = rendering; this.failure = failure;
        caller = Thread.currentThread();
    }
    @Override public String toString() {
        calls++; held = Thread.holdsLock(journal.lock()); sameThread = Thread.currentThread() == caller;
        journal.mark("render"); if (failure != null) throw failure; return rendering;
    }
    public int calls() { return calls; }
    public boolean held() { return held; }
    public boolean sameThread() { return sameThread; }
}
