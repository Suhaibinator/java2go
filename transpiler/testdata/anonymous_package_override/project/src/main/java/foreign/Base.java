package foreign;

public class Base {
  int choose(Object value) {
    return 1;
  }

  public int route(Object value) {
    return choose(value);
  }
}
