package campaign.concurrentmap;

import java.util.AbstractMap;
import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import java.util.concurrent.CountDownLatch;
import java.util.function.Supplier;

/** Proposed additive application. Source-only: no frozen oracle or execution claim. */
public final class ConcurrentMapControl {
  static String trace = "";
  static final Object contextLock = new Object();
  static final ThreadLocal<String> context = new ThreadLocal<>();
  static ConcurrentMap<Object, String> callbackTarget;
  static Thread callbackThread;

  static void check(boolean condition, String name) {
    if (!condition) throw new AssertionError(name);
  }
  static void npe(Runnable action) {
    try { action.run(); } catch (NullPointerException expected) { return; }
    throw new AssertionError("missing NullPointerException");
  }
  static void cce(Runnable action) {
    try { action.run(); } catch (ClassCastException expected) { return; }
    throw new AssertionError("missing ClassCastException");
  }
  static void illegalState(Runnable action) {
    try { action.run(); } catch (IllegalStateException expected) { return; }
    throw new AssertionError("missing IllegalStateException");
  }

  static final class Key {
    final int id;
    int hashes;
    Key(int id) { this.id = id; }
    void callback() {
      check("active".equals(context.get()), "callback ThreadLocal");
      check(Thread.holdsLock(contextLock), "callback monitor ownership");
      check(Thread.currentThread() == callbackThread, "callback Thread identity");
      check("guard".equals(callbackTarget.get("guard")), "reentrant same-map read");
    }
    public int hashCode() { hashes++; callback(); return 11; }
    public boolean equals(Object other) {
      callback(); return other instanceof Key && ((Key) other).id == id;
    }
  }

  static final class Entry implements Map.Entry<String, String> {
    final String key;
    final String value;
    final boolean failValue;
    Entry(String key, String value, boolean failValue) {
      this.key = key; this.value = value; this.failValue = failValue;
    }
    public String getKey() { trace += "K"; return key; }
    public String getValue() {
      trace += "V";
      if (failValue) throw new IllegalStateException("entry value");
      return value;
    }
    public String setValue(String value) { throw new UnsupportedOperationException(); }
  }

  static final class Source extends AbstractMap<String, String> {
    final List<Map.Entry<String, String>> entries = new ArrayList<>();
    public int size() { trace += "S"; return entries.size(); }
    public Set<Map.Entry<String, String>> entrySet() {
      trace += "E";
      return new AbstractSet<Map.Entry<String, String>>() {
        public int size() { return entries.size(); }
        public Iterator<Map.Entry<String, String>> iterator() {
          trace += "I"; return entries.iterator();
        }
      };
    }
  }

  // Source membership and overrides are nominal. This implementation permits nulls.
  static final class SourceConcurrent extends AbstractMap<String, String>
      implements ConcurrentMap<String, String> {
    final HashMap<String, String> backing = new HashMap<>();
    String calls = "";
    public synchronized Set<Map.Entry<String, String>> entrySet() { return backing.entrySet(); }
    public synchronized String get(Object key) { calls += "G"; return backing.get(key); }
    public synchronized String put(String key, String value) { return backing.put(key, value); }
    public synchronized String putIfAbsent(String key, String value) {
      calls += "P"; return backing.putIfAbsent(key, value);
    }
    public synchronized void putAll(Map<? extends String, ? extends String> source) {
      calls += "A"; backing.putAll(source);
    }
    boolean same(Object left, Object right) {
      return left == right || left != null && left.equals(right);
    }
    public synchronized boolean remove(Object key, Object value) {
      if (!backing.containsKey(key) || !same(backing.get(key), value)) return false;
      backing.remove(key); return true;
    }
    public synchronized boolean replace(String key, String oldValue, String newValue) {
      if (!backing.containsKey(key) || !same(backing.get(key), oldValue)) return false;
      backing.put(key, newValue); return true;
    }
    public synchronized String replace(String key, String value) {
      return backing.containsKey(key) ? backing.put(key, value) : null;
    }
  }

  @SuppressWarnings({"rawtypes", "unchecked"})
  static void basicAndErasure() {
    ConcurrentHashMap<String, String> concrete = new ConcurrentHashMap<>();
    ConcurrentMap<String, String> map = concrete;
    Map<String, String> supertype = map;
    check(map.get("a") == null, "absent get");
    check(map.putIfAbsent("a", "one") == null, "absent insertion");
    check("one".equals(map.putIfAbsent("a", "two")), "existing insertion result");
    check("one".equals(supertype.put("a", "three")), "Map alias write");
    check("three".equals(concrete.get("a")), "same concrete allocation");
    npe(() -> map.get(null));
    npe(() -> map.putIfAbsent(null, "x"));
    npe(() -> map.putIfAbsent("x", null));
    npe(() -> map.putAll(null));
    Supplier<ConcurrentMap<String, String>> factory = ConcurrentHashMap::new;
    check(factory.get().isEmpty(), "constructor reference");
    check(ConcurrentMap.class.isAssignableFrom(ConcurrentHashMap.class), "class ancestry");
    check(ConcurrentMap.class.isInstance(concrete), "instance ancestry");

    ConcurrentMap<String, String> polluted = new ConcurrentHashMap<>();
    ConcurrentMap raw = polluted;
    raw.put("bad", Integer.valueOf(7));
    cce(() -> { String value = polluted.get("bad"); check(value != null, "unused"); });
    cce(() -> { String previous = polluted.put("bad", "fixed"); check(previous != null, "unused"); });
    check("fixed".equals(polluted.get("bad")), "cast after mutation");
    raw.put(Integer.valueOf(1), Integer.valueOf(8));
    cce(() -> { String value = polluted.get(Integer.valueOf(1)); check(value != null, "unused"); });
    check(polluted.size() == 2, "raw key retained without eager key cast");

    SourceConcurrent source = new SourceConcurrent();
    ConcurrentMap<String, String> nominal = source;
    nominal.get("missing"); nominal.putIfAbsent("x", "source");
    HashMap<String, String> input = new HashMap<>(); input.put("y", "v");
    nominal.putAll(input);
    check("GPA".equals(source.calls), "source overrides");
    check(nominal.putIfAbsent(null, null) == null, "source null policy");
    System.out.println("basic-erasure-source=true");
  }

