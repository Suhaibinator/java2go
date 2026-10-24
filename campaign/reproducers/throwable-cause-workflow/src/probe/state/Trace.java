package probe.state;

import java.util.ArrayList;
import java.util.List;

public final class Trace {
    private final String id;
    private final List<String> events = new ArrayList<>();
    private int conversions;
    private int closes;

    public Trace(String id) {
        this.id = id;
        events.add("created@" + Thread.currentThread().getName());
    }

    public String id() {
        return id;
    }

    public synchronized String describe() {
        conversions++;
        String text = id + "@" + Thread.currentThread().getName() + "#" + conversions;
        events.add("text=" + text);
        return text;
    }

    public synchronized String describeNull() {
        conversions++;
        events.add("text=<null>@" + Thread.currentThread().getName() + "#" + conversions);
        return null;
    }

    public synchronized void describeThrowing(RuntimeException failure) {
        conversions++;
        events.add("throw=" + failure.getMessage() + "@" + Thread.currentThread().getName() + "#" + conversions);
        throw failure;
    }

    public synchronized void closed() {
        closes++;
        events.add("close@" + Thread.currentThread().getName() + "#" + closes);
    }

    public synchronized String snapshot() {
        return "calls=" + conversions + ",closes=" + closes + ",events=" + String.join("/", events);
    }
}
