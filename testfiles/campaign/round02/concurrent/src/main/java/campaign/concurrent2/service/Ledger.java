package campaign.concurrent2.service;

import java.util.HashMap;
import java.util.Map;

public final class Ledger {
    private final Map<String, String> states = new HashMap<>();
    private final Map<String, Object> contexts = new HashMap<>();
    private final Map<String, String> workers = new HashMap<>();
    private final String callerName = Thread.currentThread().getName();
    private int started;
    private int finished;
    private int opened;
    private int closed;

    public synchronized void queue(String id) {
        if (states.putIfAbsent(id, "QUEUED") != null) {
            throw new AssertionError("duplicate job " + id);
        }
    }

    public synchronized void start(String id, Object context) {
        transition(id, "QUEUED", "RUNNING");
        String owner = Thread.currentThread().getName();
        if (owner.equals(callerName)) {
            throw new AssertionError("job ran on caller: " + id);
        }
        workers.put(id, owner);
        for (Object prior : contexts.values()) {
            if (prior == context) {
                throw new AssertionError("worker-local context reused: " + id);
            }
        }
        contexts.put(id, context);
        started++;
    }

    public synchronized void succeed(String id) {
        transition(id, "RUNNING", "SUCCEEDED");
        finished++;
    }

    public synchronized void fail(String id) {
        transition(id, "RUNNING", "FAILED");
        finished++;
    }

    public synchronized void cancel(String id) {
        transition(id, "QUEUED", "CANCELLED");
    }

    private void transition(String id, String from, String to) {
        if (!from.equals(states.get(id))) {
            throw new AssertionError(id + " transition from " + states.get(id) + " expected " + from);
        }
        states.put(id, to);
    }

    public synchronized void opened() { opened++; }
    public synchronized void closed() { closed++; }

    public synchronized void assertComplete(int expectedStarted, String[] succeeded,
                                            String failed, String cancelled) {
        for (String id : succeeded) {
            assertState(id, "SUCCEEDED", true);
        }
        if (failed != null) { assertState(failed, "FAILED", true); }
        if (cancelled != null) { assertState(cancelled, "CANCELLED", false); }
        if (started != expectedStarted || finished != expectedStarted ||
            opened != expectedStarted || closed != expectedStarted) {
            throw new AssertionError("counts " + started + "/" + finished + "/" + opened + "/" + closed);
        }
    }

    private void assertState(String id, String expected, boolean ran) {
        if (!expected.equals(states.get(id))) {
            throw new AssertionError(id + " state " + states.get(id));
        }
        if (ran != (contexts.get(id) != null) || ran != (workers.get(id) != null)) {
            throw new AssertionError(id + " ownership mismatch");
        }
    }
}
