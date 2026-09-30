package transpiler

import "testing"

func TestCampaignReceiverNames(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>receiver-names</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
class Gizmo {
 int value;
 Gizmo(int initial) { value=initial; }
 Gizmo add(int amount) { this.value += amount; return this; }
 int read() { return value; }
}
class If {
 final Gizmo target;
 If(Gizmo target) { this.target=target; }
 int update(int amount) { return target.add(amount).read(); }
}
class ΔeltaΩ {
 int value;
 ΔeltaΩ(int value) { this.value=value; }
 int read(){return value;}
}
public class Main {
 public static void main(String[] args) {
  Gizmo value=new Gizmo(7);
  If wrapper=new If(value);
  System.out.println(wrapper.update(5));
  System.out.println(value.add(-2).read());
  System.out.println(new ΔeltaΩ(19).read());
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "12\n10\n19\n")
}
