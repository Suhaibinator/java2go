package parity.analytics.app;

import parity.analytics.parse.EventParser;
import parity.analytics.pipeline.AnalyticsEngine;
import parity.analytics.pipeline.AnalyticsResult;
import parity.analytics.pipeline.EngagementScorePolicy;
import parity.analytics.report.ReportPrinter;

public class AnalyticsPipelineApp {
    public static void main(String[] args) {
        AnalyticsEngine engine = new AnalyticsEngine(
                new EventParser(),
                new EngagementScorePolicy());
        AnalyticsResult result = engine.run(SampleData.lines());
        ReportPrinter printer = new ReportPrinter();
        printer.print(result);
    }
}
