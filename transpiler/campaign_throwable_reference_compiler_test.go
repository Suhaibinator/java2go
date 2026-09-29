package transpiler

import "testing"

func TestCampaignThrowableReferenceCompilerStorageJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-storage</artifactId><version>1</version></project>`,
		"src/main/java/throwprobe/Entry.java": `package throwprobe;
public class Entry {
 public static void main(String[] args) {
  String message = new String(new char[]{'x', (char)0xd800, 'y', (char)0xdc00});
  IllegalStateException failure = new IllegalStateException(message);
  Throwable view = failure;
  System.out.println("identity=" + (failure.getMessage() == message) + ":" + (view.getMessage() == message) + ":" + (view.getLocalizedMessage() == message));
  String read = view.getMessage();
  System.out.println("units=" + read.length() + ":" + (int)read.charAt(1) + ":" + (int)read.charAt(3));
  String text = view.toString();
  System.out.println("text=" + text.equals("java.lang.IllegalStateException: " + message) + ":" + (text == view.toString()));
  IllegalStateException absent = new IllegalStateException((String)null);
  System.out.println("null=" + (absent.getMessage() == null) + ":" + (absent.getLocalizedMessage() == null) + ":" + absent.toString().equals("java.lang.IllegalStateException"));
  try { throw failure; } catch (IllegalStateException caught) {
   System.out.println("caught=" + (caught.getMessage() == message));
  }
 }
}`,
	}, "throwprobe.Entry", "identity=true:true:true\nunits=4:55296:56320\ntext=true:false\nnull=true:true:true\ncaught=true\n")
}

func TestCampaignThrowableReferenceCompilerVirtualJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-virtual</artifactId><version>1</version></project>`,
		"src/main/java/throwprobe/Entry.java": `package throwprobe;
class Base extends RuntimeException {
 Base(String text) { super(text); }
 @Override public synchronized String getMessage() {
  if (!"ready".equals(Entry.context.get())) throw new IllegalStateException("wrong execution");
  Entry.calls++;
  return super.getMessage();
 }
 @Override public synchronized String getLocalizedMessage() { return getMessage(); }
 @Override public synchronized String toString() { return getLocalizedMessage(); }
 synchronized String defaultText() { return super.toString(); }
}
class Child extends Base { Child(String text) { super(text); } }
public class Entry {
 static ThreadLocal<String> context = new ThreadLocal<String>();
 static int calls;
 public static void main(String[] args) {
  context.set("ready");
  String message = new String(new char[]{'z', (char)0xd800});
  Child child = new Child(message);
  Throwable view = child;
  boolean direct = view.getMessage() == message;
  boolean localized = view.getLocalizedMessage() == message;
  boolean converted = String.valueOf((Object)child) == message;
  boolean defaulted = child.defaultText().equals("throwprobe.Child: " + message);
  System.out.println("virtual=" + direct + ":" + localized + ":" + converted + ":" + defaulted + ":" + calls);
  Exception wrapped = new Exception((Throwable)child);
  System.out.println("cause=" + (wrapped.getMessage() == message) + ":" + (wrapped.getCause() == child) + ":" + calls);
  context.remove();
 }
}`,
	}, "throwprobe.Entry", "virtual=true:true:true:true:4\ncause=true:true:5\n")
}

func TestCampaignThrowableReferenceCompilerNullCauseJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-null-cause</artifactId><version>1</version></project>`,
		"src/main/java/throwprobe/Entry.java": `package throwprobe;
class NullText extends RuntimeException {
 @Override public String toString() { return null; }
}
public class Entry {
 public static void main(String[] args) {
  NullText source = new NullText();
  Exception derived = new Exception((Throwable)source);
  System.out.println("returned-null=" + (derived.getMessage() == null) + ":" + (derived.getCause() == source) + ":" + (String.valueOf((Object)source) == null));
  Exception messageNull = new Exception((String)null);
  messageNull.initCause(source);
  Exception causeNull = new Exception((Throwable)null);
  boolean rejected = false;
  try { causeNull.initCause(source); } catch (IllegalStateException expected) { rejected = true; }
  System.out.println("overloads=" + (messageNull.getMessage() == null) + ":" + (messageNull.getCause() == source) + ":" + (causeNull.getMessage() == null) + ":" + rejected);
  String text = new String(new char[]{(char)0xdc00});
  Exception both = new Exception(text, source);
  System.out.println("both=" + (both.getMessage() == text) + ":" + (both.getCause() == source) + ":" + (int)both.getMessage().charAt(0));
 }
}`,
	}, "throwprobe.Entry", "returned-null=true:true:true\noverloads=true:true:true:true\nboth=true:true:56320\n")
}
