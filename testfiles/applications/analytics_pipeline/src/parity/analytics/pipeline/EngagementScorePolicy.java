package parity.analytics.pipeline;

import parity.analytics.model.Event;

public class EngagementScorePolicy extends ScorePolicy {
    public EngagementScorePolicy() {
        super(-20);
    }

    public int score(Event event) {
        int actionScore = baseFor(event.getAction());
        int unitScore = event.getUnits() * 2;
        int outcomeScore = event.isSuccessful() ? 5 : -8;
        int latencyPenalty = Math.min(10, event.getLatencyMs() / 100);
        return clamp(actionScore + unitScore + outcomeScore - latencyPenalty);
    }

    private int baseFor(String action) {
        switch (action) {
            case "VIEW":
                return 3;
            case "SEARCH":
                return 5;
            case "CART":
                return 9;
            case "PURCHASE":
                return 20;
            case "REFUND":
                return -10;
            case "SUPPORT":
                return 6;
            default:
                return 0;
        }
    }
}
