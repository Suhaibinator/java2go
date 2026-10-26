package transpiler

import "testing"

func TestCampaignImportedFieldQualifier44InitializationJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Owner.java": `package library;
public class Owner {
 public static Node Node = init();
 static Node init() { System.out.print("owner:"); return new Node(); }
 public static class Node { public static int count = 3; }
}`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Owner.Node;
public class Main {
 public static void main(String[] args) { System.out.println(Node.count); }
}`,
	}, "probe.Main", "owner:3\n")
}
