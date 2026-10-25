package dispatchproof;

public class ForeignPackageControl {
  public static void main(String[] args) {
    var child = new foreign.Base() {
      public int choose(Object value) {
        return 2;
      }
    };
    System.out.println(child.route(null) + ":" + child.choose(null));
  }
}
