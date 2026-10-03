package campaign.probe.base;

import campaign.probe.foreign.Foreign;

public class Base {
    private int calls;

    int token() {
        calls++;
        return 11 + calls;
    }

    public final int throughBase() {
        return token();
    }

    public final int throughCallback(Base other) {
        return other.token();
    }

    public static Base makeForeign() {
        return new Foreign();
    }
}
