package probe.app;

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
