package prereq.string.app;

import prereq.string.contract.Ledger;
import prereq.string.contract.LiteralBank;
import prereq.string.flow.StringFlow;

public final class Main {
    private Main() {}

    public static void main(String[] args) {
        int seed = Integer.parseInt(args[0]);
        boolean constants = LiteralBank.CONSTANT == "lex-unit"
                && StringFlow.CROSS_CONSTANT == "lex-unit-flow";
        System.out.println("seed=" + seed + ",constants=" + constants
                + ",before=" + Ledger.initialization.length());
        String touched = LiteralBank.touch();
        System.out.println("literal=" + (touched == LiteralBank.CONSTANT)
                + ",afterTouch=" + Ledger.initialization);
        StringFlow.exercise(seed);
    }
}
