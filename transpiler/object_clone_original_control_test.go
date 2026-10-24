package transpiler

import "testing"

func TestObjectCloneSourceHierarchyStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>cloneprobe</groupId><artifactId>cloneprobe</artifactId><version>1</version></project>`, "src/main/java/cloneprobe/Main.java": `package cloneprobe;
interface Label { int value(); default int label() { return value()+1; } }
class Base implements java.lang.Cloneable, Label {
 static int constructed;
 private final Object shared;
 private final int[] values;
 volatile int state=7;
 Base(Object shared,int[] values) { constructed++; this.shared=shared; this.values=values; }
 public int value() { return 10; }
 public Base clone() { try { return (Base) super.clone(); } catch(CloneNotSupportedException e) { throw new AssertionError(e); } }
 boolean shallow(Base other) { return shared==other.shared && values==other.values; }
}
class Child extends Base {
 int own=30;
 Child(Object shared,int[] values) { super(shared,values); }
 public int value() { return own; }
}
class Explicit extends java.lang.Object implements java.lang.Cloneable {
 int value=9;
 Explicit copy() { try { return (Explicit) super.clone(); } catch(CloneNotSupportedException e) { throw new AssertionError(e); } }
}
class Root {
 int value=14;
}
class Derived extends Root implements java.lang.Cloneable {
 Derived copy() throws CloneNotSupportedException { return (Derived)super.clone(); }
}
class Denied {
 Object copy() throws CloneNotSupportedException { return super.clone(); }
}
class Routed extends Base {
 static int calls;
 Routed(Object ref,int[] a) { super(ref,a); }
 public Routed clone() { calls++; return (Routed) super.clone(); }
}
public class Main {
 public static void main(String[] args) throws Exception {
  Object shared=new Explicit(); int[] values={2};
  Child original=new Child(shared,values);
  Base copy=original.clone();
  System.out.println((copy!=original)+":"+(copy instanceof Child)+":"+copy.shallow(original)+":"+(Base.constructed==1)+":"+copy.label());
  ((Child)copy).own=40; copy.state=8;
  System.out.println(original.label()+":"+copy.label()+":"+original.state+":"+copy.state);
  System.out.println(copy.getClass().getName());
  synchronized(original) { System.out.println(Thread.holdsLock(original)+":"+Thread.holdsLock(copy)); }
  Explicit explicit=new Explicit(); Explicit explicitCopy=explicit.copy();
  System.out.println((explicitCopy!=explicit)+":"+explicitCopy.value);
  try { new Denied().copy(); } catch(CloneNotSupportedException e) { System.out.println(e.getClass().getName()+":"+e.getMessage()); }
  Derived derived=new Derived(); Derived dc=derived.copy(); dc.value=15;
  System.out.println((derived!=dc)+":"+derived.value+":"+dc.value);
  Base anonymous=new Base(shared,values) {
   int own=55;
   public int value() { return own+values[0]; }
  };
  int count=Base.constructed;
  Base ac=anonymous.clone(); values[0]=3;
  System.out.println((anonymous!=ac)+":"+ac.shallow(anonymous)+":"+(anonymous.getClass()==ac.getClass())+":"+(Base.constructed==count)+":"+anonymous.label()+":"+ac.label());
  Routed routed=new Routed(shared,values); Routed rc=routed.clone();
  System.out.println(Routed.calls+":"+(rc!=routed)+":"+rc.shallow(routed));
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "cloneprobe.Main", "true:true:true:true:31\n31:41:7:8\ncloneprobe.Child\ntrue:false\ntrue:9\njava.lang.CloneNotSupportedException:cloneprobe.Denied\ntrue:14:15\ntrue:true:true:true:59:59\n1:true:true\n")
}

func TestObjectCloneArrayAndNominalControlsStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>cloneprobe</groupId><artifactId>controls</artifactId><version>1</version></project>`, "src/main/java/control/Main.java": `package control;
class Empty implements java.lang.Cloneable {
 Empty copy() throws CloneNotSupportedException { return (Empty) super.clone(); }
}
class Outer {
 int value=7;
 class Inner implements java.lang.Cloneable {
  Inner copy() throws CloneNotSupportedException { return (Inner) super.clone(); }
  int value() { return value; }
 }
}
public class Main {
 public static void main(String[] args) throws Exception {
  Empty original=new Empty(); Empty copy=original.copy();
  System.out.println((original!=copy)+":"+(System.identityHashCode(original)!=System.identityHashCode(copy)));
  int[] a={1,2}; int[] b=a.clone(); b[0]=9;
  System.out.println((a!=b)+":"+a[0]+":"+b[0]+":"+b.length+":"+a.clone().clone().length);
  Object ref=new Empty(); Object[] refs={ref,null}; Object[] rc=refs.clone(); rc[1]=ref;
  System.out.println((refs!=rc)+":"+(refs[0]==rc[0])+":"+(refs[1]==null)+":"+(rc[1]==ref)+":"+rc.getClass().getName());
  int[][] nested={a}; int[][] nc=nested.clone();
  System.out.println((nested!=nc)+":"+(nested[0]==nc[0]));
  int[] nil=null; try { nil.clone(); } catch(NullPointerException e) { System.out.println("null"); }
  Outer outer=new Outer(); Outer.Inner inner=outer.new Inner(); Outer.Inner ic=inner.copy(); outer.value=8;
  System.out.println((inner!=ic)+":"+inner.value()+":"+ic.value());
  final int[] capture={11};
  class Local implements java.lang.Cloneable {
   Local copy() throws CloneNotSupportedException { return (Local)super.clone(); }
   int value() { return capture[0]; }
  }
  Local local=new Local(); Local lc=local.copy(); capture[0]=12;
  System.out.println((local!=lc)+":"+local.value()+":"+lc.value());
  System.out.println(shadow.Control.check());
 }
}`,
		"src/main/java/shadow/Control.java": `package shadow;
interface Cloneable {}
class Denied implements Cloneable {
 Denied copy() throws java.lang.CloneNotSupportedException { return (Denied) super.clone(); }
}
class Object {
 static int calls;
 protected Object clone() { calls++; return this; }
}
class Child extends Object implements java.lang.Cloneable {
 Child copy() { return (Child) super.clone(); }
}
public class Control {
 public static String check() {
  Child child=new Child();
  boolean routed=child.copy()==child && Object.calls==1;
  try { new Denied().copy(); return "bad"; } catch(java.lang.CloneNotSupportedException e) {
   return routed+":"+e.getClass().getName()+":"+e.getMessage();
  }
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "control.Main", "true:true\ntrue:1:9:2:2\ntrue:true:true:true:[Ljava.lang.Object;\ntrue:true\nnull\ntrue:8:8\ntrue:12:12\ntrue:java.lang.CloneNotSupportedException:shadow.Denied\n")
}
