package deleteempty;
public final class Probe {
 static final Object lock = new Object();
 static final RuntimeException marker = new RuntimeException("conversion-marker");
 static String trace = "";
 static int targetConversions;
 static int replacementConversions;
 static boolean abruptSecond;
 static boolean failTarget;
 static boolean failReplacement;
 static String receiver(String text) { trace += "r"; return text; }
 static <C> C argument(String id, C value) { trace += id; if (abruptSecond && id.equals("b")) throw marker; return value; }
 static class Character implements CharSequence {
  final String id;
  final String value;
  Character(String id, String value) { this.id = id; this.value = value; }
  public int length() { throw new AssertionError("unexpected length callback"); }
  public char charAt(int index) { throw new AssertionError("unexpected charAt callback"); }
  public CharSequence subSequence(int start, int end) { throw new AssertionError("unexpected subsequence callback"); }
  public String toString() {
   trace += id + Thread.holdsLock(lock);
   if (id.equals("T")) { targetConversions++; if (failTarget) throw marker; }
   else { replacementConversions++; if (failReplacement) throw marker; }
   return value;
  }
 }
 static <C extends Character> String sourceOwner(String text, C target, C replacement) {
  return receiver(text).replace(argument("a", target), argument("b", replacement));
 }
 static <Character extends CharSequence> String shadowBinder(String text, Character target, Character replacement) {
  return receiver(text).replace(argument("a", target), argument("b", replacement));
 }
 static String direct(String text, CharSequence target, CharSequence replacement) {
  return receiver(text).replace(argument("a", target), argument("b", replacement));
 }
 static String units(String text) {
  if (text == null) return "null";
  String out = "";
  for (int index = 0; index < text.length(); index++) { if (index > 0) out += ","; out += (int) text.charAt(index); }
  return out;
 }
 static void show(String id, String text, CharSequence target, CharSequence replacement, boolean generic) {
  trace = ""; targetConversions = 0; replacementConversions = 0;
  try {
   String result = generic ? shadowBinder(text, target, replacement) : direct(text, target, replacement);
   System.out.println(id + "|ok|" + units(result) + "|" + (result == text) + "|" + (result == "") + "|" + trace + "|" + targetConversions + "|" + replacementConversions + "|" + Thread.holdsLock(lock));
  } catch (Throwable problem) {
   System.out.println(id + "|" + problem.getClass().getName() + "|" + (problem == marker) + "|" + trace + "|" + targetConversions + "|" + replacementConversions + "|" + Thread.holdsLock(lock));
  }
 }
 static Character target(String value) { return new Character("T", value); }
 static Character replacement(String value) { return new Character("P", value); }
 public static void main(String[] args) {
  int seed = Integer.parseInt(args[0]);
  char latin = (char) ('a' + seed % 26);
  String latinTarget = new String(new char[] { latin });
  String latinAll = new String(new char[] { latin, latin, latin, latin });
  String wideTarget = new String(new char[] { (char) (0x100 + seed) });
  String wideAll = wideTarget + wideTarget + wideTarget;
  String high = new String(new char[] { (char) 0xd800 });
  String low = new String(new char[] { (char) 0xdc00 });
  String pair = high + low;
  synchronized (lock) {
   System.out.println("seed|" + seed);
   show("latin-all", latinAll, target(latinTarget), replacement(""), false);
   show("wide-all", wideAll, target(wideTarget), replacement(""), false);
   show("pair-all", pair + pair, target(pair), replacement(""), false);
   show("high-all", high + high, target(high), replacement(""), false);
   show("low-all", low + low, target(low), replacement(""), false);
   show("nonoverlap-all", "abababab", target("ab"), replacement(""), false);
   show("overlap-partial", "ababa", target("aba"), replacement(""), false);
   show("high-partial", pair + "x" + high, target(high), replacement(""), false);
   show("low-partial", pair + "x" + low, target(low), replacement(""), false);
   show("no-match", wideAll, target("absent"), replacement(""), false);
   show("latin-no-match", latinAll, target("!"), replacement(""), false);
   show("partial", latinAll + "!" + latinAll, target(latinTarget), replacement(""), false);
   show("empty-target", latinAll, target(""), replacement(""), false);
   show("empty-receiver", new String(""), target("x"), replacement(""), false);
   show("empty-both", new String(""), target(""), replacement(""), false);
   show("target-null", latinAll, null, replacement(""), false);
   show("replacement-null", latinAll, target(latinTarget), null, false);
   show("receiver-null", null, target(latinTarget), replacement(""), false);
   show("all-null", null, null, null, false);
   abruptSecond = true;
   show("abrupt-second-null-target", latinAll, null, replacement(""), false);
   show("abrupt-second-null-receiver", null, target(latinTarget), replacement(""), false);
   abruptSecond = false;
   failTarget = true;
   show("target-conversion-failure", latinAll, target(latinTarget), replacement(""), false);
   failTarget = false; failReplacement = true;
   show("replacement-conversion-failure", latinAll, target(latinTarget), replacement(""), false);
   failReplacement = false;
   show("shadow-binding", wideAll, target(wideTarget), replacement(""), true);
   trace = ""; targetConversions = 0; replacementConversions = 0;
   String result = sourceOwner(latinAll, target(latinTarget), replacement(""));
   System.out.println("source-owner|ok|" + units(result) + "|" + (result == latinAll) + "|" + (result == "") + "|" + trace + "|" + targetConversions + "|" + replacementConversions + "|" + Thread.holdsLock(lock));
  }
 }
}
