import java.util.UUID;
public final class Identity {
  private static String message(String input) {
    try { UUID.fromString(input); throw new AssertionError("accepted"); }
    catch (IllegalArgumentException exception) { return exception.getMessage(); }
  }
  public static void main(String[] arguments) {
    System.out.println(message("00000000-0000-0000-0000-0000000000000") == "UUID string too large");
    System.out.println(message("1--1-1-1") == "");
    System.out.println(message("x") == "Invalid UUID string: x");
    UUID value = UUID.fromString("1-1-1-1-1");
    System.out.println(value.toString() == "00000001-0001-0001-0001-000000000001");
    System.out.println(value.toString() == value.toString());
  }
}
