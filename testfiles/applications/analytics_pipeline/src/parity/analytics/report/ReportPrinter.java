package parity.analytics.report;

import java.util.ArrayList;
import java.util.List;

import parity.analytics.model.SegmentMetrics;
import parity.analytics.pipeline.AnalyticsResult;
import parity.analytics.pipeline.Rejection;

public class ReportPrinter {
    public void print(AnalyticsResult result) {
        System.out.println("ANALYTICS PIPELINE v1");
        System.out.println("SOURCE rows=" + result.getSourceRows());
        System.out.println("ACCEPTED count=" + result.getAcceptedCount() + " rejected=" + result.getRejectedCount());

        for (Rejection rejection : result.getRejections()) {
            System.out.println("REJECTION row=" + rejection.getRowNumber()
                    + " code=" + rejection.getCode()
                    + " detail=" + rejection.getDetail());
        }

        System.out.println("TOTAL events=" + result.getAcceptedCount()
                + " successes=" + result.getTotalSuccesses()
                + " failures=" + result.getTotalFailures()
                + " units=" + result.getTotalUnits()
                + " avgLatency=" + result.getAverageLatency()
                + " score=" + result.getTotalScore());

        List<RankedSegment> candidates = new ArrayList<RankedSegment>();
        for (String segment : result.getSegmentOrder()) {
            candidates.add(new RankedSegment(result.getMetrics(segment)));
        }
        StableRanker<RankedSegment> ranker = new StableRanker<RankedSegment>();
        List<RankedSegment> ranked = ranker.sort(candidates);

        int rank = 1;
        for (RankedSegment entry : ranked) {
            SegmentMetrics metrics = entry.getMetrics();
            System.out.println("RANK " + rank
                    + " segment=" + metrics.getSegment()
                    + " events=" + metrics.getEventCount()
                    + " successRate=" + metrics.getSuccessRate()
                    + " avgLatency=" + metrics.getAverageLatency()
                    + " range=" + metrics.getMinimumLatency() + ".." + metrics.getMaximumLatency()
                    + " units=" + metrics.getTotalUnits()
                    + " score=" + metrics.getTotalScore()
                    + " index=" + entry.getIndex());
            rank++;
        }

        SegmentMetrics hotspot = findLatencyHotspot(result);
        System.out.println("HOTSPOT segment=" + hotspot.getSegment() + " spread=" + hotspot.getLatencySpread());
        System.out.println("CHECKSUM value=" + result.getChecksum());
        System.out.println("END analytics-v1");
    }

    private SegmentMetrics findLatencyHotspot(AnalyticsResult result) {
        SegmentMetrics hotspot = null;
        for (String segment : result.getSegmentOrder()) {
            SegmentMetrics candidate = result.getMetrics(segment);
            if (hotspot == null
                    || candidate.getLatencySpread() > hotspot.getLatencySpread()
                    || (candidate.getLatencySpread() == hotspot.getLatencySpread()
                    && candidate.getSegment().compareTo(hotspot.getSegment()) < 0)) {
                hotspot = candidate;
            }
        }
        return hotspot;
    }
}
