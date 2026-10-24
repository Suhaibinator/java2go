public class BooleanSourceShadowProbe {
 static int calls;
 static class Boolean {
  static boolean parseBoolean(String text) { calls++; return text == null || text.length() == 6; }
 }
 public static int run() {
  calls = 0;
  boolean ordinary = Boolean.parseBoolean("true");
  boolean shadow = Boolean.parseBoolean("custom");
  boolean absent = Boolean.parseBoolean(null);
  return calls * 100 + (ordinary ? 10 : 0) + (shadow ? 2 : 0) + (absent ? 1 : 0);
 }
}
