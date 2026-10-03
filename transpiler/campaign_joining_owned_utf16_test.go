package transpiler

import "testing"

func TestCampaignJoiningCanonicalUTF16JVMParity(t *testing.T) {
	previousStrict := diagnostics.strict
	setStrictMode(true)
	t.Cleanup(func() { setStrictMode(previousStrict) })
	const source = `import java.util.Arrays;
import java.util.stream.Collectors;
public class JoiningCanonicalUTF16 {
 static String observe(String text) {
  String result = text.length() + ":" + text.hashCode() + ":";
  for (int i = 0; i < text.length(); i++) result += (int)text.charAt(i) + ",";
  return result + ";";
 }
 public static String run() {
  String[] values = new String[]{"A\uE000", null, "\uD800\0\uDFFF", "", "\uD83D", "\uDE00"};
  String[] empty = new String[0];
  String a = Arrays.stream(values).collect(Collectors.joining());
  String b = Arrays.stream(values).collect(Collectors.joining("|"));
  String c = Arrays.stream(values).collect(Collectors.joining("\0", "\uDFFF[", "]\uD800"));
  String e1 = Arrays.stream(empty).collect(Collectors.joining());
  String e2 = Arrays.stream(empty).collect(Collectors.joining());
  String e3 = Arrays.stream(empty).collect(Collectors.joining("|", "[", "]"));
  String single = new String(new char[]{'Q','\uD800'});
  String s1 = Arrays.stream(new String[]{single}).collect(Collectors.joining());
  String s2 = Arrays.stream(new String[]{single}).collect(Collectors.joining());
  return observe(a) + observe(b) + observe(c) + observe(e1) + observe(e3) + observe(s1)
    + (e1 == e2) + ":" + (e1 == "") + ":" + (s1 == single) + ":" + (s1 == s2);
 }
}`
	verifyCanonicalStringStreamOracle(t, "JoiningCanonicalUTF16", source)
}

func TestCampaignJoiningArgumentsAndCallerJVMParity(t *testing.T) {
	previousStrict := diagnostics.strict
	setStrictMode(true)
	t.Cleanup(func() { setStrictMode(previousStrict) })
	const source = `import java.util.Arrays;
import java.util.stream.Collectors;
public class JoiningArgumentsAndCaller {
 static String trace = "";
 static String argument(String label, String value, boolean fail) {
  trace += label;
  if (fail) throw new IllegalStateException(label);
  return value;
 }
 public static String run() {
  String[] empty = new String[0];
  String out = "";
  try {
   Arrays.stream(empty).collect(Collectors.joining(argument("D", null, false), argument("P", "[", false), argument("S", "]", false)));
  } catch (NullPointerException expected) { out += trace + ":N;"; }
  trace = "";
  try {
   Arrays.stream(empty).collect(Collectors.joining(argument("D", "|", false), argument("P", null, false), argument("S", "]", false)));
  } catch (NullPointerException expected) { out += trace + ":N;"; }
  trace = "";
  try {
   Arrays.stream(empty).collect(Collectors.joining(argument("D", "|", false), argument("P", "[", false), argument("S", null, false)));
  } catch (NullPointerException expected) { out += trace + ":N;"; }
  trace = "";
  try {
   Arrays.stream(empty).collect(Collectors.joining(argument("D", null, false), argument("P", "[", true), argument("S", "]", false)));
  } catch (IllegalStateException expected) { out += trace + ":I;"; }
  trace = "";
  Object lock = new Object();
  Thread caller = Thread.currentThread();
  synchronized (lock) {
   String joined = Arrays.stream(new String[]{"a","b"}).map(word -> {
    trace += (Thread.currentThread() == caller) + ":" + Thread.holdsLock(lock) + ";";
    return word;
   }).collect(Collectors.joining("|"));
   out += joined + ":" + trace;
  }
  return out;
 }
}`
	verifyCanonicalStringStreamOracle(t, "JoiningArgumentsAndCaller", source)
}
