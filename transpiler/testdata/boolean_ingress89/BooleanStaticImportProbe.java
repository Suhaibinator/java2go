import static java.lang.Boolean.parseBoolean;
public class BooleanStaticImportProbe {
 static int calls;
 static String argument(String text) { calls++; return text; }
 public static int run() {
  calls = 0;
  boolean first = parseBoolean(argument("TrUe"));
  boolean missing = parseBoolean(argument(null));
  boolean spaces = parseBoolean(argument(" true "));
  return calls * 100 + (first ? 10 : 0) + (missing ? 2 : 0) + (spaces ? 1 : 0);
 }
}
