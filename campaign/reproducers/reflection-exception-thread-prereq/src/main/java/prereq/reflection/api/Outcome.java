package prereq.reflection.api;

import prereq.reflection.state.Context;

public final class Outcome<T extends Exception> {
  public final T constructed;
  public final Throwable intendedCause;
  public final ReflectiveOperationException reflectiveFailure;
  public final Thread callbackThread;
  public final Context context;
  public final boolean callbackOnWorker;

  public Outcome(T constructed, Throwable intendedCause,
      ReflectiveOperationException reflectiveFailure, Thread callbackThread,
      Context context, boolean callbackOnWorker) {
    this.constructed = constructed;
    this.intendedCause = intendedCause;
    this.reflectiveFailure = reflectiveFailure;
    this.callbackThread = callbackThread;
    this.context = context;
    this.callbackOnWorker = callbackOnWorker;
  }
}
