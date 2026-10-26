public class BooleanIngressProbe {
 static int calls, finalized;
 static String argument(String text) { calls++; return text; }
 static boolean parse(String text) {
  try { return Boolean.parseBoolean(argument(text)); } finally { finalized++; }
 }
 static <S extends String> boolean bound(S text) { return parse(text); }
 public static int run() {
  calls = 0; finalized = 0;
  String[] values = {"true", "TRUE", "TrUe", "tRuE", null, "", " true", "true ", "\ttrue", "true\n", "false", "yes", "1", "tr\uD800e", "tr\uDC00e", "tr\u0000ue", "tru\uFFFD", "tr\u200Be", "ＴＲＵＥ", "true\u00A0"};
  int flags = 0;
  for (int i = 0; i < values.length; i++) { if (parse(values[i])) { flags = flags | (1 << i); } }
  boolean qualified = java.lang.Boolean.parseBoolean("tRuE");
  boolean generic = bound("TrUe"); boolean absent = bound((String)null);
  return flags * 10000 + calls * 100 + finalized * 10 + (qualified ? 2 : 0) + (generic ? 1 : 0) + (absent ? 4 : 0);
 }
}
