package parity.analytics.pipeline;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import parity.analytics.common.ParseResult;
import parity.analytics.model.Event;
import parity.analytics.model.SegmentMetrics;

public class AnalyticsResult {
    private int sourceRows;
    private List<Event> events;
    private List<Rejection> rejections;
    private List<String> segmentOrder;
    private Map<String, SegmentMetrics> metricsBySegment;
    private int totalSuccesses;
    private int totalLatency;
    private int totalUnits;
    private int totalScore;
    private int checksum;

    public AnalyticsResult(int sourceRows) {
        this.sourceRows = sourceRows;
        this.events = new ArrayList<Event>();
        this.rejections = new ArrayList<Rejection>();
        this.segmentOrder = new ArrayList<String>();
        this.metricsBySegment = new HashMap<String, SegmentMetrics>();
        this.totalSuccesses = 0;
        this.totalLatency = 0;
        this.totalUnits = 0;
        this.totalScore = 0;
        this.checksum = 17;
    }

    public void recordAccepted(Event event, int score) {
        this.events.add(event);
        if (event.isSuccessful()) {
            this.totalSuccesses++;
        }
        this.totalLatency += event.getLatencyMs();
        this.totalUnits += event.getUnits();
        this.totalScore += score;

        int token = event.getId().length() * 7 + event.getAction().length() * 3 + score;
        this.checksum = (this.checksum * 31 + token) % 100000;

        String segment = event.getSegment();
        SegmentMetrics metrics = this.metricsBySegment.get(segment);
        if (metrics == null) {
            metrics = new SegmentMetrics(segment);
            this.metricsBySegment.put(segment, metrics);
            this.segmentOrder.add(segment);
        }
        metrics.accept(event, score);
    }

    public void recordRejected(int rowNumber, ParseResult<Event> parsed) {
        this.rejections.add(new Rejection(rowNumber, parsed.getCode(), parsed.getDetail()));
    }

    public int getSourceRows() {
        return this.sourceRows;
    }

    public int getAcceptedCount() {
        return this.events.size();
    }

    public int getRejectedCount() {
        return this.rejections.size();
    }

    public List<Rejection> getRejections() {
        return this.rejections;
    }

    public List<String> getSegmentOrder() {
        return this.segmentOrder;
    }

    public SegmentMetrics getMetrics(String segment) {
        return this.metricsBySegment.get(segment);
    }

    public int getTotalSuccesses() {
        return this.totalSuccesses;
    }

    public int getTotalFailures() {
        return getAcceptedCount() - this.totalSuccesses;
    }

    public int getAverageLatency() {
        if (getAcceptedCount() == 0) {
            return 0;
        }
        return this.totalLatency / getAcceptedCount();
    }

    public int getTotalUnits() {
        return this.totalUnits;
    }

    public int getTotalScore() {
        return this.totalScore;
    }

    public int getChecksum() {
        return this.checksum;
    }
}
