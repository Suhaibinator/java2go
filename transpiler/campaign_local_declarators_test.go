package transpiler

import "testing"

func TestCampaignLocalMultipleDeclarators(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>locals</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;
public class Main {
 static int calls;
 static String trace = "";
 static int next() { calls++; trace += calls; return calls; }
 static int fail() { trace += "!"; throw new IllegalArgumentException("stop"); }
 public static void main(String[] args) {
  boolean a = true, b = false;
  System.out.println(a + ":" + b);
  int first = next(), second = first + next(), third = second + next();
  System.out.println(first + ":" + second + ":" + third + ":" + trace);
  int uninitialized, ready = 7;
  uninitialized = ready + 1;
  System.out.println(uninitialized);
  Object object = "x", boxed = 2;
  System.out.println(object + ":" + boxed);
  String nullable = null, text = "kept";
  System.out.println(nullable == null);
  System.out.println(text);
  int ignored = next(), alsoIgnored = next();
  System.out.println(trace);
  try {
   int left = next(), failure = fail(), never = next();
   System.out.println(left + failure + never);
  } catch (IllegalArgumentException failure) {
   System.out.println(failure.getMessage() + ":" + trace);
  }
  int[] original = {1}, alias = original;
  alias[0] = 9;
  System.out.println(original[0]);
 }
}`,
	}, "example.Main", "true:false\n1:3:6:123\n8\nx:2\ntrue\nkept\n12345\nstop:123456!\n9\n")
}
