package prereq.string.contract;

public final class Ledger {
    public static final StringBuilder initialization = new StringBuilder();
    public static final StringBuilder steps = new StringBuilder();
    public static int stringifications;
    public static int selectors;

    private Ledger() {}

    public static void initialized(char label) {
        initialization.append(label);
    }

    public static String selected(String value) {
        selectors++;
        steps.append('S');
        return value;
    }
}
