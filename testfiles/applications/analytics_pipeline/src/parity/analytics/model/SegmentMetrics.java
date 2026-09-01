package parity.analytics.model;

public class SegmentMetrics {
    private String segment;
    private int eventCount;
    private int successCount;
    private int totalLatency;
    private int minimumLatency;
    private int maximumLatency;
    private int totalUnits;
    private int totalScore;

    public SegmentMetrics(String segment) {
        this.segment = segment;
        this.eventCount = 0;
        this.successCount = 0;
        this.totalLatency = 0;
        this.minimumLatency = 0;
        this.maximumLatency = 0;
        this.totalUnits = 0;
        this.totalScore = 0;
    }

    public void accept(Event event, int score) {
        int latency = event.getLatencyMs();
        if (this.eventCount == 0) {
            this.minimumLatency = latency;
            this.maximumLatency = latency;
        } else {
            this.minimumLatency = Math.min(this.minimumLatency, latency);
            this.maximumLatency = Math.max(this.maximumLatency, latency);
        }

        this.eventCount++;
        if (event.isSuccessful()) {
            this.successCount++;
        }
        this.totalLatency += latency;
        this.totalUnits += event.getUnits();
        this.totalScore += score;
    }

    public String getSegment() {
        return this.segment;
    }

    public int getEventCount() {
        return this.eventCount;
    }

    public int getSuccessCount() {
        return this.successCount;
    }

    public int getFailureCount() {
        return this.eventCount - this.successCount;
    }

    public int getAverageLatency() {
        if (this.eventCount == 0) {
            return 0;
        }
        return this.totalLatency / this.eventCount;
    }

    public int getMinimumLatency() {
        return this.minimumLatency;
    }

    public int getMaximumLatency() {
        return this.maximumLatency;
    }

    public int getLatencySpread() {
        return Math.abs(this.maximumLatency - this.minimumLatency);
    }

    public int getTotalUnits() {
        return this.totalUnits;
    }

    public int getTotalScore() {
        return this.totalScore;
    }

    public int getSuccessRate() {
        if (this.eventCount == 0) {
            return 0;
        }
        return this.successCount * 100 / this.eventCount;
    }
}
