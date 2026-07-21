package parity.analytics.model;

public class Event {
    private String id;
    private String segment;
    private String action;
    private int latencyMs;
    private boolean successful;
    private int units;

    public Event(String id, String segment, String action, int latencyMs, boolean successful, int units) {
        this.id = id;
        this.segment = segment;
        this.action = action;
        this.latencyMs = latencyMs;
        this.successful = successful;
        this.units = units;
    }

    public String getId() {
        return this.id;
    }

    public String getSegment() {
        return this.segment;
    }

    public String getAction() {
        return this.action;
    }

    public int getLatencyMs() {
        return this.latencyMs;
    }

    public boolean isSuccessful() {
        return this.successful;
    }

    public int getUnits() {
        return this.units;
    }
}
