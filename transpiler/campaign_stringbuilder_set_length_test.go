package transpiler

import "testing"

func TestCampaignStringBuilderSetLengthJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>builder-length</artifactId><version>1</version></project>`,
		"src/main/java/owned/StringBuilder.java": `package owned;
public class StringBuilder { public void setLength(int value) { System.out.println("source=" + value); } }`,
		"src/main/java/lengthprobe/Entry.java": `package lengthprobe;
import owned.StringBuilder;
import java.util.function.IntConsumer;
import java.util.function.ObjIntConsumer;
public class Entry {
 static int effects;
 static java.lang.StringBuilder absent() { effects = effects * 10 + 1; return null; }
 static int negative() { effects = effects * 10 + 2; return -1; }
 static java.lang.String units(java.lang.String value) {
  java.lang.String result = "" + value.length();
  for (int i = 0; i < value.length(); i++) result += "," + (int)value.charAt(i);
  return result;
 }
 public static void main(java.lang.String[] args) {
  java.lang.StringBuilder value = new java.lang.StringBuilder("A😀Z");
  java.lang.String snapshot = value.toString();
  value.setLength(2);
  System.out.println("cut=" + units(value.toString()));
  value.setLength(5);
  System.out.println("grow=" + units(value.toString()));
  value.setLength(0); value.setLength(3); value.setLength(3);
  System.out.println("reset=" + units(value.toString()));
  value.append('Q'); value.setLength(2); value.setLength(4);
  System.out.println("regrow=" + units(value.toString()) + ";snapshot=" + units(snapshot));
  try { value.setLength(-1); System.out.println("negative=missed"); } catch(StringIndexOutOfBoundsException expected) { System.out.println("negative=" + expected.getMessage()); }
  try { value.setLength(Integer.MIN_VALUE); System.out.println("min=missed"); } catch(StringIndexOutOfBoundsException expected) { System.out.println("min=" + expected.getMessage()); }
  System.out.println("after-negative=" + units(value.toString()));
  value.setLength((byte)2); value.setLength((short)3); value.setLength((char)4);
  Integer boxed = Integer.valueOf(5); value.setLength(boxed);
  System.out.println("promoted=" + units(value.toString()));
  Integer missingLength = null;
  try { value.setLength(missingLength); System.out.println("boxed-null=missed"); } catch(NullPointerException expected) { System.out.println("boxed-null=now"); }
  java.lang.StringBuffer buffer = new java.lang.StringBuffer("XY");
  buffer.setLength(4);
  System.out.println("buffer-grow=" + units(buffer.toString()));
  buffer.setLength(1); buffer.setLength(3);
  System.out.println("buffer-regrow=" + units(buffer.toString()));
  try { absent().setLength(negative()); System.out.println("receiver-null=missed"); } catch(NullPointerException expected) { System.out.println("receiver-null=" + effects); }
  java.lang.StringBuilder original = value;
  IntConsumer captured = value::setLength;
  value = new java.lang.StringBuilder("Z");
  captured.accept(1);
  ObjIntConsumer<java.lang.StringBuilder> unbound = java.lang.StringBuilder::setLength;
  unbound.accept(original,3);
  System.out.println("refs=" + units(original.toString()) + ";" + units(value.toString()));
  java.lang.StringBuilder missing = null;
  try { IntConsumer invalid = missing::setLength; System.out.println("bound-null=missed"); } catch(NullPointerException expected) { System.out.println("bound-null=now"); }
  try { unbound.accept(null,0); System.out.println("unbound-null=missed"); } catch(NullPointerException expected) { System.out.println("unbound-null=call"); }
  StringBuilder source = new StringBuilder(); source.setLength(-7);
 }
}`,
	}, "lengthprobe.Entry", "cut=2,65,55357\ngrow=5,65,55357,0,0,0\nreset=3,0,0,0\nregrow=4,0,0,0,0;snapshot=4,65,55357,56832,90\nnegative=String index out of range: -1\nmin=String index out of range: -2147483648\nafter-negative=4,0,0,0,0\npromoted=5,0,0,0,0,0\nboxed-null=now\nbuffer-grow=4,88,89,0,0\nbuffer-regrow=3,88,0,0\nreceiver-null=12\nrefs=3,0,0,0;1,90\nbound-null=now\nunbound-null=call\nsource=-7\n")
}
