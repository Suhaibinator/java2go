package prereq.string.flow;

import prereq.string.contract.Ledger;
import prereq.string.contract.LiteralBank;
import prereq.string.contract.SourceValue;

public final class StringFlow {
    public static final String CROSS_CONSTANT = LiteralBank.CONSTANT + "-flow";

    static {
        Ledger.initialized('F');
    }

    private StringFlow() {}

    private static int branch(String value) {
        return switch (Ledger.selected(value)) {
            case "alpha" -> 1;
            case "beta" -> 2;
            default -> 3;
        };
    }

    public static void exercise(int seed) {
        String first = "run-" + seed;
        String second = "run-" + seed;
        System.out.println("concat=" + first.equals(second) + ",same=" + (first == second));

        String stored = new StringBuilder().append("stored-").append(seed).toString();
        String stringValue = String.valueOf(stored);
        String objectValue = String.valueOf((Object) stored);
        String sourceValue = String.valueOf(new SourceValue(stored));
        String sourceNull = String.valueOf(new SourceValue(null));
        String nullString = String.valueOf((String) null);
        String nullObject = String.valueOf((Object) null);
        System.out.println("valueOf=" + (stringValue == stored) + "," + (objectValue == stored)
                + "," + (sourceValue == stored) + ",sourceNull=" + (sourceNull == null)
                + ",nullOverloads=" + nullString.equals("null") + "," + nullObject.equals("null"));

        StringBuilder builder = new StringBuilder().append("build-").append(seed);
        String builtOne = builder.toString();
        String builtTwo = builder.toString();
        System.out.println("builder=" + builtOne.equals(builtTwo) + ",same=" + (builtOne == builtTwo)
                + ",full=" + (builtOne.substring(0, builtOne.length()) == builtOne)
                + ",emptyConcat=" + (builtOne.concat("") == builtOne));

        char[] units = {'Q', '\uD83D', '\uDE03', 'Z'};
        String packed = new String(units);
        units[0] = 'X';
        char[] exposed = packed.toCharArray();
        exposed[3] = 'X';
        String high = packed.substring(1, 2);
        String low = packed.substring(2, 3);
        System.out.println("utf16=" + packed.length() + "," + (int) packed.charAt(0)
                + "," + (int) packed.charAt(1) + "," + (int) packed.charAt(2)
                + "," + (int) packed.charAt(3) + ",isolated=" + high.length() + ":"
                + (int) high.charAt(0) + "," + low.length() + ":" + (int) low.charAt(0));

        int alpha = branch(new String("alpha"));
        int beta = branch(new String("beta"));
        boolean nullSwitch = false;
        try {
            branch(null);
        } catch (NullPointerException expected) {
            nullSwitch = true;
        }
        System.out.println("switch=" + alpha + "," + beta + ",null=" + nullSwitch);
        System.out.println("state=" + Ledger.initialization + "," + Ledger.stringifications
                + "," + Ledger.selectors + "," + Ledger.steps);
    }
}
