package prereq.typed.mid;

import prereq.typed.api.Base;
import prereq.typed.api.Trace;

public final class Derived extends Base {
    public static final int SHIFTED = 1 << 33;
    public static final long LONG_SHIFTED = 1L << 65;
    public static final java.lang.String FOLDED = Base.FOLDED + ":derived:"
            + SHIFTED + ":" + LONG_SHIFTED;

    static {
        Trace.add("D");
    }

    public static java.lang.String touch() {
        return FOLDED;
    }
}
