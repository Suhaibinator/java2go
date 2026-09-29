package transpiler

import "testing"

func TestCampaignIntegerHexRenderingJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>hex</artifactId><version>1</version></project>`,
		"src/main/java/probe/HexProbe.java": `package probe;
public final class HexProbe {
 static int calls;
 static Integer input(boolean absent) { calls++; return absent ? null : Integer.valueOf(-1); }
 public static void main(String[] args) {
  System.out.println(Integer.toHexString(0)+":"+Integer.toHexString(42)+":"+Integer.toHexString(-1));
  System.out.println(Integer.toHexString(Integer.MIN_VALUE)+":"+Integer.toHexString(Integer.MAX_VALUE));
  byte b=-1; short s=16;
  System.out.println(Integer.toHexString(b)+":"+Integer.toHexString(s)+":"+Integer.toHexString(input(false)));
  try { System.out.println(Integer.toHexString(input(true))); } catch (NullPointerException e) { System.out.println("null:"+calls); }
 }
}
`,
	}, "probe.HexProbe", "0:2a:ffffffff\n80000000:7fffffff\nffffffff:10:ffffffff\nnull:2\n")
}

func TestCampaignIntegerHexSourceOwnerJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>hex-owner</artifactId><version>1</version></project>`,
		"src/main/java/shadow/Integer.java": `package shadow;
public final class Integer { public static String toHexString(int value) { return "source:"+value; } }
`,
		"src/main/java/probe/HexOwner.java": `package probe;
import shadow.Integer;
public final class HexOwner {
 public static void main(String[] args) {
  System.out.println(Integer.toHexString(42));
  System.out.println(java.lang.Integer.toHexString(-1));
 }
}
`,
	}, "probe.HexOwner", "source:42\nffffffff\n")
}

func TestCampaignInheritedObjectTextNoReflectionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>object-text</artifactId><version>1</version></project>`,
		"src/main/java/probe/Plain.java": `package probe;
public class Plain {
 public int hashes, errors, messages, names;
 public String error(){errors++;return "error";}
 public String message(){messages++;return "message";}
 public String throwableTypeName(){names++;return "name";}
 @Override public int hashCode(){hashes++;return 42;}
}
`,
		"src/main/java/probe/Derived.java": `package probe;
public final class Derived extends Plain {
 public String toString(String... values){return "varargs:"+values.length;}
}
`,
		"src/main/java/probe/Custom.java": `package probe;
public class Custom extends Plain {
 @Override public String toString(){return "custom";}
 public String parent(){return super.toString();}
}
`,
		"src/main/java/probe/Inherited.java": `package probe;
public final class Inherited extends Custom {}
`,
		"src/main/java/probe/TextProbe.java": `package probe;
public final class TextProbe {
 static int receivers;
 static Plain receiver(Plain value){receivers++;return value;}
 public static void main(String[] args){
  Plain plain=new Plain(); Object alias=plain;
  System.out.println(receiver(plain).toString());
  System.out.println(String.valueOf(plain));
  System.out.println(alias.toString());
  System.out.println(plain.hashes+":"+plain.errors+":"+plain.messages+":"+plain.names+":"+receivers);
  Derived derived=new Derived(); Plain base=derived;
  System.out.println(derived.toString());
  System.out.println(base.toString());
  System.out.println(derived.toString("x"));
  Inherited custom=new Inherited(); Plain customBase=custom;
  System.out.println(customBase.toString()+":"+custom.toString());
  System.out.println(custom.parent());
  Plain absent=null;
  try { absent.toString(); } catch(NullPointerException e){System.out.println("null");}
 }
}
`,
	}, "probe.TextProbe", "probe.Plain@2a\nprobe.Plain@2a\nprobe.Plain@2a\n3:0:0:0:1\nprobe.Derived@2a\nprobe.Derived@2a\nvarargs:1\ncustom:custom\nprobe.Inherited@2a\nnull\n")
}

func TestCampaignInheritedObjectTextCallerExecutionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>object-monitor</artifactId><version>1</version></project>`,
		"src/main/java/probe/Locked.java": `package probe;
public final class Locked {
 public Thread owner; public int hashes; public boolean correct=true;
 @Override public synchronized int hashCode(){hashes++;correct=correct && Thread.currentThread()==owner;return 42;}
}
`,
		"src/main/java/probe/MonitorProbe.java": `package probe;
public final class MonitorProbe {
 public static void main(String[] args){
  Locked value=new Locked();value.owner=Thread.currentThread();Object alias=value;
  synchronized(value){
   System.out.println(String.valueOf(value));
   System.out.println(value.toString());
   System.out.println(alias.toString());
  }
  System.out.println(value.correct+":"+value.hashes);
 }
}
`,
	}, "probe.MonitorProbe", "probe.Locked@2a\nprobe.Locked@2a\nprobe.Locked@2a\ntrue:3\n")
}
