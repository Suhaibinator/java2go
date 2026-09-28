package prereq.thread.flow;

import prereq.thread.state.Contexts;
import prereq.thread.state.SharedState;
import prereq.thread.state.WorkerContext;

/** The Main callbacks enter these methods on their designated executor threads. */
public final class WorkerStep {
  private WorkerStep() {}

  private static WorkerContext visit() {
    WorkerContext context = Contexts.LOCAL.get();
    context.visits++;
    return context;
  }

  public static Observation first(SharedState shared) throws InterruptedException {
    shared.start.await();
    WorkerContext context = visit();
    try {
      String text = "job-" + shared.seed + "-\u03c0";
      shared.firstFresh = new String(text.toCharArray());
      shared.canonical = shared.firstFresh.intern();
      shared.unicodeFresh = new String(new char[] {'\ud800', 'x', '\udc00', '\u03c0'});
      shared.unicodeCanonical = shared.unicodeFresh.intern();
      shared.lateFresh = new String(new char[] {
          'l', 'a', 't', 'e', '-', 'c', 'a', 'n', 'o', 'n', 'i', 'c', 'a', 'l'});
      shared.lateCanonical = shared.lateFresh.intern();
      shared.event("A:intern");
      shared.firstInterned.countDown();
      shared.secondObserved.await();
      shared.event("A:complete");
      return new Observation(Thread.currentThread(), context,
          Thread.currentThread() != shared.caller, true, true, true);
    } finally {
      shared.firstInterned.countDown();
    }
  }

  public static Observation second(SharedState shared) throws InterruptedException {
    shared.start.await();
    WorkerContext context = visit();
    try {
      shared.firstInterned.await();
      String text = new StringBuilder().append("job-").append(shared.seed)
          .append('-').append('\u03c0').toString();
      String unicode = new StringBuilder().append('\ud800').append('x')
          .append('\udc00').append('\u03c0').toString();
      boolean equalDistinct = text.equals(shared.firstFresh)
          && text != shared.firstFresh;
      boolean sameIntern = text.intern() == shared.canonical
          && unicode.intern() == shared.unicodeCanonical;
      boolean unicodePreserved = unicode != shared.unicodeFresh
          && unicode.equals(shared.unicodeFresh)
          && unicode.length() == 4
          && unicode.charAt(0) == '\ud800'
          && unicode.charAt(2) == '\udc00';
      shared.event("B:compare");
      shared.event("B:intern");
      return new Observation(Thread.currentThread(), context,
          Thread.currentThread() != shared.caller,
          equalDistinct, sameIntern, unicodePreserved);
    } finally {
      shared.secondObserved.countDown();
    }
  }

  public static Observation persistent(SharedState shared) {
    WorkerContext context = visit();
    boolean preserved = context.visits == 2;
    Contexts.LOCAL.remove();
    return new Observation(Thread.currentThread(), context,
        Thread.currentThread() != shared.caller, preserved, true, true);
  }

  public static Observation reinitialized(SharedState shared) {
    WorkerContext context = visit();
    boolean reset = context.visits == 1;
    Contexts.LOCAL.remove();
    return new Observation(Thread.currentThread(), context,
        Thread.currentThread() != shared.caller, reset, true, true);
  }
}
