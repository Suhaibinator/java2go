package dispatchproof;

public class AliasCollisionWitness {
  interface PublicRole {
    int choose(Object value);
  }

  static abstract class LowerGate {
    abstract int choose(Object value);

    int route(Object value) {
      return choose(value);
    }
  }

  static abstract class Both extends LowerGate implements PublicRole {}

  public static void main(String[] args) {
    var both = new Both() {
      public int Choose() {
        return 9;
      }

      public int choose(Object value) {
        return value == null ? 5 : 7;
      }
    };
    PublicRole role = both;
    System.out.println(both.Choose() + ":" + both.route(null) + ":" + role.choose(new Object()));
  }
}
