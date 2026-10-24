package prereq.reflection.state;

/** Mutable callback state, owned by exactly one executor thread. */
public final class Context {
  public static final ThreadLocal<Context> LOCAL = ThreadLocal.withInitial(Context::new);
  public int visits;
}
