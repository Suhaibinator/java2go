package prereq.typed.shadow;

import prereq.typed.api.Trace;

public final class String {
    public static final java.lang.String CONSTANT = "sh" + "adow";
    public static final String INSTANCE = new String("instance");

    static {
        Trace.add("S");
    }

    private final java.lang.String value;

    public String(java.lang.String value) {
        this.value = value;
    }

    public java.lang.String value() {
        return value;
    }
}
