package prereq.thread.app;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import prereq.thread.flow.Observation;
import prereq.thread.flow.WorkerStep;
import prereq.thread.literal.LateLiteral;
import prereq.thread.state.Contexts;
import prereq.thread.state.SharedState;

public final class Main {
  private Main() {}

  private static void require(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }

  public static void main(String[] args) throws Exception {
    if (args.length != 1) throw new IllegalArgumentException("one seed required");
    int seed = Integer.parseInt(args[0]);
    SharedState shared = new SharedState(seed, Thread.currentThread());
    ExecutorService firstWorker = Executors.newSingleThreadExecutor();
    ExecutorService secondWorker = Executors.newSingleThreadExecutor();
    try {
      Future<Observation> first = firstWorker.submit(() -> WorkerStep.first(shared));
      Future<Observation> second = secondWorker.submit(() -> WorkerStep.second(shared));
      shared.start.countDown();
      Observation a1 = first.get();
      Observation b1 = second.get();
      boolean workerContract = a1.onWorker && b1.onWorker && a1.thread != b1.thread;
      require(workerContract,
          "callbacks must execute on separate workers");
      boolean isolated = a1.context != b1.context
          && Contexts.LOCAL.get() != a1.context
          && Contexts.LOCAL.get() != b1.context;
      require(isolated, "thread-local isolation");
      Contexts.LOCAL.remove();
      require(b1.equalDistinct && b1.sameIntern && b1.unicodePreserved,
          "cross-thread value, reference, and UTF-16 contract");

      String lateLiteral = LateLiteral.value();
      boolean lateSame = lateLiteral == shared.lateCanonical
          && shared.lateFresh == shared.lateCanonical;
      require(lateSame, "interning before late literal resolution");

      Future<Observation> aPersistent = firstWorker.submit(
          () -> WorkerStep.persistent(shared));
      Observation a2 = aPersistent.get();
      Future<Observation> bPersistent = secondWorker.submit(
          () -> WorkerStep.persistent(shared));
      Observation b2 = bPersistent.get();
      boolean persistent = a2.thread == a1.thread && b2.thread == b1.thread
          && a2.context == a1.context && b2.context == b1.context
          && a2.equalDistinct && b2.equalDistinct;
      require(persistent, "worker-local context persists for second callback");

      Future<Observation> aReset = firstWorker.submit(
          () -> WorkerStep.reinitialized(shared));
      Observation a3 = aReset.get();
      Future<Observation> bReset = secondWorker.submit(
          () -> WorkerStep.reinitialized(shared));
      Observation b3 = bReset.get();
      boolean reset = a3.thread == a1.thread && b3.thread == b1.thread
          && a3.context != a1.context && b3.context != b1.context
          && a3.context != b3.context && a3.equalDistinct && b3.equalDistinct;
      require(reset, "remove must reinitialize on each worker");

      shared.event("main:joined");
      require(shared.trace().equals(
          "A:intern>B:compare>B:intern>A:complete>main:joined"),
          "latch-defined event ordering");
      System.out.println("seed=" + seed);
      System.out.println("trace=" + shared.trace());
      System.out.println("workers=separate:" + (a1.thread != b1.thread)
          + ":callback:" + workerContract);
      System.out.println("context=isolated:" + isolated + ":persistent:"
          + persistent + ":reset:" + reset);
      System.out.println("dynamic=distinct-equal:" + b1.equalDistinct
          + ":intern:" + b1.sameIntern + ":hash:" + shared.firstFresh.hashCode());
      System.out.println("late-literal=" + lateSame);
      System.out.println("unicode=length:" + shared.unicodeFresh.length()
          + ":high:" + (int) shared.unicodeFresh.charAt(0)
          + ":low:" + (int) shared.unicodeFresh.charAt(2)
          + ":intern:" + b1.sameIntern);
    } finally {
      shared.start.countDown();
      shared.firstInterned.countDown();
      shared.secondObserved.countDown();
      firstWorker.shutdownNow();
      secondWorker.shutdownNow();
    }
  }
}
