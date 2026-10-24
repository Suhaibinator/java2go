package prereq.string.contract;

public final class LiteralBank {
    public static final String CONSTANT = "lex" + "-unit";

    static {
        Ledger.initialized('L');
    }

    private LiteralBank() {}

    public static String touch() {
        return "lex-unit";
    }
}
