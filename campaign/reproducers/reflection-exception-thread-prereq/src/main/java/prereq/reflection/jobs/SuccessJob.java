package prereq.reflection.jobs;

import prereq.reflection.api.ReflectiveFactory;
import prereq.reflection.error.ProbeException;
import prereq.reflection.state.Schedule;

public final class SuccessJob extends ReflectiveFactory<ProbeException> {
  public SuccessJob(Schedule schedule) {
    super("A", ProbeException.class, schedule);
  }
}
