package prereq.constant.flow;

import prereq.constant.api.Constants;
import prereq.constant.api.Trace;
import prereq.constant.shadow.String;

public final class ConstantFlow {
    public static final java.lang.String FIELD_ALIAS = Constants.TEXT + "/" + String.TEXT;
    public static final java.lang.String FIELD_CHAIN = FIELD_ALIAS + "/" + Constants.FOLDED;

    static {
        Trace.add("F");
    }

    private ConstantFlow() {}

    public static java.lang.String touch() {
        return FIELD_CHAIN;
    }

    public static java.lang.String shadowPayload(int seed) {
        String shadow = new String("source-" + seed);
        return shadow.payload();
    }
}