  static void orderedPutAll() {
    ConcurrentMap<String, String> map = new ConcurrentHashMap<>();
    Source empty = new Source(); trace = ""; map.putAll(empty);
    check("SEI".equals(trace), "zero-size source traversal");
    Source nullKey = new Source();
    nullKey.entries.add(new Entry("prefix", "committed", false));
    nullKey.entries.add(new Entry(null, "value-evaluated", false));
    trace = ""; npe(() -> map.putAll(nullKey));
    check("SEIKVKV".equals(trace), "key and value getter before validation");
    check("committed".equals(map.get("prefix")) && map.size() == 1, "null partial prefix");
    Source abrupt = new Source();
    abrupt.entries.add(new Entry("second", "committed", false));
    abrupt.entries.add(new Entry("failure", "unused", true));
    trace = ""; illegalState(() -> map.putAll(abrupt));
    check("SEIKVKV".equals(trace), "abrupt getter order");
    check(map.size() == 2 && !map.containsKey("failure"), "abrupt partial prefix");
    System.out.println("putAll-order-prefix=true");
  }

  static void views() {
    ConcurrentMap<String, String> map = new ConcurrentHashMap<>();
    Set<String> keys = map.keySet();
    Collection<String> values = map.values();
    Set<Map.Entry<String, String>> entries = map.entrySet();
    map.put("one", "before");
    check(keys.size() == 1 && values.contains("before") && entries.size() == 1, "live views");
    Iterator<Map.Entry<String, String>> cursor = entries.iterator();
    map.put("one", "at-next");
    Map.Entry<String, String> entry = cursor.next();
    check("at-next".equals(entry.getValue()), "value read at next");
    map.put("one", "after-next");
    check("at-next".equals(entry.getValue()), "entry local value");
    check("at-next".equals(entry.setValue("write-through")), "entry local previous");
    check("write-through".equals(map.get("one")), "entry write through");
    map.remove("one");
    check("write-through".equals(entry.setValue("recreated")), "entry recreates removed mapping");
    npe(() -> entry.setValue(null));
    cursor.remove(); check(map.isEmpty(), "iterator remove latest mapping");
    illegalState(() -> cursor.remove());
    map.put("two", "remove"); check(values.remove("remove") && map.isEmpty(), "values removal");
    map.put("three", "clear"); keys.clear(); check(entries.isEmpty(), "view clear");
    System.out.println("live-views-entry=true");
  }

  static void callbacksAndAtomicity() throws InterruptedException {
    context.set("active"); callbackThread = Thread.currentThread();
    ConcurrentMap<Object, String> callbacks = new ConcurrentHashMap<>();
    callbackTarget = callbacks; callbacks.put("guard", "guard");
    Key first = new Key(1), alias = new Key(1), collision = new Key(2);
    synchronized (contextLock) {
      callbacks.put(first, "initial");
      check("initial".equals(callbacks.putIfAbsent(alias, "wrong")), "equal key");
      callbacks.put(collision, "collision");
      check("initial".equals(callbacks.get(alias)), "callback equality direction");
      Key rejected = new Key(3);
      npe(() -> callbacks.putIfAbsent(rejected, null));
      check(rejected.hashes == 0, "null before hash");
    }
    ConcurrentMap<String, String> map = new ConcurrentHashMap<>();
    CountDownLatch ready = new CountDownLatch(2), start = new CountDownLatch(1);
    String[] prior = new String[2]; Throwable[] failures = new Throwable[2];
    Thread left = new Thread(() -> {
      ready.countDown();
      try { start.await(); prior[0] = map.putIfAbsent("shared", "left"); }
      catch (Throwable failure) { failures[0] = failure; }
    });
    Thread right = new Thread(() -> {
      ready.countDown();
      try { start.await(); prior[1] = map.putIfAbsent("shared", "right"); }
      catch (Throwable failure) { failures[1] = failure; }
    });
    left.start(); right.start(); ready.await(); start.countDown(); left.join(); right.join();
    check(failures[0] == null && failures[1] == null, "workers succeeded");
    check((prior[0] == null) != (prior[1] == null), "exactly one insertion");
    String winner = map.get("shared"), observed = prior[0] == null ? prior[1] : prior[0];
    check(winner.equals(observed) && map.size() == 1, "atomic previous value");
    check("left".equals(winner) || "right".equals(winner), "allowed winner");
    System.out.println("callbacks-atomic=true");
  }

  public static void main(String[] args) throws InterruptedException {
    basicAndErasure(); orderedPutAll(); views(); callbacksAndAtomicity();
  }
}
