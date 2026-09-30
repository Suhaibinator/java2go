package prereq.constant.shadow;

import prereq.constant.api.Trace;

public final class String {
    public static final java.lang.String TEXT = "sh" + "adow";

    static {
        Trace.add("S");
    }

    private final java.lang.String payload;

    public String(java.lang.String payload) {
        this.payload = payload;
    }

    public java.lang.String payload() {
        return payload;
    }

    public static java.lang.String touch() {
        return TEXT;
    }
}
