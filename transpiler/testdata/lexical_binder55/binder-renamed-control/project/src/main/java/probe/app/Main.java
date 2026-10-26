package probe.app;

class Set implements Iterable<String> {
  public java.util.Iterator<String> iterator() {
    return new java.util.Iterator<String>() {
      private int position;
      public boolean hasNext() { return position < 2; }
      public String next() { position++; return position == 1 ? "a" : "bb"; }
    };
  }
}

public class Main {
  static <Item extends Iterable<String>> int throughBinder(Item source) {
    int length = 0;
    for (String value : source) length += value.length();
    return length;
  }
  public static void main(String[] args) {
    Set source = new Set();
    java.util.Set<String> builtin = new java.util.HashSet<>();
    builtin.add("a"); builtin.add("bb");
    System.out.println(throughBinder(source) + ":" + throughBinder(builtin));
  }
}
