package transpiler

import "testing"

// Calls are explicit; this oracle makes no observation about GC or finalizer
// scheduling. Source overrides and overloads remain ordinary Java methods.
func TestObjectSuperFinalizeStrictJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>finalizeprobe</groupId><artifactId>direct</artifactId><version>1</version></project>`,
		"src/main/java/finalizeprobe/Main.java": `package finalizeprobe;
public class Main {
 static String events = "";
 static final RuntimeException marker = new IllegalStateException("sentinel");
 static void mark(String value) { events += value + ","; }
 static class Implicit {
  protected void finalize() throws Throwable { try { super.finalize(); mark("i"); } finally { mark("f"); } }
  void run() throws Throwable { finalize(); }
 }
 static class Explicit extends java.lang.Object {
  void run() throws Throwable { super.finalize(); mark("e"); }
 }
 static class Empty {}
 static class Inherited extends Empty {
  void run() throws Throwable { super.finalize(); mark("h"); }
 }
 static class Varargs {
  protected void finalize(int... values) { mark("v" + values.length); }
 }
 static class VarargsChild extends Varargs {
  void run() throws Throwable { super.finalize(); mark("z"); super.finalize(7); }
 }
 static class Parent {
  protected void finalize() throws Throwable { mark("p"); throw marker; }
 }
 static class Child extends Parent {
  protected void finalize() throws Throwable { try { super.finalize(); } finally { mark("c"); } }
  void run() throws Throwable { finalize(); }
 }
 static class Object {
  protected void finalize() throws Throwable { mark("o"); }
 }
 static class Shadow extends Object {
  void run() throws Throwable { super.finalize(); mark("s"); }
 }
 public static void main(String[] args) throws Throwable {
  new Implicit().run(); new Explicit().run(); new Inherited().run(); new VarargsChild().run();
  System.out.println(events);
  events = ""; boolean same = false;
  try { new Child().run(); } catch (Throwable got) { same = got == marker; }
  System.out.println(events + ":" + same);
  events = ""; new Shadow().run(); System.out.println(events);
  events = "";
  class Local { void run() throws Throwable { super.finalize(); mark("l"); } }
  new Local().run();
  new Empty() { void run() throws Throwable { super.finalize(); mark("a"); } }.run();
  System.out.println(events);
 }
}`,
	}, "finalizeprobe.Main", "i,f,e,h,z,v1,\np,c,:true\no,s,\nl,a,\n")
}
