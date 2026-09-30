package p;
public final class Overloaded {
  public static final class Token<T> {}
  @SuppressWarnings("unchecked")
  public <T> T fromJson(String json, Class<T> type) { return (T) json; }
  @SuppressWarnings("unchecked")
  public <T> T fromJson(String json, Token<T> type) { return (T) json; }
  public static void main(String[] args) {
    Overloaded codec = new Overloaded();
    String left = codec.fromJson("a", String.class);
    String right = codec.fromJson("b", new Token<String>());
    System.out.println(left + right);
  }
}
