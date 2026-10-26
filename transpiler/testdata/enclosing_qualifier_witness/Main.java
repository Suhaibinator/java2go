class State {
 static String trace = "";
 static int reads;
 static int sourceInitializations;
 static int mark(String tag, int value) { trace += tag + ","; return value; }
 static String observe(String label, int value) { return label + "=" + value + ";reads=" + reads + ";source=" + sourceInitializations + ";trace=" + trace + "\n"; }
}
class Normalizer {
 static class Form {
  static int NFC = initialize();
  static int initialize() { State.sourceInitializations++; return State.mark("source-form", 23); }
 }
}
class HolderForm {
 int NFC;
 HolderForm(String tag, int value) { NFC = State.mark(tag, value); }
}
class Holder {
 HolderForm Form;
 Holder(String tag, int value) { Form = new HolderForm(tag, value); }
}
class InstanceOuter {
 Holder Normalizer = new Holder("instance-holder", 31);
 class Inner {
  int read() { State.reads++; return Normalizer.Form.NFC; }
 }
 int read() { return new Inner().read(); }
}
class StaticOuter {
 static Holder Normalizer = new Holder("static-holder", 41);
 static class Inner {
  int read() { State.reads++; return Normalizer.Form.NFC; }
 }
 static int read() { return new Inner().read(); }
}
public class Main {
 static String instance() {
  InstanceOuter outer = new InstanceOuter();
  Holder alias = outer.Normalizer;
  String result;
  try {
   result = State.observe("instance-first", outer.read());
   alias.Form.NFC = 32;
   result += State.observe("instance-second", outer.read());
   result += "alias=" + (alias == outer.Normalizer) + "\n";
  } finally { outer.Normalizer = null; State.trace += "instance-cleanup,"; }
  return result + "final=" + (outer.Normalizer == null) + ";retained=" + alias.Form.NFC + ";source=" + State.sourceInitializations + ";trace=" + State.trace + "\n";
 }
 static String statics() {
  Holder alias = StaticOuter.Normalizer;
  String result;
  try {
   result = State.observe("static-first", StaticOuter.read());
   alias.Form.NFC = 42;
   result += State.observe("static-second", StaticOuter.read());
   result += "alias=" + (alias == StaticOuter.Normalizer) + "\n";
  } finally { StaticOuter.Normalizer = null; State.trace += "static-cleanup,"; }
  return result + "final=" + (StaticOuter.Normalizer == null) + ";retained=" + alias.Form.NFC + ";source=" + State.sourceInitializations + ";trace=" + State.trace + "\n";
 }
 static String priorLocal() {
  State.reads++;
  String result = State.observe("type-before-local", Normalizer.Form.NFC);
  Holder Normalizer = new Holder("local-holder", 51);
  Holder alias = Normalizer;
  try {
   State.reads++;
   result += State.observe("value-after-local", Normalizer.Form.NFC);
   Normalizer.Form.NFC = 52;
   State.reads++;
   result += State.observe("value-after-write", Normalizer.Form.NFC);
   result += "alias=" + (alias == Normalizer) + "\n";
  } finally { Normalizer = null; State.trace += "local-cleanup,"; }
  return result + "final=" + (Normalizer == null) + ";retained=" + alias.Form.NFC + ";source=" + State.sourceInitializations + ";trace=" + State.trace + "\n";
 }
 public static String run(String mode) {
  if (mode.equals("instance")) return instance();
  if (mode.equals("static")) return statics();
  if (mode.equals("priorlocal")) return priorLocal();
  throw new IllegalArgumentException("unknown mode: " + mode);
 }
 public static void main(String[] args) { System.out.print(run(args[0])); }
}
