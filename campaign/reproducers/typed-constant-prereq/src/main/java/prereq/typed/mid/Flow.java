package prereq.typed.mid;

import prereq.typed.api.Trace;
import prereq.typed.shadow.String;

public final class Flow {
    public static final java.lang.String CROSS = Derived.FOLDED + ":" + String.CONSTANT;

    static {
        Trace.add("F");
    }

    private Flow() {}

    public static java.lang.String touch() {
        return CROSS;
    }

    public static java.lang.String shadowValue(int seed) {
        String sourceString = new String("source-" + seed);
        return sourceString.value();
    }
}
