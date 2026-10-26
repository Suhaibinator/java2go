package anonymouscollections;

import java.util.AbstractCollection;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;
import java.util.Set;

public final class Main {
  static final Object lock = new Object();
  static final ThreadLocal<String> context = new ThreadLocal<>();
  static Thread owner;
  static String trace = "";
  static boolean active;
  static Collection<?> removalInput;
  static void check(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }
  static void callback(String event) {
    if (active) {
      check(Thread.currentThread() == owner, "caller Thread");
      check(Thread.holdsLock(lock), "caller monitor");
      check("live".equals(context.get()), "caller ThreadLocal");
    }
    trace += event;
  }
  static final class Named extends AbstractSet<String> {
    final ArrayList<String> values;
    Named(ArrayList<String> values) { this.values = values; }
    public int size() { callback("S"); return values.size(); }
    public Iterator<String> iterator() { callback("I"); return values.iterator(); }
  }
  static final class Box<T> {
    final ArrayList<T> values = new ArrayList<>();
    Collection<T> view() {
      return new AbstractCollection<T>() {
        public int size() { return values.size(); }
        public Iterator<T> iterator() { return values.iterator(); }
      };
    }
  }
  static final class Shape {
    public int size() { return 0; }
    public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
  }
  static Set<String> anonymous(ArrayList<String> values) {
    return new AbstractSet<String>() {
      public int size() { callback("S"); return values.size(); }
      public Iterator<String> iterator() { callback("I"); return values.iterator(); }
    };
  }
  static void defaults() {
    ArrayList<String> values = new ArrayList<>(); values.add("a"); values.add("b");
    ArrayList<String> peerValues = new ArrayList<>(); peerValues.add("b"); peerValues.add("a");
    Set<String> view = anonymous(values);
    Set<String> named = new Named(peerValues);
    Object erased = view;
    check(erased instanceof AbstractSet && erased instanceof AbstractCollection,
        "canonical superclass ancestry");
    check(erased instanceof Set && erased instanceof Collection && erased instanceof Iterable,
        "canonical interface ancestry");
    check(!((Object) new Shape() instanceof Set), "shape is not membership");
    check(!view.isEmpty() && view.contains("a") && !view.contains(null), "collection defaults");
    check(view.equals(named) && named.equals(view), "set equality");
    check(view.hashCode() == named.hashCode(), "set hash");
    check(view.containsAll(named), "containsAll");
    check(!view.equals(new Shape()) && !view.equals(null), "set equality negatives");
    check(view.remove("a") && !view.remove("missing") && values.size() == 1,
        "remove forwards iterator");
    view.clear(); check(values.isEmpty() && view.isEmpty(), "clear forwards iterator");
    Box<String> box = new Box<>(); box.values.add("captured");
    Collection<String> generic = box.view();
    check(generic.contains("captured") && "captured".equals(generic.iterator().next()),
        "hidden generic capture");
    System.out.println("ancestry-defaults-generic=true");
  }
  static void overrides() {
    ArrayList<String> values = new ArrayList<>(); values.add("a");
    Set<String> special = new AbstractSet<String>() {
      public int size() { callback("S"); return values.size(); }
      public Iterator<String> iterator() { callback("I"); return values.iterator(); }
      public boolean contains(Object value) { callback("C"); return "virtual".equals(value); }
      public boolean isEmpty() { callback("Z"); return true; }
      public int hashCode() { callback("H"); return 73; }
    };
    ArrayList<String> other = new ArrayList<>(); other.add("virtual");
    trace = "";
    check(special.containsAll(new Named(other)), "default uses override contains");
    check(trace.equals("IC"), "containsAll callback order: " + trace);
    trace = "";
    check(special.isEmpty() && special.hashCode() == 73 && trace.equals("ZH"),
        "direct default override");
    Set<String> equalsOverride = new AbstractSet<String>() {
      public int size() { throw new AssertionError("equals override size"); }
      public Iterator<String> iterator() { throw new AssertionError("equals override iterator"); }
      public boolean equals(Object value) { return value == this; }
    };
    Object alias = equalsOverride;
    check(alias.equals(equalsOverride) && !alias.equals(special), "Object alias override");
    System.out.println("overrides-order=true");
  }
  static void contextAndMutation() {
    owner = Thread.currentThread(); context.set("live"); active = true;
    synchronized (lock) {
      ArrayList<String> values = new ArrayList<>(); values.add("a"); values.add("b");
      Set<String> set = anonymous(values);
      trace = "";
      check(!set.isEmpty() && set.contains("b"), "context defaults");
      check(trace.equals("SI"), "isEmpty then contains callbacks");
      Set<String> inheritedRemoval = new AbstractSet<String>() {
        public int size() { callback("S"); return values.size(); }
        public Iterator<String> iterator() { callback("I"); return values.iterator(); }
        // A source override invokes the inherited AbstractSet algorithm. Static
        // Set.removeAll lowering is a separately recorded existing frontier.
        public boolean isEmpty() { return removeAll(removalInput); }
      };
      ArrayList<String> one = new ArrayList<>(); one.add("a");
      removalInput = new Named(one);
      check(inheritedRemoval.isEmpty() && values.size() == 1,
          "removeAll larger self branch");
      values.add("a");
      ArrayList<String> many = new ArrayList<>(); many.add("a"); many.add("x"); many.add("y");
      removalInput = new Named(many);
      check(inheritedRemoval.isEmpty() && values.size() == 1 && "b".equals(values.get(0)),
          "removeAll iterator self branch");
      ArrayList<String> partial = new ArrayList<>(); partial.add("a"); partial.add("b");
      Collection<String> failure = new AbstractCollection<String>() {
        public int size() { callback("S"); return partial.size(); }
        public Iterator<String> iterator() {
          callback("I"); final Iterator<String> delegate = partial.iterator();
          return new Iterator<String>() {
            int removed;
            public boolean hasNext() { callback("h"); return delegate.hasNext(); }
            public String next() { callback("n"); return delegate.next(); }
            public void remove() {
              callback("r");
              if (removed == 1) throw new IllegalStateException("second remove");
              removed++; delegate.remove();
            }
          };
        }
      };
      trace = "";
      try { failure.clear(); throw new AssertionError("missing iterator exception"); }
      catch (IllegalStateException expected) {
        check(partial.size() == 1 && "b".equals(partial.get(0)), "clear partial effects");
        check(trace.equals("Ihnrhnr"), "clear exact callback order: " + trace);
      }
    }
    active = false; context.remove();
    System.out.println("execution-mutation-prefix=true");
  }
  static void selectorsAndIdentity() {
    Set<String> fields = new AbstractSet<String>() {
      int Contains = 7;
      int HashCode = 11;
      int JavaDynamicTypeID = 13;
      int IteratorJava2goExecution = 17;
      public int size() { return Contains + HashCode + JavaDynamicTypeID + IteratorJava2goExecution; }
      public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
    };
    check(fields.size() == 48 && !fields.contains("missing"), "inherited selector storage");
    Set<String> left = new AbstractSet<String>() {
      public int size() { return 0; }
      public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
    };
    Set<String> right = new AbstractSet<String>() {
      public int size() { return 0; }
      public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
    };
    Object alias = left;
    check(left != right && alias == left && left.equals(right), "allocation vs set equality");
    System.out.println("selectors-allocation=true");
  }
  public static void main(String[] args) {
    defaults(); overrides(); contextAndMutation(); selectorsAndIdentity();
  }
}
