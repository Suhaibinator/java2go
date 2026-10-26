package app;
import audit.Trace;
import buffers.CompositeFlow;
import flow.Interaction;
import io.netty.util.internal.logging.InternalLoggerFactory;
import io.netty.util.internal.logging.JdkLoggerFactory;
public final class Main {
 public static void main(String[] args) throws Exception {
  int seed = Integer.parseInt(args[0]);
  InternalLoggerFactory.setDefaultFactory(JdkLoggerFactory.INSTANCE);
  Trace.record("seed", Integer.toString(seed));
  CompositeFlow.run(seed);
  new Interaction(seed).run();
  Trace.record("done", "complete");
 }
}
