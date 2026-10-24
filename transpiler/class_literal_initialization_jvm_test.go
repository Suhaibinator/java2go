package transpiler

import "testing"

func TestClassLiteralDoesNotInitializeSourceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/literal/plain/Only.java": `package literal.plain;
public class Only { static { System.out.println("unexpected plain initialization"); } }`,
		"src/main/java/literal/generic/Box.java": `package literal.generic;
public class Box<T> { static { System.out.println("unexpected generic initialization"); } }`,
		"src/main/java/literal/nested/Outer.java": `package literal.nested;
public class Outer {
 static { System.out.println("unexpected outer initialization"); }
 public static class Inner { static { System.out.println("unexpected inner initialization"); } }
}`,
		"src/main/java/literal/array/Element.java": `package literal.array;
public class Element { static { System.out.println("unexpected array component initialization"); } }`,
		"src/main/java/literal/app/Main.java": `package literal.app;
import literal.plain.Only;
import literal.generic.Box;
import literal.nested.Outer.Inner;
public class Main {
 public static void main(String[] args) {
  System.out.println(Only.class.getName());
  System.out.println(Box.class.getName());
  System.out.println(Inner.class.getName());
  System.out.println(literal.array.Element[][].class.getName());
  System.out.println(java.lang.String.class.getName());
 }
}`,
	}, "literal.app.Main", "literal.plain.Only\nliteral.generic.Box\nliteral.nested.Outer$Inner\n[[Lliteral.array.Element;\njava.lang.String\n")
}
