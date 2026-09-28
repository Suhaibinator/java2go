package prereq.reflection.state;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicInteger;

public final class Schedule {
  public final int seed;
  public final Thread caller;
  public final CountDownLatch start = new CountDownLatch(1);
  public final CountDownLatch successPublished = new CountDownLatch(1);
  public final CountDownLatch failurePublished = new CountDownLatch(1);
  public final AtomicInteger closed = new AtomicInteger();
  private final StringBuilder trace = new StringBuilder();

  public Schedule(int seed, Thread caller) {
    this.seed = seed;
    this.caller = caller;
  }

  public synchronized void event(String name) {
    if (trace.length() > 0) trace.append('>');
    trace.append(name);
  }

  public synchronized String trace() {
    return trace.toString();
  }
}
