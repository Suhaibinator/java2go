package prereq.thread.state;

/** Mutable state whose identity must remain local to one executor thread. */
public final class WorkerContext {
  public int visits;
}
