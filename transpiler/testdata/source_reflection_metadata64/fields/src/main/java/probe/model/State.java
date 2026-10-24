package probe.model;
import java.util.List;
public final class State extends Base {
 public int visible;
 private String hidden;
 protected List<String> labels;
 int packageValue;
 public static int shared=3;
 public static final int FIXED=11;
 public transient int transientValue=13;
 public volatile int volatileValue=17;
 public State(int seed){visible=seed;hidden="initial-"+seed;packageValue=seed+1;}
 public String snapshot(){return visible+"|"+hidden+"|"+packageValue+"|"+shared+"|"+inheritedValue();}
}
