class Root {
 final int marker;
 Root(int marker) { this.marker = marker; }
}
public class Main {
 static int reads;
 static String trace = "";
 static Root retained;
 static Root next(int marker) { reads++; trace += "next,"; return new Root(marker); }
 @SuppressWarnings("unchecked")
 public static <T extends Root, U extends T> void main(String[] actual) {
  int marker = Integer.parseInt(actual[0]);
  String text = actual[0];
  U selected = (U) next(marker);
  T alias = selected;
  retained = selected;
  try {
   System.out.println("body=" + actual.length + ":" + actual[0] + ":" + (text == actual[0]) + ":" + reads + ":" + selected.marker + ":" + (alias == selected) + ":" + (retained == selected) + ":" + trace);
  } finally { retained = null; trace += "cleanup,"; }
  System.out.println("final=" + (retained == null) + ":" + reads + ":" + trace);
 }
}
