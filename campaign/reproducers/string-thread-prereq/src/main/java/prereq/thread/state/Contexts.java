package prereq.thread.state;

public final class Contexts {
  public static final ThreadLocal<WorkerContext> LOCAL = ThreadLocal.withInitial(WorkerContext::new);

  private Contexts() {}
}
