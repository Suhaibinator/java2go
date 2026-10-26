package anonymousshadow;

class AbstractSet {
  int contains() { return 9; }
}
final class Box<AbstractSet> {
  final AbstractSet item;
  Box(AbstractSet item) { this.item = item; }
  AbstractSet get() { return item; }
}
public final class Shadow {
  static void check(boolean condition) {
    if (!condition) throw new AssertionError("canonical shadow");
  }
  public static void main(String[] args) {
    AbstractSet source = new AbstractSet() {
      int contains() { return 17; }
    };
    Box<AbstractSet> box = new Box<>(source);
    Object value = box.get();
    check(source.contains() == 17 && value == source);
    check(!(value instanceof java.util.Collection) && !(value instanceof java.util.Set));
    java.util.Set<String> canonical = new java.util.AbstractSet<String>() {
      public int size() { return 0; }
      public java.util.Iterator<String> iterator() {
        return new java.util.ArrayList<String>().iterator();
      }
    };
    check(canonical.isEmpty() && canonical instanceof java.util.AbstractCollection);
    System.out.println("source-binder-and-qualified-owner=true");
  }
}
