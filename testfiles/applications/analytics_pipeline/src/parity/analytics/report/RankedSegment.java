package parity.analytics.report;

import parity.analytics.model.SegmentMetrics;

public class RankedSegment implements Ranked {
    private SegmentMetrics metrics;
    private int index;

    public RankedSegment(SegmentMetrics metrics) {
        this.metrics = metrics;
        this.index = metrics.getTotalScore() * 10
                + metrics.getSuccessRate()
                - metrics.getAverageLatency() / 10;
    }

    public int primaryScore() {
        return this.index;
    }

    public String stableKey() {
        return this.metrics.getSegment();
    }

    public SegmentMetrics getMetrics() {
        return this.metrics;
    }

    public int getIndex() {
        return this.index;
    }
}
