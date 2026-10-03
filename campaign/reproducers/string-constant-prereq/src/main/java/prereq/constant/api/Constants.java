package prereq.constant.api;

public final class Constants {
    public static final java.lang.String TEXT = "lex" + "ical";
    public static final int OVERFLOW = 2147483647 + 1;
    public static final int INT_SHIFT = 1 << 32;
    public static final long LONG_SHIFT = 1L << 64;
    public static final char LETTER = (char) ('A' + 1);
    public static final boolean FLAG = (3 * 7 == 21) && !false;
    public static final float FRACTION = 1.25f + 0.5f;
    public static final java.lang.String FOLDED = TEXT + ":" + OVERFLOW + ":"
            + INT_SHIFT + ":" + LONG_SHIFT + ":" + LETTER + ":" + FLAG
            + ":" + FRACTION;
    public static final java.lang.String RUNTIME = new StringBuilder()
            .append("lex").append("ical").toString();

    static {
        Trace.add("C");
    }

    private Constants() {}

    public static java.lang.String touch() {
        return TEXT;
    }
}
