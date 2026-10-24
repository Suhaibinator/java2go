package erasedalias;

class Token { final int id; Token(int id) { this.id = id; } }
class Alien {}
final class Box<Token> {
  Token item;
  Box(Token item) { this.item = item; }
  Token get() { Main.events += "G"; return item; }
  Token qualified() { Main.events += "Q"; return this.item; }
  void set(Token item) { Main.events += "S"; this.item = item; }
}
public class Main {
  static String events = "";
  static int receiverCalls;
  static Box<Token> once(Box<Token> value) { receiverCalls++; events += "R"; return value; }
  static void check(boolean value) { if (!value) throw new AssertionError("erased alias"); }
  public static void main(String[] args) {
    Token token = new Token(7);
    Box<Token> box = new Box<>(token);
    Object value = once(box).get();
    check(value == token && receiverCalls == 1);
    check(box.qualified() == token);
    Box<? extends Token> wildcard = box;
    check(wildcard.get() == token);
    box.set(null);
    check(box.get() == null && box.qualified() == null);
    Box raw = box;
    raw.set(new Alien());
    Object polluted = box.get();
    check(polluted instanceof Alien);
    boolean cast = false;
    try { int id = box.get().id; } catch (ClassCastException expected) { cast = true; }
    check(cast);
    check(events.equals("RGQGSGQSGG"));
    System.out.println("alias-erasure-null-once-wildcard-pollution=true:" + receiverCalls + ":" + events);
  }
}
