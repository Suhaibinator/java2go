package transpiler

import "testing"

func TestCampaignPrimitiveConsumersJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>primitive-consumers</artifactId><version>1</version></project>`,
		"src/main/java/owned/IntConsumer.java": `package owned;
public class IntConsumer { public void accept(int value) { System.out.println("owned=" + value); } }`,
		"src/main/java/consumerprobe/Sink.java": `package consumerprobe;
public class Sink {
 public int sum;
 public void add(int value) { sum += value; }
 public void fail(int value) { Entry.effects = Entry.effects * 10 + 3; throw new IllegalStateException("fail=" + value); }
}`,
		"src/main/java/consumerprobe/Entry.java": `package consumerprobe;
import owned.IntConsumer;
public class Entry {
 public static int effects;
 static int argument() { effects = effects * 10 + 1; return 7; }
 public static void main(String[] args) {
  Sink initial = new Sink();
  java.util.function.IntConsumer lambda = value -> initial.add(value);
  java.util.function.IntConsumer bound = initial::add;
  java.util.function.ObjIntConsumer<Sink> unbound = Sink::add;
  lambda.accept(2); bound.accept(3); unbound.accept(initial,4);
  System.out.println("sum=" + initial.sum);
  Sink mutable = initial;
  java.util.function.IntConsumer captured = mutable::add;
  mutable = new Sink(); captured.accept(1);
  System.out.println("capture=" + initial.sum + ":" + mutable.sum);
  java.util.function.IntConsumer throwing = value -> { effects = effects * 10 + 2; throw new IllegalArgumentException("bad=" + value); };
  effects = 0;
  try { throwing.accept(argument()); System.out.println("throw=missed"); } catch(IllegalArgumentException expected) { System.out.println("throw=" + expected.getMessage() + ":" + effects); }
  java.util.function.ObjIntConsumer<Sink> failure = Sink::fail;
  effects = 0;
  try { failure.accept(initial,argument()); System.out.println("fail=missed"); } catch(IllegalStateException expected) { System.out.println("fail=" + expected.getMessage() + ":" + effects); }
  Sink missing = null;
  try { java.util.function.IntConsumer invalid = missing::add; System.out.println("bound-null=missed"); } catch(NullPointerException expected) { System.out.println("bound-null=now"); }
  try { unbound.accept(null,0); System.out.println("unbound-null=missed"); } catch(NullPointerException expected) { System.out.println("unbound-null=call"); }
  IntConsumer owned = new IntConsumer(); owned.accept(5);
 }
}`,
	}, "consumerprobe.Entry", "sum=9\ncapture=10:0\nthrow=bad=7:12\nfail=fail=7:13\nbound-null=now\nunbound-null=call\nowned=5\n")
}
