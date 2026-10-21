package anonymouscontext;

import java.util.AbstractCollection;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;
import java.util.Set;

public final class Context {
  static final Object lock = new Object();
  static final ThreadLocal<String> local = new ThreadLocal<>();
  static Thread owner;
  static String trace = "";
  static void check(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }
  static void callback() {
    check(Thread.currentThread() == owner, "caller Thread");
    check(Thread.holdsLock(lock), "caller monitor");
    check("same".equals(local.get()), "caller ThreadLocal");
  }
  static int fromTry() {
    try {
      synchronized (lock) {
        final ArrayList<String> values = new ArrayList<>(); values.add("a");
        Set<String> view = new AbstractSet<String>() {
          public int size() { callback(); return values.size(); }
          public Iterator<String> iterator() { callback(); return values.iterator(); }
        };
        check(!view.isEmpty() && view.contains("a"), "anonymous inside try and monitor");
        return 7;
      }
    } finally { trace += "F"; }
  }
  static int fromLoop() {
    int index = 0;
    outer: do {
      index++; final int captured = index;
      Collection<String> view = new AbstractCollection<String>() {
        public int size() {
          callback();
          inner: do { if (captured > 0) return captured; break inner; } while (false);
          return 0;
        }
        public Iterator<String> iterator() { callback(); return new ArrayList<String>().iterator(); }
      };
      check(view.size() == index, "method return vs outer labeled loop");
      if (index == 1) continue outer;
      if (index == 2) break outer;
    } while (index < 3);
    return index;
  }
  static int fromException() {
    try {
      Collection<String> view = new AbstractCollection<String>() {
        public int size() { callback(); return 1; }
        public Iterator<String> iterator() {
          callback(); throw new IllegalStateException("source iterator");
        }
      };
      check(!view.isEmpty(), "return inside outer try");
      view.contains("a");
      return 99;
    } catch (IllegalStateException expected) { return 11; }
    finally { trace += "X"; }
  }
  public static void main(String[] args) {
    owner = Thread.currentThread(); local.set("same");
    check(fromTry() == 7 && !Thread.holdsLock(lock), "outer return and monitor exit");
    synchronized (lock) {
      check(fromLoop() == 2, "outer break and continue preserved");
      check(fromException() == 11, "outer catch return and finally preserved");
    }
    check(!Thread.holdsLock(lock) && trace.equals("FX"), "outer finalization order");
    local.remove();
    System.out.println("nested-method-boundaries-execution=true");
  }
}
