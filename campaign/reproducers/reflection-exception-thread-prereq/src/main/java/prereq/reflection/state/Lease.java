package prereq.reflection.state;

/** Cleanup is observable on both successful and failed reflective construction. */
public final class Lease implements AutoCloseable {
  private final Schedule schedule;
  private final String role;
  private boolean closed;

  public Lease(Schedule schedule, String role) {
    this.schedule = schedule;
    this.role = role;
    schedule.event(role + ":open");
  }

  @Override
  public void close() {
    if (closed) throw new AssertionError("double close");
    closed = true;
    schedule.closed.incrementAndGet();
    schedule.event(role + ":closed");
  }
}
