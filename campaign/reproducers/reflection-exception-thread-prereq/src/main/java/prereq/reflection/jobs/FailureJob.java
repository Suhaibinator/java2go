package prereq.reflection.jobs;

import prereq.reflection.api.ReflectiveFactory;
import prereq.reflection.error.NoMatchingConstructor;
import prereq.reflection.state.Schedule;

public final class FailureJob extends ReflectiveFactory<NoMatchingConstructor> {
  public FailureJob(Schedule schedule) {
    super("B", NoMatchingConstructor.class, schedule);
  }
}
