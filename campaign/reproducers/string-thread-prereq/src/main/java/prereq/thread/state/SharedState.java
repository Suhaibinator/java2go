package prereq.thread.state;

import java.util.concurrent.CountDownLatch;

/** The latches also publish the references that cross the worker boundary. */
public final class SharedState {
  public final int seed;
  public final Thread caller;
  public final CountDownLatch start = new CountDownLatch(1);
  public final CountDownLatch firstInterned = new CountDownLatch(1);
  public final CountDownLatch secondObserved = new CountDownLatch(1);

  public volatile String firstFresh;
  public volatile String canonical;
  public volatile String unicodeFresh;
  public volatile String unicodeCanonical;
  public volatile String lateFresh;
  public volatile String lateCanonical;

  private final StringBuilder trace = new StringBuilder();

  public SharedState(int seed, Thread caller) {
    this.seed = seed;
    this.caller = caller;
  }

  public synchronized void event(String value) {
    if (trace.length() != 0) trace.append('>');
    trace.append(value);
  }

  public synchronized String trace() {
    return trace.toString();
  }
}
