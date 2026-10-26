package prereq.reflection.api;

import java.lang.reflect.Constructor;
import java.util.concurrent.Callable;
import prereq.reflection.state.Context;
import prereq.reflection.state.Lease;
import prereq.reflection.state.Schedule;

/** Subclasses inherit both the generic reflective owner and Callable callback. */
public abstract class ReflectiveFactory<T extends Exception>
    implements Callable<Outcome<T>> {
  private final String role;
  private final Class<T> type;
  protected final Schedule schedule;

  protected ReflectiveFactory(String role, Class<T> type, Schedule schedule) {
    this.role = role;
    this.type = type;
    this.schedule = schedule;
  }

  protected final T construct(String message, Throwable cause)
      throws ReflectiveOperationException {
    Constructor<T> constructor = type.getConstructor(String.class, Throwable.class);
    return constructor.newInstance(message, cause);
  }

  @Override
  public final Outcome<T> call() throws InterruptedException {
    schedule.start.await();
    Context context = Context.LOCAL.get();
    context.visits++;
    try {
      if ("B".equals(role)) schedule.successPublished.await();
      Throwable cause = new IllegalStateException("root-" + schedule.seed);
      T constructed = null;
      ReflectiveOperationException failure = null;
      try (Lease ignored = new Lease(schedule, role)) {
        try {
          constructed = construct("job-" + schedule.seed, cause);
          schedule.event(role + ":constructed");
        } catch (ReflectiveOperationException ex) {
          failure = ex;
          schedule.event(role + ":failure");
        }
      }
      schedule.event(role + ":publish");
      if ("A".equals(role)) {
        schedule.successPublished.countDown();
        schedule.failurePublished.await();
        schedule.event("A:complete");
      } else {
        schedule.failurePublished.countDown();
      }
      return new Outcome<>(constructed, cause, failure, Thread.currentThread(),
          context, Thread.currentThread() != schedule.caller);
    } finally {
      Context.LOCAL.remove();
      schedule.successPublished.countDown();
      schedule.failurePublished.countDown();
    }
  }
}
