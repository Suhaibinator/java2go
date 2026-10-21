package fieldread;
import foreignread.Value;
class Alien {}
final class Box<Value> {
  Value item;
  Box(Value item) { this.item = item; }
  Value get() { Main.events += "G"; if (Main.throwGetter) throw new IllegalStateException("getter"); return item; }
  void set(Value item) { this.item = item; }
}
public class Main {
  static String events = "";
  static int receivers;
  static boolean throwGetter;
  static Box<Value> once(Box<Value> box) { receivers++; events += "R"; return box; }
  static int later() { events += "A"; return 2; }
  static int accept(int value,int later) { events += "D"; return value + later; }
  static int read(Box<Value> box) { try { return accept(once(box).get().number,later()); } finally { events += "F"; } }
  static void reset() { events = ""; receivers = 0; throwGetter = false; }
  public static void main(String[] args) {
    Box<Value> box = new Box<>(new Value(7));
    reset(); System.out.println("valid=" + read(box) + ":" + events + ":" + receivers);
    box.set(null); reset();
    try { read(box); System.out.println("missing-nil-item"); } catch (NullPointerException expected) { System.out.println("nil-item=" + events + ":" + receivers); }
    reset();
    try { read(null); System.out.println("missing-nil-box"); } catch (NullPointerException expected) { System.out.println("nil-box=" + events + ":" + receivers); }
    Box raw = box; raw.set(new Alien()); reset();
    try { read(box); System.out.println("missing-cast"); } catch (ClassCastException expected) { System.out.println("cast=" + events + ":" + receivers); }
    reset(); throwGetter = true;
    try { read(box); System.out.println("missing-getter"); } catch (IllegalStateException expected) { System.out.println("getter=" + events + ":" + receivers); }
  }
}
