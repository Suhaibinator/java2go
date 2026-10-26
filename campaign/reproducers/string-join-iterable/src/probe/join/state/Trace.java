package probe.join.state;

import java.util.ArrayList;
import java.util.List;

public final class Trace {
    private final List<String> events = new ArrayList<>();

    public void add(String event) {
        events.add(event);
    }

    public String snapshot() {
        StringBuilder out = new StringBuilder();
        for (int i = 0; i < events.size(); i++) {
            if (i != 0) {
                out.append('/');
            }
            out.append(events.get(i));
        }
        return out.toString();
    }
}
