package transpiler

import "testing"

func TestCampaignThrowableStructuralCollisionJVM(t *testing.T) {
	t.Run("original", func(t *testing.T) {
		runCampaignCompilerStrictProjectOracle(t, map[string]string{
			"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-structural</artifactId><version>1</version></project>
`,
			"src/main/java/probe/Lookalike.java": `package probe;
public final class Lookalike {
    public int messageCalls;
    public int nameCalls;
    public int errorCalls;
    public String throwableTypeName() { nameCalls++; return "Pretend"; }
    public String message() { messageCalls++; return "not-a-java-message"; }
    public String error() { errorCalls++; return "not-a-java-error"; }
    @Override public int hashCode() { return 42; }
}
`,
			"src/main/java/probe/Main.java": `package probe;
public final class Main {
    public static void main(String[] args) {
        Lookalike value = new Lookalike();
        Object object = value;
        String expected = object.getClass().getName() + "@2a";
        String fromValueOf = String.valueOf(object);
        String fromObject = object.toString();
        System.out.println(fromValueOf.equals(expected) + ":" + fromObject.equals(expected));
        System.out.println(value.messageCalls + ":" + value.nameCalls + ":" + value.errorCalls);
    }
}
`,
		}, "probe.Main", "true:true\n0:0:0\n")
	})
	t.Run("prefix", func(t *testing.T) {
		runCampaignCompilerStrictProjectOracle(t, map[string]string{
			"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-structural</artifactId><version>1</version></project>
`,
			"src/main/java/probe/Lookalike.java": `package probe;
public final class Lookalike {
    public int messageCalls;
    public int nameCalls;
    public int errorCalls;
    public String throwableTypeName() { nameCalls++; return "Pretend"; }
    public String message() { messageCalls++; return "not-a-java-message"; }
    public String error() { errorCalls++; return "not-a-java-error"; }
    @Override public int hashCode() { return 42; }
}
`,
			"src/main/java/probe/Main.java": `package probe;
public final class Main {
    public static void main(String[] args) {
        Lookalike value = new Lookalike();
        Object object = value;
        String prefix = object.getClass().getName() + "@";
        String fromValueOf = String.valueOf(object);
        System.out.println("valueOf.prefix=" + fromValueOf.startsWith(prefix));
        System.out.println("afterValueOf=" + value.messageCalls + ":" + value.nameCalls + ":" + value.errorCalls);
        String fromObject = object.toString();
        System.out.println("object.prefix=" + fromObject.startsWith(prefix));
        System.out.println("afterObject=" + value.messageCalls + ":" + value.nameCalls + ":" + value.errorCalls);
    }
}
`,
		}, "probe.Main", "valueOf.prefix=true\nafterValueOf=0:0:0\nobject.prefix=true\nafterObject=0:0:0\n")
	})
}
