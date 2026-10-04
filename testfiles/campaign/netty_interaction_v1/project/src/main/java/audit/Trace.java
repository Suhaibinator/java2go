package audit;
import java.util.ArrayList;
import java.util.List;
public final class Trace {
 public final List<String> events = new ArrayList<String>();
 public void add(String text) { events.add(text); }
 public String joined() {
  StringBuilder b = new StringBuilder();
  for (String e : events) { if (b.length() != 0) b.append(';'); b.append(e); }
  return b.toString();
 }
 public static void record(String name, String value) { System.out.println(name + "=" + value); }
 public static void require(boolean test, String label) { if (!test) throw new IllegalStateException(label); }
}
