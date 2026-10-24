package prereq.typed.api;

public class Base {
    public static final int SHIFTED = 1 << 32;
    public static final byte NARROW = (byte) (127 + SHIFTED);
    public static final java.lang.String FOLDED = "base:" + SHIFTED + ":" + NARROW;

    static {
        Trace.add("B");
    }

    protected Base() {}
}
