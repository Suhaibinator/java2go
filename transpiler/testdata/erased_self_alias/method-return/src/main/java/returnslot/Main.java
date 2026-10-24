package returnslot;
class Root {}
class Other extends Root {}
class Child extends Other {}
final class Outer<B extends Root,T extends B> {
 <B extends Other,U extends B> B widen(U value) { Main.effects++; return value; }
}
public class Main {
 static int effects;
 public static void main(String[] args) {
  Outer<Root,Root> outer = new Outer<>(); Child child = new Child();
  Other same = outer.<Other,Child>widen(child);
  Other missing = outer.<Other,Child>widen(null);
  System.out.println("method-return=" + (same == child) + ":" + (missing == null) + ":" + effects);
 }
}
