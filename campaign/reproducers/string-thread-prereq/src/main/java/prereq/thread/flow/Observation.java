package prereq.thread.flow;

import prereq.thread.state.WorkerContext;

public final class Observation {
  public final Thread thread;
  public final WorkerContext context;
  public final boolean onWorker;
  public final boolean equalDistinct;
  public final boolean sameIntern;
  public final boolean unicodePreserved;

  public Observation(Thread thread, WorkerContext context, boolean onWorker,
      boolean equalDistinct, boolean sameIntern, boolean unicodePreserved) {
    this.thread = thread;
    this.context = context;
    this.onWorker = onWorker;
    this.equalDistinct = equalDistinct;
    this.sameIntern = sameIntern;
    this.unicodePreserved = unicodePreserved;
  }
}
