package parity.analytics.pipeline;

import parity.analytics.common.ParseResult;
import parity.analytics.model.Event;
import parity.analytics.parse.RecordParser;

public class AnalyticsEngine {
    private RecordParser<Event> parser;
    private ScorePolicy scorePolicy;

    public AnalyticsEngine(RecordParser<Event> parser, ScorePolicy scorePolicy) {
        this.parser = parser;
        this.scorePolicy = scorePolicy;
    }

    public AnalyticsResult run(String[] lines) {
        AnalyticsResult result = new AnalyticsResult(lines.length);
        for (int i = 0; i < lines.length; i++) {
            ParseResult<Event> parsed = this.parser.parse(lines[i]);
            if (parsed.isAccepted()) {
                Event event = parsed.getValue();
                int score = this.scorePolicy.score(event);
                result.recordAccepted(event, score);
            } else {
                result.recordRejected(i + 1, parsed);
            }
        }
        return result;
    }
}
