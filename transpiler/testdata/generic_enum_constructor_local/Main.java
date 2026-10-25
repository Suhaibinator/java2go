package genericenumconstructor;
abstract class Base<A> { abstract A read(); abstract void write(A value); }
class Plain<A> extends Base<A> {
  A value;
  A read() { return value; }
  void write(A replacement) { value = replacement; }
}
class EnumBox<E extends Enum<E>> extends Base<E> {
  E value;
  EnumBox(E supplied) { E local = supplied; value = local; }
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
public class Main {
  public static void main(String[] args) {
    Factory factory = new Maker();
    Base<String> text = factory.create(new Token<String>(), "s");
    EnumBox<Color> box = new EnumBox<Color>(Color.RED);
    Base<Color> view = box;
    Color first = view.read();
    view.write(Color.BLUE);
    System.out.println(text.read()+"|"+first.name()+"|"+(first==Color.RED)+"|"+view.read().name());
  }
}
