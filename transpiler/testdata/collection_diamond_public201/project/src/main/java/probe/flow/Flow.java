package probe.flow;

import java.util.Collection;
import probe.model.State;

public final class Flow {
    public static Collection<String> alias(Collection<String> values) { return values; }
    public static int digest(Collection<String> values, State state) {
        for (String value : values) state.accept(value);
        return state.checksum();
    }
}
