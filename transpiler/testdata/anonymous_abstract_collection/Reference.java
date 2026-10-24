package anonymousreference;

import java.util.AbstractSet;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Iterator;
import java.util.Set;

public final class Reference {
  static String trace = "";
  static Set<String> shared;
  static void check(boolean condition, String message) {
    if (!condition) throw new AssertionError(message);
  }
  static Set<String> left() { trace += "L"; return shared; }
  static Collection<String> right() { trace += "R"; return shared; }
  static class Base extends AbstractSet<String> {
    public int size() { return 0; }
    public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
  }
  static final class Child extends Base {}
  public static void main(String[] args) {
    Set<String> a = new AbstractSet<String>() {
      public int size() { return 0; }
      public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
    };
    Set<String> b = new AbstractSet<String>() {
      public int size() { return 0; }
      public Iterator<String> iterator() { return new ArrayList<String>().iterator(); }
    };
    check(a != b && a.equals(b), "distinct allocation vs Set value equality");
    Collection<String> collection = a;
    AbstractSet<String> base = (AbstractSet<String>) a;
    Object erased = a;
    check(a == collection && base == a && erased == collection, "static aliases");
    Set<String> nullSet = null;
    Collection<String> nullCollection = nullSet;
    check(nullSet == nullCollection && nullSet != a && a != null, "typed null");
    Child child = new Child(); Base parent = child; Collection<String> inherited = child;
    check(child == parent && inherited == parent && inherited == child, "source allocation aliases");
    shared = a;
    check(left() == right() && trace.equals("LR"), "single left-to-right evaluation");
    Integer boxed = 7;
    check(boxed == 7 && 7 == boxed && 3 != 4, "primitive/unboxing path");
    Set<String>[] array = new Set[1]; Object arrayAlias = array;
    check(array == arrayAlias && array != null, "array identity outside scalar helper");
    System.out.println("collection-reference-identity-order=true");
  }
}
