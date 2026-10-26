package campaign.concurrent.service;

import java.util.HashMap;
import java.util.Map;

public final class Ledger {
    private final Map<String, String> states = new HashMap<>();
    private final Map<String, String> owners = new HashMap<>();
    private int started;
    private int finished;

    public synchronized void queue(String id) {
        if (states.putIfAbsent(id, "QUEUED") != null) {
            throw new AssertionError("duplicate job " + id);
        }
    }

    public synchronized void start(String id) {
        change(id, "QUEUED", "RUNNING");
        owners.put(id, Thread.currentThread().getName());
        started++;
    }

    public synchronized void succeed(String id) {
        change(id, "RUNNING", "SUCCEEDED");
        finished++;
    }

    public synchronized void fail(String id) {
        change(id, "RUNNING", "FAILED");
        finished++;
    }

    public synchronized void cancel(String id) {
        change(id, "QUEUED", "CANCELLED");
    }

    private void change(String id, String before, String after) {
        String actual = states.get(id);
        if (!before.equals(actual)) {
            throw new AssertionError(id + ": expected " + before + ", found " + actual);
        }
        states.put(id, after);
    }

    public synchronized void assertState(String id, String expected) {
        if (!expected.equals(states.get(id))) {
            throw new AssertionError(id + ": expected " + expected + ", found " + states.get(id));
        }
        String owner = owners.get(id);
        if ("CANCELLED".equals(expected)) {
            if (owner != null) {
                throw new AssertionError("cancelled task ran: " + id);
            }
        } else if (owner == null || owner.equals(Thread.currentThread().getName())) {
            throw new AssertionError("task did not run on a worker: " + id + " owner=" + owner);
        }
    }

    public synchronized void assertCounts(int expectedStarted, int expectedFinished) {
        if (started != expectedStarted || finished != expectedFinished) {
            throw new AssertionError("counts " + started + "/" + finished);
        }
    }
}
