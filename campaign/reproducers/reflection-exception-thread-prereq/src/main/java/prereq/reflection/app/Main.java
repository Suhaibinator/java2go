package prereq.reflection.app;

import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import prereq.reflection.api.Outcome;
import prereq.reflection.error.NoMatchingConstructor;
import prereq.reflection.error.ProbeException;
import prereq.reflection.jobs.FailureJob;
import prereq.reflection.jobs.SuccessJob;
import prereq.reflection.state.Context;
import prereq.reflection.state.Schedule;

public final class Main {
  private Main() {}

  private static void require(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }

  public static void main(String[] args) throws Exception {
    if (args.length != 1) throw new IllegalArgumentException("one seed required");
    int seed = Integer.parseInt(args[0]);
    Schedule schedule = new Schedule(seed, Thread.currentThread());
    ExecutorService firstWorker = Executors.newSingleThreadExecutor();
    ExecutorService secondWorker = Executors.newSingleThreadExecutor();
    try {
      Future<Outcome<ProbeException>> first = firstWorker.submit(new SuccessJob(schedule));
      Future<Outcome<NoMatchingConstructor>> second =
          secondWorker.submit(new FailureJob(schedule));
      schedule.start.countDown();
      Outcome<ProbeException> success = first.get();
      Outcome<NoMatchingConstructor> failure = second.get();

      boolean threadIdentity = success.callbackOnWorker && failure.callbackOnWorker
          && success.callbackThread != failure.callbackThread;
      boolean successContract = success.reflectiveFailure == null
          && success.constructed != null
          && success.constructed.getClass().getSimpleName().equals("ProbeException")
          && success.constructed.getMessage().equals("job-" + seed)
          && success.constructed.getCause() == success.intendedCause
          && success.constructed.getCause().getClass().getSimpleName()
              .equals("IllegalStateException")
          && success.constructed.getCause().getMessage().equals("root-" + seed);
      boolean failureContract = failure.constructed == null
          && failure.reflectiveFailure != null
          && failure.reflectiveFailure.getClass().getSimpleName()
              .equals("NoSuchMethodException")
          && failure.reflectiveFailure.getMessage() != null
          && failure.reflectiveFailure.getMessage().contains("NoMatchingConstructor")
          && failure.reflectiveFailure.getCause() == null;
      Context callerContext = Context.LOCAL.get();
      boolean isolated = success.context != failure.context
          && callerContext != success.context && callerContext != failure.context;
      Context.LOCAL.remove();
      Future<Boolean> firstReset = firstWorker.submit(() -> {
        Context next = Context.LOCAL.get();
        boolean correct = next != success.context && next.visits == 0
            && Thread.currentThread() == success.callbackThread;
        Context.LOCAL.remove();
        return correct;
      });
      Future<Boolean> secondReset = secondWorker.submit(() -> {
        Context next = Context.LOCAL.get();
        boolean correct = next != failure.context && next.visits == 0
            && Thread.currentThread() == failure.callbackThread;
        Context.LOCAL.remove();
        return correct;
      });
      boolean cleaned = schedule.closed.get() == 2
          && firstReset.get() && secondReset.get();
      schedule.event("main:joined");
      String expectedTrace = "A:open>A:constructed>A:closed>A:publish>"
          + "B:open>B:failure>B:closed>B:publish>A:complete>main:joined";
      require(threadIdentity && successContract && failureContract
          && isolated && cleaned && schedule.trace().equals(expectedTrace),
          "reflection, inherited callback, ordering, or cleanup contract");

      System.out.println("seed=" + seed);
      System.out.println("trace=" + schedule.trace());
      System.out.println("threads=distinct:" + threadIdentity + ":isolated:" + isolated);
      System.out.println("success=" + success.constructed.getClass().getSimpleName()
          + ":" + success.constructed.getMessage() + ":cause="
          + success.constructed.getCause().getClass().getSimpleName() + ":"
          + success.constructed.getCause().getMessage()
          + ":identity:" + (success.constructed.getCause() == success.intendedCause));
      System.out.println("failure=" + failure.reflectiveFailure.getClass().getSimpleName()
          + ":message:" + (failure.reflectiveFailure.getMessage() != null)
          + ":cause-null:" + (failure.reflectiveFailure.getCause() == null));
      System.out.println("cleanup=closed:" + schedule.closed.get() + ":reset:" + cleaned);
    } finally {
      schedule.start.countDown();
      schedule.successPublished.countDown();
      schedule.failurePublished.countDown();
      firstWorker.shutdownNow();
      secondWorker.shutdownNow();
    }
  }
}
