package transpiler

import "testing"

func TestCampaignCapitalStringOriginalProbeJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>original</artifactId><version>1</version></project>`,
		"src/main/java/probe/app/Main.java": `package probe.app;

import probe.model.CapitalText;
import probe.model.RealText;

public final class Main {
    private static void check(String label, String actual, String expected, int calls, int expectedCalls) {
        if (!expected.equals(actual) || calls != expectedCalls) {
            throw new AssertionError(label + " text=" + actual + " expected=" + expected
                    + " calls=" + calls + " expectedCalls=" + expectedCalls);
        }
        System.out.println(label + "=" + actual + ";calls=" + calls);
    }

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        CapitalText capital = new CapitalText(seed);
        Object erasedCapital = capital;
        String defaultText = "probe.model.CapitalText@" + Integer.toHexString(seed + 42);

        String direct = String.valueOf(capital);
        check("capital.value.direct", direct, defaultText, capital.hashCalls(), 1);
        check("capital.method.after.direct", "" + capital.capitalCalls(), "0", capital.capitalCalls(), 0);

        String erased = String.valueOf(erasedCapital);
        check("capital.value.erased", erased, defaultText, capital.hashCalls(), 2);
        String concatDirect = "[" + capital + "]";
        check("capital.concat.direct", concatDirect, "[" + defaultText + "]", capital.hashCalls(), 3);
        String concatErased = "[" + erasedCapital + "]";
        check("capital.concat.erased", concatErased, "[" + defaultText + "]", capital.hashCalls(), 4);
        check("capital.explicit", capital.String(), "capital:" + seed + ":1", capital.capitalCalls(), 1);
        check("capital.hash.stable", "" + capital.hashCalls(), "4", capital.hashCalls(), 4);

        RealText real = new RealText(seed);
        Object erasedReal = real;
        check("real.value.direct", String.valueOf(real), "real:" + seed + ":1", real.toStringCalls(), 1);
        check("real.value.erased", String.valueOf(erasedReal), "real:" + seed + ":2", real.toStringCalls(), 2);
        check("real.concat.direct", "[" + real + "]", "[real:" + seed + ":3]", real.toStringCalls(), 3);
        check("real.concat.erased", "[" + erasedReal + "]", "[real:" + seed + ":4]", real.toStringCalls(), 4);
    }
}
`,
		"src/main/java/probe/model/CapitalText.java": `package probe.model;

public final class CapitalText {
    private final int seed;
    private int capitalCalls;
    private int hashCalls;

    public CapitalText(int seed) {
        this.seed = seed;
    }

    // This is an ordinary Java method. It does not override Object.toString().
    public String String() {
        capitalCalls++;
        return "capital:" + seed + ":" + capitalCalls;
    }

    @Override
    public int hashCode() {
        hashCalls++;
        return seed + 42;
    }

    public int capitalCalls() {
        return capitalCalls;
    }

    public int hashCalls() {
        return hashCalls;
    }
}
`,
		"src/main/java/probe/model/RealText.java": `package probe.model;

public final class RealText {
    private final int seed;
    private int toStringCalls;

    public RealText(int seed) {
        this.seed = seed;
    }

    @Override
    public String toString() {
        toStringCalls++;
        return "real:" + seed + ":" + toStringCalls;
    }

    public int toStringCalls() {
        return toStringCalls;
    }
}
`,
		"src/main/java/probe/app/Runner.java": `package probe.app; public class Runner { public static void main(String[] args) { Main.main(new String[]{"17"}); } }`,
	}, "probe.app.Runner", "capital.value.direct=probe.model.CapitalText@3b;calls=1\ncapital.method.after.direct=0;calls=0\ncapital.value.erased=probe.model.CapitalText@3b;calls=2\ncapital.concat.direct=[probe.model.CapitalText@3b];calls=3\ncapital.concat.erased=[probe.model.CapitalText@3b];calls=4\ncapital.explicit=capital:17:1;calls=1\ncapital.hash.stable=4;calls=4\nreal.value.direct=real:17:1;calls=1\nreal.value.erased=real:17:2;calls=2\nreal.concat.direct=[real:17:3];calls=3\nreal.concat.erased=[real:17:4];calls=4\n")
}
