package anonymousoverrides;

import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Iterator;
import java.util.Set;

public final class Overrides {
  static void check(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }
  public static void main(String[] args) {
    final ArrayList<String> values = new ArrayList<>(); values.add("a"); values.add("b");
    Set<String> overloaded = new AbstractSet<String>() {
      int Contains = 7;
      int HashCode = 11;
      int JavaDynamicTypeID = 13;
      int IteratorJava2goExecution = 17;
      public int size() {
        check(Contains + HashCode + JavaDynamicTypeID + IteratorJava2goExecution == 48,
            "fields preserve independent Java names");
        return values.size();
      }
      public Iterator<String> iterator() { return values.iterator(); }
      public boolean contains(String value) { throw new AssertionError("contains overload"); }
      public boolean equals(String value) { throw new AssertionError("equals overload"); }
      public int hashCode(int ignored) { throw new AssertionError("hashCode overload"); }
    };
    Set<String> peer = new AbstractSet<String>() {
      public int size() { return values.size(); }
      public Iterator<String> iterator() { return values.iterator(); }
    };
    Object erased = overloaded;
    check(overloaded.contains("a") && !overloaded.contains(null), "Object contains inherited default");
    check(erased.equals(peer) && peer.equals(erased), "Object equals inherited default");
    check(erased.hashCode() == peer.hashCode(), "Object hashCode inherited default");
    check(overloaded.remove("a") && values.size() == 1, "iterator remove default");
    overloaded.clear(); check(values.isEmpty() && overloaded.isEmpty(), "clear/isEmpty default");
    System.out.println("default-overload-field-dispatch=true");
  }
}
