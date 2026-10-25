package genericenumconstructorloop;
abstract class Base<A> { abstract A read(); abstract void write(A value); }
class Plain<A> extends Base<A> {
  A value;
  A read() { return value; }
  void write(A replacement) { value = replacement; }
}
class EnumBox<E extends Enum<E>> extends Base<E> {
  E value;
  @SuppressWarnings("unchecked")
  EnumBox(E supplied, Object replacement) {
    boolean again = true;
    for (E local = supplied; again; again = false) {
      local = (E) replacement;
      value = local;
    }
  }
  E read() { return value; }
  void write(E replacement) { value = replacement; }
}
class Token<A> {}
interface Factory { <M> Base<M> create(Token<M> token, M value); }
class Maker implements Factory {
  public <X> Base<X> create(Token<X> token, X value) {
    Plain<X> box = new Plain<X>(); box.write(value); return box;
  }
}
enum Color { RED, BLUE }
enum Other { PURPLE }
public class Main {
  public static void main(String[] args) {
    Factory factory = new Maker();
    Base<String> text = factory.create(new Token<String>(), "s");
    boolean built = false;
    boolean same = false;
    boolean delayed = false;
    try {
      EnumBox<Color> box = new EnumBox<Color>(Color.RED, Other.PURPLE);
      built = true;
      Base<?> raw = box;
      same = raw.read() == Other.PURPLE;
      Color wrong = box.read();
    } catch (ClassCastException expected) { delayed = built; }
    EnumBox<Color> empty = new EnumBox<Color>(null, null);
    boolean absent = empty.read() == null;
    System.out.println(text.read()+"|"+built+"|"+same+"|"+delayed+"|"+absent);
  }
}
