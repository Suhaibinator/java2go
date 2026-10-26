package transpiler

import "testing"

func TestCampaignObjectTextBoundedHoldsLockJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>bound-holds-lock</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class Locked {
 boolean held;
 int calls;
 @Override public int hashCode() {
  calls++;
  held = Thread.holdsLock(this);
  return 42;
 }
}
public class Main {
 static <T extends Locked> String render(T value) { return String.valueOf(value); }
 public static void main(String[] args) {
  Locked value = new Locked();
  synchronized(value) { System.out.println(render(value)); }
  System.out.println(value.held + ":" + value.calls);
 }
}
`,
	}, "probe.Main", "probe.Locked@2a\ntrue:1\n")
}
